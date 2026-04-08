package controller

import (
	crd "github.com/emqx/emqx-operator/api/v3alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reconciler syncCoreSet", Ordered, func() {
	var ns *corev1.Namespace = &corev1.Namespace{}
	var instance *crd.EMQX

	var round *reconcileRound
	var coreSet *appsv1.StatefulSet
	var pod0, pod1 *corev1.Pod

	const (
		currentRevision string = "current"
		updateRevision  string = "update"
	)

	BeforeAll(func() {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "controller-sync-core-set-test",
				Labels: map[string]string{"test": "e2e"},
			},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})

	BeforeEach(func() {
		coreLabels := emqx.DefaultLabelsWith(crd.CoreLabels())
		coreSet = &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      emqx.Name + "-core",
				Namespace: ns.Name,
				Labels:    coreLabels,
			},
			Spec: appsv1.StatefulSetSpec{
				ServiceName: emqx.Name + "-core",
				Replicas:    ptr.To(int32(2)),
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
		Expect(k8sClient.Create(ctx, coreSet)).Should(Succeed())

		pod0 = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      coreSet.Name + "-0",
				Namespace: ns.Name,
				Labels: map[string]string{
					crd.LabelInstance:                     "emqx",
					crd.LabelManagedBy:                    "emqx-operator",
					crd.LabelDBRole:                       "core",
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
		pod1 = pod0.DeepCopy()
		pod1.ObjectMeta.Name = coreSet.Name + "-1"
		Expect(k8sClient.Create(ctx, pod0)).Should(Succeed())
		Expect(k8sClient.Create(ctx, pod1)).Should(Succeed())
		pod0.Status.Conditions = []corev1.PodCondition{
			{Type: corev1.PodReady, Status: corev1.ConditionTrue, LastTransitionTime: metav1.Now()},
		}
		pod1.Status.Conditions = pod0.Status.Conditions
		Expect(k8sClient.Status().Update(ctx, pod0)).Should(Succeed())
		Expect(k8sClient.Status().Update(ctx, pod1)).Should(Succeed())

		coreSet.Status.Replicas = 2
		coreSet.Status.ReadyReplicas = 2
		coreSet.Status.AvailableReplicas = 2
		coreSet.Status.CurrentRevision = currentRevision
		coreSet.Status.UpdateRevision = updateRevision
		Expect(k8sClient.Status().Update(ctx, coreSet)).Should(Succeed())

		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(2))
		instance.Status.CoreNodes = []crd.EMQXNode{
			{Name: "emqx@" + pod0.Name, PodName: pod0.Name, Status: "running"},
			{Name: "emqx@" + pod1.Name, PodName: pod1.Name, Status: "running"},
		}

		round = newReconcileRound()
		Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
	})

	AfterEach(func() {
		_ = k8sClient.Delete(ctx, pod0)
		_ = k8sClient.Delete(ctx, pod1)
		Expect(k8sClient.Delete(ctx, coreSet)).Should(Succeed())
	})

	It("waits when fewer than N-1 other core pods are available (ready)", func() {
		// Removing pod1 requires at least one *other* available core; pod0 is not Ready.
		pod0.Status.Conditions = []corev1.PodCondition{}
		Expect(k8sClient.Status().Update(ctx, pod0)).Should(Succeed())
		coreSet.Status.AvailableReplicas = 1
		Expect(k8sClient.Status().Update(ctx, coreSet)).Should(Succeed())
		Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
		admission := checkCorePodRemoval(round, instance, pod1, false)
		Expect(admission).Should(And(
			HaveField("Action", Equal(admissionWait)),
			HaveField("Reason", Equal("cores are not available yet")),
		))
	})

	It("waits while a node evacuation is still in progress", func() {
		instance.Status.NodeEvacuations = []crd.NodeEvacuationStatus{
			{NodeName: "emqx@" + pod1.Name, State: "evicting_sessions"},
		}
		admission := checkCorePodRemoval(round, instance, pod1, false)
		Expect(admission).Should(And(
			HaveField("Action", Equal(admissionWait)),
			HaveField("Reason", Equal("node evacuation is still in progress")),
		))
	})

	It("node session > 0", func() {
		instance.Status.CoreNodes[1].Sessions = 99999
		admission := checkCorePodRemoval(round, instance, pod1, false)
		Expect(admission).Should(And(
			HaveField("Action", Equal(admissionEvacuate)),
			HaveField("Reason", ContainSubstring("active sessions")),
		))
	})

	It("single node session > 0", func() {
		instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(1))
		instance.Status.CoreNodes = []crd.EMQXNode{
			{Name: "emqx@" + pod0.Name, PodName: pod0.Name, Status: "running", Sessions: 99999},
		}
		admission := checkCorePodRemoval(round, instance, pod0, false)
		Expect(admission).Should(And(
			HaveField("Action", Equal(admissionRemove)),
			HaveField("Reason", ContainSubstring("nowhere")),
		))
	})

	It("node session is 0", func() {
		instance.Status.CoreNodes[1].Sessions = 0
		admission := checkCorePodRemoval(round, instance, pod1, false)
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
			s := &syncCoreSet{emqxReconciler}
			result := s.rollingUpdate(round, instance)
			Expect(result).To(Equal(subResult{}))
			_, err := actualObject(pod1)
			Expect(k8sErrors.IsNotFound(err)).To(BeTrue(), "highest-ordinal outdated pod should be deleted first")
		})

		It("should block rolling update with 1 old core", func() {
			// Simulate `pod1` already on new revision, while `pod0` is outdated.
			pod1.Labels[appsv1.ControllerRevisionHashLabelKey] = coreSet.Status.UpdateRevision
			Expect(k8sClient.Update(ctx, pod1)).Should(Succeed())
			Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
			s := &syncCoreSet{emqxReconciler}
			result := s.rollingUpdate(round, instance)
			Expect(result).To(Equal(subResult{}))
			Expect(actualObject(pod0)).To(
				Not(BeNil()),
				"sole remaining outdated core must stay up while replicant ReplicaSet migrates",
			)
		})

	})

	It("DS replication site blocks scale-down", func() {
		pod1.Status.Conditions = []corev1.PodCondition{
			{Type: crd.DSReplicationSite, Status: corev1.ConditionTrue},
		}
		admission := checkCorePodRemoval(round, instance, pod1, true)
		Expect(admission).To(And(
			HaveField("Action", Equal(admissionWait)),
			HaveField("Reason", ContainSubstring("DS replication site")),
		))
	})

	It("DS replication site does not block rolling update", func() {
		pod1.Status.Conditions = []corev1.PodCondition{
			{Type: crd.DSReplicationSite, Status: corev1.ConditionTrue},
		}
		admission := checkCorePodRemoval(round, instance, pod1, false)
		Expect(admission).To(
			HaveField("Action", Equal(admissionRemove)),
		)
	})
})
