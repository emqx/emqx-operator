package controller

import (
	crdv2 "github.com/emqx/emqx-operator/api/v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const currentRevision string = "current"
const updateRevision string = "update"

var _ = Describe("Reconciler syncReplicantSet", Ordered, func() {
	var ns *corev1.Namespace = &corev1.Namespace{}
	var instance *crdv2.EMQX

	var s *syncReplicantSets
	var round *reconcileRound

	var coreSet *appsv1.StatefulSet
	var update, current *appsv1.ReplicaSet
	var currentReplicantPod *corev1.Pod

	BeforeAll(func() {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-sync-pods-suite-test",
				Labels: map[string]string{
					"test": "e2e",
				},
			},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

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

		updateReplicantLabels := instance.DefaultLabelsWith(
			crdv2.ReplicantLabels(),
			map[string]string{crdv2.LabelPodTemplateHash: updateRevision},
		)
		update = &appsv1.ReplicaSet{
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
		current = update.DeepCopy()
		current.Labels[crdv2.LabelPodTemplateHash] = currentRevision
		current.Spec.Selector.MatchLabels[crdv2.LabelPodTemplateHash] = currentRevision
		current.Spec.Template.Labels[crdv2.LabelPodTemplateHash] = currentRevision

		Expect(k8sClient.Create(ctx, coreSet)).Should(Succeed())
		Expect(k8sClient.Create(ctx, update)).Should(Succeed())
		Expect(k8sClient.Create(ctx, current)).Should(Succeed())

		currentReplicantPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName:    current.Name + "-",
				Namespace:       current.Namespace,
				Labels:          current.Spec.Template.Labels,
				OwnerReferences: ownerReferences(current),
			},
			Spec: current.Spec.Template.Spec,
		}
		Expect(k8sClient.Create(ctx, currentReplicantPod)).Should(Succeed())

		// Create a pod for the update RS so areReplicantsAvailable is satisfied.
		updateReplicantPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName:    update.Name + "-",
				Namespace:       update.Namespace,
				Labels:          update.Spec.Template.Labels,
				OwnerReferences: ownerReferences(update),
			},
			Spec: update.Spec.Template.Spec,
		}
		Expect(k8sClient.Create(ctx, updateReplicantPod)).Should(Succeed())
		updateReplicantPod.Status.Conditions = []corev1.PodCondition{
			{Type: corev1.PodReady, Status: corev1.ConditionTrue, LastTransitionTime: metav1.Now()},
		}
		Expect(k8sClient.Status().Update(ctx, updateReplicantPod)).Should(Succeed())

		coreSet.Status.Replicas = 1
		coreSet.Status.ReadyReplicas = 1
		coreSet.Status.UpdateRevision = updateRevision
		coreSet.Status.CurrentRevision = updateRevision
		update.Status.Replicas = 1
		update.Status.ReadyReplicas = 1
		current.Status.Replicas = 1
		current.Status.ReadyReplicas = 1
		Expect(k8sClient.Status().Update(ctx, coreSet)).Should(Succeed())
		Expect(k8sClient.Status().Update(ctx, update)).Should(Succeed())
		Expect(k8sClient.Status().Update(ctx, current)).Should(Succeed())
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})

	BeforeEach(func() {
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.ReplicantTemplate = &crdv2.EMQXReplicantTemplate{
			Spec: crdv2.EMQXReplicantTemplateSpec{
				Replicas: ptr.To(int32(1)),
			},
		}
		instance.Status = crdv2.EMQXStatus{
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
		s = &syncReplicantSets{emqxReconciler}
		round = newReconcileRound()
		Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
	})

	It("should scale down current replicant set and annotate pod", func() {
		Expect(s.reconcile(round, instance)).To(Equal(subResult{}))
		// Pod should be annotated with deletion cost:
		Expect(actualObject(currentReplicantPod)).To(
			HaveField("Annotations", HaveKeyWithValue("controller.kubernetes.io/pod-deletion-cost", "-99999")),
		)
		// Current RS should be scaled down:
		Expect(actualObject(current)).To(
			HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(0))),
		)
	})
})

