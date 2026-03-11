package controller

import (
	"time"

	crdv2 "github.com/emqx/emqx-operator/api/v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const currentRevision string = "current"
const updateRevision string = "update"

func actualize(instance client.Object) (client.Object, error) {
	err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
	return instance, err
}

var _ = Describe("Reconciler syncPods", Ordered, func() {
	var ns *corev1.Namespace = &corev1.Namespace{}
	var instance *crdv2.EMQX

	var sr *syncReplicantSets
	var sc *syncCoreSet
	var round *reconcileRound

	// Single core StatefulSet with two revisions tracked by pods.
	var coreSet *appsv1.StatefulSet
	var updateReplicantSet, currentReplicantSet *appsv1.ReplicaSet
	var currentReplicantPod *corev1.Pod

	BeforeAll(func() {
		// Create namespace:
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-sync-pods-suite-test",
				Labels: map[string]string{
					"test": "e2e",
				},
			},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
		// Set up single core StatefulSet (no hash in labels/selector):
		instance = emqx.DeepCopy()
		coreLabels := instance.DefaultLabelsWith(crdv2.CoreLabels())
		coreSet = &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instance.CoreNamespacedName().Name,
				Namespace: ns.Name,
				Labels:    coreLabels,
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas:    ptr.To(int32(1)),
				ServiceName: instance.HeadlessServiceNamespacedName().Name,
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

		// Set up "update" replicantSet:
		updateReplicantLabels := instance.DefaultLabelsWith(
			crdv2.ReplicantLabels(),
			map[string]string{crdv2.LabelPodTemplateHash: updateRevision},
		)
		updateReplicantSet = &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: instance.Name + "-",
				Namespace:    ns.Name,
				Labels:       updateReplicantLabels,
			},
			Spec: appsv1.ReplicaSetSpec{
				Replicas: ptr.To(int32(1)),
				Selector: &metav1.LabelSelector{
					MatchLabels: updateReplicantLabels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: updateReplicantLabels,
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "emqx", Image: "emqx"},
						},
					},
				},
			},
		}
		// Set up "current" replicantSet:
		currentReplicantSet = updateReplicantSet.DeepCopy()
		currentReplicantSet.Labels[crdv2.LabelPodTemplateHash] = currentRevision
		currentReplicantSet.Spec.Selector.MatchLabels[crdv2.LabelPodTemplateHash] = currentRevision
		currentReplicantSet.Spec.Template.Labels[crdv2.LabelPodTemplateHash] = currentRevision
		// Create resources:
		Expect(k8sClient.Create(ctx, coreSet)).Should(Succeed())
		Expect(k8sClient.Create(ctx, updateReplicantSet)).Should(Succeed())
		Expect(k8sClient.Create(ctx, currentReplicantSet)).Should(Succeed())
		// Create "current" replicantSet pod:
		currentReplicantPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: currentReplicantSet.Name + "-",
				Namespace:    currentReplicantSet.Namespace,
				Labels:       currentReplicantSet.Spec.Template.Labels,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "apps/v1",
						Kind:       "ReplicaSet",
						Name:       currentReplicantSet.Name,
						UID:        currentReplicantSet.UID,
						Controller: ptr.To(true),
					},
				},
			},
			Spec: currentReplicantSet.Spec.Template.Spec,
		}
		Expect(k8sClient.Create(ctx, currentReplicantPod)).Should(Succeed())
		// Mock resource status:
		coreSet.Status.Replicas = 1
		coreSet.Status.ReadyReplicas = 1
		coreSet.Status.UpdateRevision = updateRevision
		coreSet.Status.CurrentRevision = updateRevision
		updateReplicantSet.Status.Replicas = 1
		updateReplicantSet.Status.ReadyReplicas = 1
		currentReplicantSet.Status.Replicas = 1
		currentReplicantSet.Status.ReadyReplicas = 1
		Expect(k8sClient.Status().Update(ctx, coreSet)).Should(Succeed())
		Expect(k8sClient.Status().Update(ctx, updateReplicantSet)).Should(Succeed())
		Expect(k8sClient.Status().Update(ctx, currentReplicantSet)).Should(Succeed())
	})

	AfterAll(func() {
		Expect(k8sClient.DeleteAllOf(ctx, &corev1.Pod{}, client.InNamespace(ns.Name))).Should(Succeed())
		Expect(k8sClient.DeleteAllOf(ctx, &appsv1.ReplicaSet{}, client.InNamespace(ns.Name))).Should(Succeed())
		Expect(k8sClient.DeleteAllOf(ctx, &appsv1.StatefulSet{}, client.InNamespace(ns.Name))).Should(Succeed())
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})

	BeforeEach(func() {
		// Mock instance state:
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.ReplicantTemplate = &crdv2.EMQXReplicantTemplate{
			Spec: crdv2.EMQXReplicantTemplateSpec{
				Replicas: ptr.To(int32(1)),
			},
		}
		instance.Status = crdv2.EMQXStatus{
			Conditions: []metav1.Condition{
				{
					Type:               crdv2.Available,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
				},
			},
			CoreNodesStatus: crdv2.CoreNodesStatus{
				ReadyReplicas: 1,
			},
			CoreNodes: []crdv2.EMQXNode{},
			ReplicantNodesStatus: crdv2.ReplicantNodesStatus{
				UpdateRevision:  updateRevision,
				UpdateReplicas:  1,
				CurrentRevision: currentRevision,
				CurrentReplicas: 1,
				ReadyReplicas:   2,
			},
			ReplicantNodes: []crdv2.EMQXNode{
				{Name: "emqx@10.0.0.1", PodName: currentReplicantPod.Name, Status: "running"},
			},
		}
		// Instantiate reconciler:
		sc = &syncCoreSet{emqxReconciler}
		sr = &syncReplicantSets{emqxReconciler}
		round = newReconcileRound()
		round.state = loadReconcileState(ctx, k8sClient, instance)
	})

	It("running update emqx node controller", func() {
		Eventually(func() *crdv2.EMQX {
			_ = sc.reconcile(round, instance)
			_ = sr.reconcile(round, instance)
			return instance
		}).WithTimeout(timeout).WithPolling(interval).Should(And(
			// should add pod deletion cost
			WithTransform(
				func(*crdv2.EMQX) (client.Object, error) { return actualize(currentReplicantPod) },
				HaveField("Annotations", HaveKeyWithValue("controller.kubernetes.io/pod-deletion-cost", "-99999")),
			),
			// should scale down rs
			WithTransform(
				func(*crdv2.EMQX) (client.Object, error) { return actualize(currentReplicantSet) },
				HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(0))),
			),
		))
	})

})

