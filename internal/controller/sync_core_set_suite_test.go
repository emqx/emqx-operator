package controller

import (
	"fmt"
	"time"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = DescribeClientFaultMatrix("Reconciler syncCoreSet", Ordered, func() {
	var ns *corev1.Namespace
	var instance *crd.EMQX

	var round *reconcileRound
	var coreSet *appsv1.StatefulSet
	var pods []*corev1.Pod

	const (
		currentRevision string = "current"
		updateRevision  string = "update"
	)

	BeforeAll(func() {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "controller-sync-core-set-test",
				Labels:       map[string]string{"test": "e2e"},
			},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})

	mkCoreSet := func(replicas int32) *appsv1.StatefulSet {
		coreLabels := emqx.DefaultLabelsWith(crd.CoreLabels())
		return &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:        emqx.Name + "-core",
				Namespace:   ns.Name,
				Labels:      coreLabels,
				Annotations: newCoreSetRetirementState(replicas).annotations(),
			},
			Spec: appsv1.StatefulSetSpec{
				ServiceName: emqx.Name + "-core",
				Replicas:    ptr.To(replicas),
				UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
					Type: appsv1.OnDeleteStatefulSetStrategyType,
				},
				Selector: &metav1.LabelSelector{
					MatchLabels: coreLabels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: coreLabels,
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "emqx", Image: "emqx"},
						},
					},
				},
			},
		}
	}

	mkPod := func(ordinal int) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%d", coreSet.Name, ordinal),
				Namespace: ns.Name,
				Labels: map[string]string{
					crd.LabelInstance:                     "emqx",
					crd.LabelManagedBy:                    "emqx-operator",
					crd.LabelMriaRole:                     "core",
					appsv1.ControllerRevisionHashLabelKey: currentRevision,
				},
				OwnerReferences: ownerReferences(coreSet),
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "emqx", Image: "emqx"},
				},
			},
		}
	}

	BeforeEach(func() {
		coreSet = mkCoreSet(3)
		Expect(k8sClient.Create(ctx, coreSet)).Should(Succeed())
		coreSet.Status.Replicas = 3
		coreSet.Status.ReadyReplicas = 3
		coreSet.Status.AvailableReplicas = 3
		coreSet.Status.CurrentRevision = currentRevision
		coreSet.Status.UpdateRevision = updateRevision
		Expect(k8sClient.Status().Update(ctx, coreSet)).Should(Succeed())

		pods = []*corev1.Pod{}
		nodes := []crd.EMQXNode{}
		for i := range 3 {
			pod := mkPod(i)
			Expect(k8sClient.Create(ctx, pod)).Should(Succeed())
			pod.Status.Conditions = []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue, LastTransitionTime: metav1.Now()},
			}
			Expect(k8sClient.Status().Update(ctx, pod)).Should(Succeed())
			pods = append(pods, pod)
			nodes = append(nodes, crd.EMQXNode{Name: "emqx@" + pod.Name, PodName: pod.Name, Status: "running"})
		}

		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(3))
		instance.Status.CoreNodes = nodes

		round = newReconcileRound()
		Expect(ensureReconcileState(round, instance)).To(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, coreSet)).Should(Succeed())
		for _, pod := range pods {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, pod))).Should(Succeed())
		}
	})

	It("waits when fewer than N-1 other core pods are available (ready)", func() {
		// Removing pod1 requires at least one *other* available core; pod0 is not Ready.
		pods[0].Status.Conditions = []corev1.PodCondition{}
		Expect(k8sClient.Status().Update(ctx, pods[0])).Should(Succeed())
		coreSet.Status.AvailableReplicas = 2
		Expect(k8sClient.Status().Update(ctx, coreSet)).Should(Succeed())
		Expect(ensureReconcileState(round, instance)).To(Succeed())
		admission := checkCorePodRemoval(round, instance, pods[2], coreRollingUpdate)
		Expect(admission).Should(And(
			HaveField("Action", Equal(admissionWait)),
			HaveField("Reason", Equal("cores are not available yet")),
		))
	})

	It("waits while a node evacuation is in progress", func() {
		instance.Status.NodeEvacuations = []crd.NodeEvacuationStatus{
			{NodeName: "emqx@" + pods[2].Name, State: "evicting_sessions"},
		}
		admission := checkCorePodRemoval(round, instance, pods[2], coreRollingUpdate)
		Expect(admission).Should(And(
			HaveField("Action", Equal(admissionWait)),
			HaveField("Reason", Equal("node evacuation in progress")),
		))
	})

	It("node session > 0", func() {
		instance.Status.CoreNodes[2].Sessions = 99999
		admission := checkCorePodRemoval(round, instance, pods[2], coreRollingUpdate)
		Expect(admission).Should(And(
			HaveField("Action", Equal(admissionEvacuate)),
			HaveField("Reason", ContainSubstring("active sessions")),
		))
	})

	It("single node session > 0", func() {
		instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(1))
		instance.Status.CoreNodes = []crd.EMQXNode{
			{Name: "emqx@" + pods[0].Name, PodName: pods[0].Name, Status: "running", Sessions: 99999},
		}
		admission := checkCorePodRemoval(round, instance, pods[0], coreRollingUpdate)
		Expect(admission).Should(And(
			HaveField("Action", Equal(admissionRemove)),
			HaveField("Reason", ContainSubstring("nowhere")),
		))
	})

	It("node session > 0 & node evacuation is disabled", func() {
		instance.Spec.UpdateStrategy.EvacuationStrategy.Type = crd.DisabledEvacuationStrategy
		instance.Status.CoreNodes[2].Sessions = 99999
		admission := checkCorePodRemoval(round, instance, pods[2], coreRollingUpdate)
		Expect(admission).Should(And(
			HaveField("Action", Equal(admissionRemove)),
		))
	})

	It("node session is 0", func() {
		instance.Status.CoreNodes[2].Sessions = 0
		admission := checkCorePodRemoval(round, instance, pods[2], coreRollingUpdate)
		Expect(admission).To(
			HaveField("Action", Equal(admissionRemove)),
		)
	})

	When("replicant replicaSet updating", func() {
		BeforeEach(func() {
			instance.Spec.ReplicantTemplate = &crd.EMQXReplicantTemplate{
				Spec: crd.EMQXReplicantTemplateSpec{
					Replicas: ptr.To(int32(3)),
				},
			}
			instance.Status.ReplicantNodesStatus = crd.ReplicantNodesStatus{
				UpdateRevision:  updateRevision,
				CurrentRevision: currentRevision,
			}
		})

		It("should allow rolling update with multiple old cores", func() {
			s := &syncCoreSet{emqxReconciler()}
			Eventually(s.rollingUpdate).WithArguments(round, instance).
				Should(BeSuccessfulReconcile())
			_, err := actualObject(pods[2])
			Expect(k8sErrors.IsNotFound(err)).To(BeTrue(), "highest-ordinal outdated pod should be deleted first")
		})

		It("should block rolling update with 1 old core", func() {
			// Simulate `pod1` already on new revision, while `pod0` is outdated.
			pods[1].Labels[appsv1.ControllerRevisionHashLabelKey] = coreSet.Status.UpdateRevision
			pods[2].Labels[appsv1.ControllerRevisionHashLabelKey] = coreSet.Status.UpdateRevision
			Expect(k8sClient.Update(ctx, pods[1])).Should(Succeed())
			Expect(k8sClient.Update(ctx, pods[2])).Should(Succeed())
			Expect(ensureReconcileState(round, instance)).To(Succeed())
			s := &syncCoreSet{emqxReconciler()}
			Eventually(s.rollingUpdate).WithArguments(round, instance).
				Should(BeSuccessfulReconcile())
			Expect(actualObject(pods[0])).To(
				Not(BeNil()),
				"sole remaining outdated core must stay up while replicant ReplicaSet migrates",
			)
		})

	})

	It("DS replication site blocks scale-down", func() {
		pods[2].Status.Conditions = []corev1.PodCondition{
			{Type: crd.DSReplicationSite, Status: corev1.ConditionTrue},
		}
		admission := checkCorePodRemoval(round, instance, pods[2], coreScaleDown)
		Expect(admission).To(And(
			HaveField("Action", Equal(admissionWait)),
			HaveField("Reason", ContainSubstring("DS replication site")),
		))
	})

	It("DS replication site does not block rolling update", func() {
		pods[2].Status.Conditions = []corev1.PodCondition{
			{Type: crd.DSReplicationSite, Status: corev1.ConditionTrue},
		}
		admission := checkCorePodRemoval(round, instance, pods[2], coreRollingUpdate)
		Expect(admission).To(
			HaveField("Action", Equal(admissionRemove)),
		)
	})

	It("blocks reuse until the watermark is forcibly lowered", func() {
		// Preconditions:
		// - coreSet was scaled down 5 → 3 replicas
		// - coreSet has 3 live replicas, with 1 replica `emqx-core-3` still retiring
		// - watermark is still set to 5, blocking scale-ups
		instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(5))
		coreSet.Annotations = newCoreSetRetirementState(5).annotations()
		Expect(k8sClient.Update(ctx, coreSet)).To(Succeed())
		pod3 := mkPod(3)
		Expect(k8sClient.Create(ctx, pod3)).To(Succeed())

		round := newReconcileRound()
		reconciler := &syncCoreSet{emqxReconciler()}
		Eventually(runRoundReconcile).WithArguments(round, instance, reconciler).
			Should(BeSuccessfulReconcile())

		// Conditions:
		// - `emqx-core-3` is deleted: it's below watermark and should be retiring
		// - scaling up is blocked
		Expect(actualize(pod3)).To(WithTransform(k8sErrors.IsNotFound, BeTrue()))
		Expect(actualize(coreSet)).To(Succeed())
		Expect(coreSet).To(HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(3))))

		coreSet.Annotations = newCoreSetRetirementState(3).annotations()
		Expect(k8sClient.Update(ctx, coreSet)).To(Succeed())
		Eventually(runRoundReconcile).WithArguments(round, instance, reconciler).
			Should(BeSuccessfulReconcile())

		// Postconditions:
		// - coreSet scaled up
		// - watermark is updated to reflect current number of replicas
		Expect(actualize(coreSet)).To(Succeed())
		Expect(coreSet).To(HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(5))))
		state, err := readCoreSetRetirementState(coreSet)
		Expect(err).NotTo(HaveOccurred())
		Expect(state.watermark).To(BeEquivalentTo(5))
	})

	It("starts one ordinal retirement", func() {
		// Preconditions:
		// - coreSet is scaling down 3 → 1 replicas
		// - watermark is at 3
		// - coreSet retirement state was last updated an hour ago
		instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(1))
		retirement := &coreSetRetirement{3, time.Now().Add(-time.Hour)}
		coreSet.Annotations = retirement.annotations()
		Expect(k8sClient.Update(ctx, coreSet)).To(Succeed())

		round := newReconcileRound()
		reconciler := &syncCoreSet{emqxReconciler()}
		Eventually(runRoundReconcile).WithArguments(round, instance, reconciler).
			Should(BeSuccessfulReconcile())

		// Postconditions:
		// - only one replica is removed per call
		// - watermark remains at 3, preventing ordinal 2 from being reused
		// - the timestamp is refreshed because a new retirement epoch began
		Expect(actualize(coreSet)).To(Succeed())
		Expect(coreSet).To(HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(2))))
		state, err := readCoreSetRetirementState(coreSet)
		Expect(err).NotTo(HaveOccurred())
		Expect(state.watermark).To(BeEquivalentTo(3))
		Expect(state.updatedAt).To(BeTemporally("~", time.Now(), time.Second*5))
	})

	It("continues scale-down while earlier ordinals retire", func() {
		// Preconditions:
		// - coreSet is scaling down 3 → 1 replicas, currently has 2 replicas
		// - watermark is still at 3
		// - coreSet retirement state was last updated a minute ago
		instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(1))
		startedAt := time.Now().Add(-time.Minute)
		retirement := &coreSetRetirement{3, startedAt}
		coreSet.Spec.Replicas = ptr.To(int32(2))
		coreSet.Annotations = retirement.annotations()
		Expect(k8sClient.Update(ctx, coreSet)).To(Succeed())

		round := newReconcileRound()
		reconciler := &syncCoreSet{emqxReconciler()}
		Eventually(runRoundReconcile).WithArguments(round, instance, reconciler).
			Should(BeSuccessfulReconcile())
		Eventually(runRoundReconcile).WithArguments(round, instance, reconciler).
			Should(BeSuccessfulReconcile())

		// Postconditions:
		// - coreSet scaling down succeeded, 1 replica left
		// - watermark is still at 3
		// - coreSet retirement update timestamp unchanged
		Expect(actualize(coreSet)).To(Succeed())
		Expect(coreSet).To(HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(1))))
		Expect(actualize(pods[2])).To(WithTransform(k8sErrors.IsNotFound, BeTrue()))
		Expect(actualize(pods[1])).To(WithTransform(k8sErrors.IsNotFound, BeTrue()))
		state, err := readCoreSetRetirementState(coreSet)
		Expect(err).NotTo(HaveOccurred())
		Expect(state.watermark).To(BeEquivalentTo(3))
		Expect(state.updatedAt).To(BeTemporally("~", startedAt.UTC(), time.Nanosecond))
	})

})