var _ = Describe("Reconciler syncCoreSet", Ordered, func() {
	var ns *corev1.Namespace = &corev1.Namespace{}
	var instance *crdv2.EMQX

	var round *reconcileRound
	var coreSet *appsv1.StatefulSet
	var pod0, pod1 *corev1.Pod

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
		coreLabels := emqx.DefaultLabelsWith(crdv2.CoreLabels())
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
					crdv2.LabelInstance:                   "emqx",
					crdv2.LabelManagedBy:                  "emqx-operator",
					crdv2.LabelDBRole:                     "core",
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
		coreSet.Status.CurrentRevision = currentRevision
		coreSet.Status.UpdateRevision = updateRevision
		Expect(k8sClient.Status().Update(ctx, coreSet)).Should(Succeed())

		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(2))
		instance.Status.CoreNodes = []crdv2.EMQXNode{
			{Name: "emqx@" + pod0.Name, PodName: pod0.Name, Status: "running"},
			{Name: "emqx@" + pod1.Name, PodName: pod1.Name, Status: "running"},
		}

		round = newReconcileRound()
		Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
	})

	AfterEach(func() {
		k8sClient.Delete(ctx, pod0)
		k8sClient.Delete(ctx, pod1)
		Expect(k8sClient.Delete(ctx, coreSet)).Should(Succeed())
	})

	It("cores not available", func() {
		// Pod 0 is not Ready, so areCoresAvailable returns false.
		pod0.Status.Conditions = []corev1.PodCondition{}
		Expect(k8sClient.Status().Update(ctx, pod0)).Should(Succeed())
		Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
		admission := checkCorePodRemoval(round, instance, pod1, false)
		Expect(admission).Should(And(
			HaveField("Action", Equal(admissionWait)),
			HaveField("Reason", ContainSubstring("not available")),
		))
	})

	It("cores available / MinReadySeconds has not passed", func() {
		// But MinReadySeconds is very large, so pod is not yet "available".
		instance.Spec.UpdateStrategy.MinReadySeconds = 99999999
		admission := checkCorePodRemoval(round, instance, pod1, false)
		Expect(admission).Should(And(
			HaveField("Action", Equal(admissionWait)),
			HaveField("Reason", ContainSubstring("not available")),
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
		instance.Status.CoreNodes = []crdv2.EMQXNode{
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
			instance.Spec.ReplicantTemplate = &crdv2.EMQXReplicantTemplate{
				Spec: crdv2.EMQXReplicantTemplateSpec{
					Replicas: ptr.To(int32(3)),
				},
			}
			instance.Status.ReplicantNodesStatus = crdv2.ReplicantNodesStatus{
				UpdateRevision:  updateRevision,
				CurrentRevision: currentRevision,
			}
		})

		It("should allow rolling update with multiple old cores", func() {
			s := &syncCoreSet{emqxReconciler}
			result := s.rollingUpdate(round, instance)
			Expect(result.err).ShouldNot(HaveOccurred())
			// Highest ordinal `pod1` should have been deleted.
			_, err := actualObject(pod1)
			Expect(err).Should(HaveOccurred())
		})

		It("should block rolling update with 1 old core", func() {
			// Simulate `pod1` already on new revision, while `pod0` is outdated.
			pod1.Labels[appsv1.ControllerRevisionHashLabelKey] = coreSet.Status.UpdateRevision
			Expect(k8sClient.Update(ctx, pod1)).Should(Succeed())
			Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
			// Perform rolling update reconciliation.
			s := &syncCoreSet{emqxReconciler}
			result := s.rollingUpdate(round, instance)
			Expect(result.err).ShouldNot(HaveOccurred())
			// Latest old core `pod0` should have been deleted.
			Expect(actualObject(pod0)).ShouldNot(BeNil())
		})

	})

	It("DS replication site blocks scale-down", func() {
		pod1.Status.Conditions = []corev1.PodCondition{
			{Type: crdv2.DSReplicationSite, Status: corev1.ConditionTrue},
		}
		admission := checkCorePodRemoval(round, instance, pod1, true)
		Expect(admission).To(And(
			HaveField("Action", Equal(admissionWait)),
			HaveField("Reason", ContainSubstring("DS replication site")),
		))
	})

	It("DS replication site does not block rolling update", func() {
		pod1.Status.Conditions = []corev1.PodCondition{
			{Type: crdv2.DSReplicationSite, Status: corev1.ConditionTrue},
		}
		admission := checkCorePodRemoval(round, instance, pod1, false)
		Expect(admission).To(
			HaveField("Action", Equal(admissionRemove)),
		)
	})
})