var _ = Describe("Reconciler syncCoreSet", Ordered, func() {
	var ns *corev1.Namespace = &corev1.Namespace{}
	var instance *crdv2.EMQX

	var s *syncCoreSet
	var round *reconcileRound
	var coreSet *appsv1.StatefulSet
	var currentPod *corev1.Pod

	BeforeAll(func() {
		// Create namespace:
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "controller-sync-core-set-test",
				Labels: map[string]string{"test": "e2e"},
			},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
		// Create single core StatefulSet:
		instance = emqx.DeepCopy()
		coreLabels := instance.DefaultLabelsWith(crdv2.CoreLabels())
		coreSet = &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instance.Name + "-core",
				Namespace: ns.Name,
				Labels:    coreLabels,
			},
			Spec: appsv1.StatefulSetSpec{
				ServiceName: instance.Name + "-core",
				Replicas:    ptr.To(int32(1)),
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

		// Create core pod with an outdated revision label:
		currentPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      coreSet.Name + "-0",
				Namespace: ns.Name,
				Labels: map[string]string{
					crdv2.LabelInstance:                   "emqx",
					crdv2.LabelManagedBy:                  "emqx-operator",
					crdv2.LabelDBRole:                     "core",
					appsv1.ControllerRevisionHashLabelKey: "old-revision",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "apps/v1",
						Kind:       "StatefulSet",
						Name:       coreSet.Name,
						UID:        coreSet.UID,
						Controller: ptr.To(true),
					},
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "emqx", Image: "emqx"},
				},
			},
		}
		Expect(k8sClient.Create(ctx, currentPod)).Should(Succeed())

		// Set StatefulSet status with update revision different from pod's revision.
		coreSet.Status.Replicas = 1
		coreSet.Status.ReadyReplicas = 1
		coreSet.Status.UpdateRevision = "new-revision"
		coreSet.Status.CurrentRevision = "old-revision"
		Expect(k8sClient.Status().Update(ctx, coreSet)).Should(Succeed())
	})

	AfterAll(func() {
		Expect(k8sClient.DeleteAllOf(ctx, &corev1.Pod{}, client.InNamespace(ns.Name))).Should(Succeed())
		Expect(k8sClient.DeleteAllOf(ctx, &appsv1.StatefulSet{}, client.InNamespace(ns.Name))).Should(Succeed())
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})

	BeforeEach(func() {
		// Mock instance state:
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Status.Conditions = []metav1.Condition{
			{
				Type:               crdv2.Available,
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
			},
		}
		instance.Status.CoreNodes = []crdv2.EMQXNode{
			{Name: "emqx@" + currentPod.Name, PodName: currentPod.Name, Status: "running"},
		}
		// Instantiate reconciler:
		s = &syncCoreSet{emqxReconciler}
		round = newReconcileRound()
		round.state = loadReconcileState(ctx, k8sClient, instance)
	})

	It("emqx is not available", func() {
		instance.Status.Conditions = []metav1.Condition{}
		admission := checkCorePodRemoval(instance, currentPod, false)
		Expect(admission).Should(And(
			HaveField("Action", Equal(admissionWait)),
			HaveField("Reason", ContainSubstring("not ready")),
		))
	})

	It("emqx is available / initial delay has not passed", func() {
		instance.Spec.UpdateStrategy.InitialDelaySeconds = 99999999
		admission := checkCorePodRemoval(instance, currentPod, false)
		Expect(admission).Should(And(
			HaveField("Action", Equal(admissionWait)),
			HaveField("Reason", ContainSubstring("not ready")),
		))
	})

	It("replicaSet is not ready", func() {
		instance.Spec.ReplicantTemplate = &crdv2.EMQXReplicantTemplate{
			Spec: crdv2.EMQXReplicantTemplateSpec{
				Replicas: ptr.To(int32(3)),
			},
		}
		instance.Status.ReplicantNodesStatus = crdv2.ReplicantNodesStatus{
			UpdateRevision:  updateRevision,
			CurrentRevision: currentRevision,
		}
		admission := checkCorePodRemoval(instance, currentPod, false)
		Expect(admission).Should(And(
			HaveField("Action", Equal(admissionWait)),
			HaveField("Reason", ContainSubstring("replicaSet")),
		))
		Eventually(s.reconcile).WithArguments(newReconcileRound(), instance).
			WithTimeout(timeout).
			WithPolling(interval).
			Should(Equal(subResult{}))
	})

	It("node session > 0", func() {
		instance.Status.CoreNodes[0].Sessions = 99999
		admission := checkCorePodRemoval(instance, currentPod, false)
		Expect(admission).Should(And(
			HaveField("Action", Equal(admissionEvacuate)),
			HaveField("Reason", ContainSubstring("active sessions")),
		))
	})

	It("node session is 0", func() {
		instance.Status.CoreNodes[0].Sessions = 0
		admission := checkCorePodRemoval(instance, currentPod, false)
		Expect(admission).Should(And(
			HaveField("Action", Equal(admissionRemove)),
			HaveField("Reason", BeEmpty()),
		))
	})

	It("DS replication site blocks scale-down", func() {
		instance.Status.CoreNodes[0].Sessions = 0
		currentPod.Status.Conditions = append(currentPod.Status.Conditions, corev1.PodCondition{
			Type:   crdv2.DSReplicationSite,
			Status: corev1.ConditionTrue,
		})
		admission := checkCorePodRemoval(instance, currentPod, true)
		Expect(admission).Should(And(
			HaveField("Action", Equal(admissionWait)),
			HaveField("Reason", ContainSubstring("DS replication site")),
		))
	})

	It("DS replication site does not block rolling update", func() {
		instance.Status.CoreNodes[0].Sessions = 0
		currentPod.Status.Conditions = append(currentPod.Status.Conditions, corev1.PodCondition{
			Type:   crdv2.DSReplicationSite,
			Status: corev1.ConditionTrue,
		})
		admission := checkCorePodRemoval(instance, currentPod, false)
		Expect(admission).Should(And(
			HaveField("Action", Equal(admissionRemove)),
		))
	})
})