var _ = Describe("Reconciler syncReplicantSets admission", Ordered, func() {
	var ns *corev1.Namespace = &corev1.Namespace{}
	var instance *crdv2.EMQX

	var s *syncReplicantSets
	var round *reconcileRound
	var current *appsv1.ReplicaSet
	var update *appsv1.ReplicaSet
	var currentPod *corev1.Pod
	var updatePod *corev1.Pod

	BeforeAll(func() {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-sync-replicant-sets-test",
				Labels: map[string]string{
					"test": "e2e",
				},
			},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

		// Create "current" (old) RS with a known hash label.
		currentLabels := emqx.DefaultLabelsWith(
			crdv2.ReplicantLabels(),
			map[string]string{crdv2.LabelPodTemplateHash: currentRevision},
		)
		current = &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: emqx.Name + "-",
				Namespace:    ns.Name,
				Labels:       currentLabels,
			},
			Spec: appsv1.ReplicaSetSpec{
				Replicas: ptr.To(int32(1)),
				Selector: &metav1.LabelSelector{
					MatchLabels: currentLabels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: currentLabels,
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

		// Create pod owned by "current" RS.
		currentPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName:    current.Name + "-",
				Namespace:       ns.Name,
				Labels:          current.Spec.Selector.MatchLabels,
				OwnerReferences: ownerReferences(current),
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "emqx", Image: "emqx"},
				},
			},
		}
		Expect(k8sClient.Create(ctx, currentPod)).Should(Succeed())

		// Create "update" (new) RS with a different hash label.
		updateLabels := emqx.DefaultLabelsWith(
			crdv2.ReplicantLabels(),
			map[string]string{crdv2.LabelPodTemplateHash: updateRevision},
		)
		update = &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: emqx.Name + "-",
				Namespace:    ns.Name,
				Labels:       updateLabels,
			},
			Spec: appsv1.ReplicaSetSpec{
				Replicas: ptr.To(int32(1)),
				Selector: &metav1.LabelSelector{
					MatchLabels: updateLabels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: updateLabels,
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "emqx", Image: "emqx:new"},
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, update)).Should(Succeed())

		// Create a Ready pod owned by "update" RS so that areReplicantsAvailable passes.
		updatePod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName:    update.Name + "-",
				Namespace:       ns.Name,
				Labels:          update.Spec.Selector.MatchLabels,
				OwnerReferences: ownerReferences(update),
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "emqx", Image: "emqx:new"},
				},
			},
		}
		Expect(k8sClient.Create(ctx, updatePod)).Should(Succeed())
		updatePod.Status.Conditions = []corev1.PodCondition{
			{Type: corev1.PodReady, Status: corev1.ConditionTrue, LastTransitionTime: metav1.Now()},
		}
		Expect(k8sClient.Status().Update(ctx, updatePod)).Should(Succeed())
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})

	BeforeEach(func() {
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.ReplicantTemplate = &crdv2.EMQXReplicantTemplate{
			Spec: crdv2.EMQXReplicantTemplateSpec{
				Replicas: ptr.To(int32(1)),
			},
		}
		instance.Status.ReplicantNodesStatus.CurrentRevision = currentRevision
		instance.Status.ReplicantNodesStatus.UpdateRevision = updateRevision
		instance.Status.ReplicantNodes = []crdv2.EMQXNode{
			{Name: "emqx@10.0.0.1", PodName: currentPod.Name, Status: "running"},
		}
		s = &syncReplicantSets{emqxReconciler}
		round = newReconcileRound()
		Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
	})

	It("replicants not available (update pod not yet ready)", func() {
		instance.Spec.UpdateStrategy.MinReadySeconds = 99999999
		admission, err := s.chooseScaleDownReplicant(round, instance, current)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(admission).Should(And(
			HaveField("Reason", ContainSubstring("not available")),
			HaveField("Pod", BeNil()),
		))
	})

	It("node evacuation in progress", func() {
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