var _ = Describe("Reconciler syncReplicantSets", Ordered, func() {
	var ns *corev1.Namespace = &corev1.Namespace{}
	var instance *crdv2.EMQX

	var s *syncReplicantSets
	var round *reconcileRound
	var current *appsv1.ReplicaSet
	var currentPod *corev1.Pod

	BeforeAll(func() {
		// Create namespace:
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-sync-replicant-sets-test",
				Labels: map[string]string{
					"test": "e2e",
				},
			},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
		// Create "current" replicaSet:
		instance = emqx.DeepCopy()
		current = &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: instance.Name + "-",
				Namespace:    ns.Name,
				Labels:       instance.DefaultLabelsWith(crdv2.ReplicantLabels()),
			},
			Spec: appsv1.ReplicaSetSpec{
				Replicas: ptr.To(int32(1)),
				Selector: &metav1.LabelSelector{
					MatchLabels: instance.DefaultLabelsWith(crdv2.ReplicantLabels()),
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: instance.DefaultLabelsWith(crdv2.ReplicantLabels()),
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "emqx", Image: "emqx"},
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, current)).Should(Succeed())
		// Create "current" replicaSet pod:
		currentPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: current.Name + "-",
				Namespace:    ns.Name,
				Labels:       current.Spec.Selector.MatchLabels,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "apps/v1",
						Kind:       "ReplicaSet",
						Name:       current.Name,
						UID:        current.UID,
						Controller: ptr.To(true),
					},
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "emqx", Image: "emqx"},
				},
			},
		}
		Expect(k8sClient.Create(ctx, currentPod)).Should(Succeed())
	})

	AfterAll(func() {
		Expect(k8sClient.DeleteAllOf(ctx, &corev1.Pod{}, client.InNamespace(ns.Name))).Should(Succeed())
		Expect(k8sClient.DeleteAllOf(ctx, &appsv1.ReplicaSet{}, client.InNamespace(ns.Name))).Should(Succeed())
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})

	BeforeEach(func() {
		// Mock instance state:
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Status.ReplicantNodesStatus.CurrentRevision = "fake"
		instance.Status.Conditions = []metav1.Condition{
			{
				Type:               crdv2.Available,
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
			},
		}
		instance.Status.ReplicantNodes = []crdv2.EMQXNode{
			{Name: "emqx@10.0.0.1", PodName: currentPod.Name, Status: "running"},
		}
		// Instantiate reconciler:
		s = &syncReplicantSets{emqxReconciler}
		round = newReconcileRound()
		round.state = loadReconcileState(ctx, k8sClient, instance)
	})

	It("emqx is not available", func() {
		instance.Status.Conditions = []metav1.Condition{}
		admission, err := s.chooseScaleDownReplicant(round, instance, current)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(admission).Should(And(
			HaveField("Reason", ContainSubstring("not ready")),
			HaveField("Pod", BeNil()),
		))
	})

	It("emqx is available / initial delay has not passed", func() {
		instance.Spec.UpdateStrategy.InitialDelaySeconds = 99999999
		admission, err := s.chooseScaleDownReplicant(round, instance, current)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(admission).Should(And(
			HaveField("Reason", ContainSubstring("not ready")),
			HaveField("Pod", BeNil()),
		))
	})

	It("emqx is in node evacuations", func() {
		instance.Status.NodeEvacuations = []crdv2.NodeEvacuationStatus{
			{State: "fake"},
		}
		admission, err := s.chooseScaleDownReplicant(round, instance, current)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(admission).Should(And(
			HaveField("Reason", ContainSubstring("evacuation")),
			HaveField("Pod", BeNil()),
		))
	})

	It("node session > 0", func() {
		instance.Status.ReplicantNodes[0].Sessions = 99999
		admission, err := s.chooseScaleDownReplicant(round, instance, current)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(admission).Should(And(
			HaveField("Reason", Not(BeEmpty())),
			HaveField("Pod", BeNil()),
		))
	})

	It("node session is 0", func() {
		instance.Status.ReplicantNodes[0].Sessions = 0
		admission, err := s.chooseScaleDownReplicant(round, instance, current)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(admission).Should(And(
			HaveField("Reason", BeEmpty()),
			HaveField("Pod", Not(BeNil())),
		))
	})
})
