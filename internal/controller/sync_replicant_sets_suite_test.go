package controller

import (
	crd "github.com/emqx/emqx-operator/api/v3alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reconciler syncReplicantSets", Ordered, func() {
	var ns *corev1.Namespace = &corev1.Namespace{}
	var instance *crd.EMQX

	var s *syncReplicantSets
	var round *reconcileRound

	var coreSet *appsv1.StatefulSet
	var update, current *appsv1.ReplicaSet
	var currentReplicantPod *corev1.Pod

	const (
		currentRevision string = "current"
		updateRevision  string = "update"
	)

	BeforeAll(func() {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-sync-replicant-sets-suite-test",
				Labels: map[string]string{
					"test": "e2e",
				},
			},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

		instance = emqx.DeepCopy()
		coreLabels := instance.DefaultLabelsWith(crd.CoreLabels())
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
			crd.ReplicantLabels(),
			map[string]string{crd.LabelPodTemplateHash: updateRevision},
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
		current.Labels[crd.LabelPodTemplateHash] = currentRevision
		current.Spec.Selector.MatchLabels[crd.LabelPodTemplateHash] = currentRevision
		current.Spec.Template.Labels[crd.LabelPodTemplateHash] = currentRevision

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
		update.Status.AvailableReplicas = 1
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
		instance.Spec.ReplicantTemplate = &crd.EMQXReplicantTemplate{
			Spec: crd.EMQXReplicantTemplateSpec{
				Replicas: ptr.To(int32(1)),
			},
		}
		instance.Status = crd.EMQXStatus{
			CoreNodesStatus: crd.CoreNodesStatus{
				ReadyReplicas: 1,
			},
			CoreNodes: []crd.EMQXNode{},
			ReplicantNodesStatus: crd.ReplicantNodesStatus{
				UpdateRevision:  updateRevision,
				UpdateReplicas:  1,
				CurrentRevision: currentRevision,
				CurrentReplicas: 1,
				ReadyReplicas:   2,
			},
			ReplicantNodes: []crd.EMQXNode{
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

var _ = Describe("Reconciler syncReplicantSets admission", Ordered, func() {
	var ns *corev1.Namespace = &corev1.Namespace{}
	var instance *crd.EMQX

	var s *syncReplicantSets
	var round *reconcileRound
	var current *appsv1.ReplicaSet
	var update *appsv1.ReplicaSet
	var currentPod *corev1.Pod
	var updatePod *corev1.Pod

	const (
		currentRevision string = "current"
		updateRevision  string = "update"
	)

	BeforeAll(func() {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-sync-replicant-sets-admission-test",
				Labels: map[string]string{
					"test": "e2e",
				},
			},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

		// Create "current" (old) RS with a known hash label.
		currentLabels := emqx.DefaultLabelsWith(
			crd.ReplicantLabels(),
			map[string]string{crd.LabelPodTemplateHash: currentRevision},
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
			crd.ReplicantLabels(),
			map[string]string{crd.LabelPodTemplateHash: updateRevision},
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

		update.Status.Replicas = 1
		update.Status.ReadyReplicas = 1
		update.Status.AvailableReplicas = 1
		Expect(k8sClient.Status().Update(ctx, update)).Should(Succeed())
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})

	BeforeEach(func() {
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.ReplicantTemplate = &crd.EMQXReplicantTemplate{
			Spec: crd.EMQXReplicantTemplateSpec{
				Replicas: ptr.To(int32(1)),
			},
		}
		instance.Status.ReplicantNodesStatus.CurrentRevision = currentRevision
		instance.Status.ReplicantNodesStatus.UpdateRevision = updateRevision
		instance.Status.ReplicantNodes = []crd.EMQXNode{
			{Name: "emqx@10.0.0.1", PodName: currentPod.Name, Status: "running"},
		}
		s = &syncReplicantSets{emqxReconciler}
		round = newReconcileRound()
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(update), update)).Should(Succeed())
		update.Status.AvailableReplicas = 1
		Expect(k8sClient.Status().Update(ctx, update)).Should(Succeed())
		Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
	})

	It("replicants not available (update pod not yet ready)", func() {
		update.Status.AvailableReplicas = 0
		Expect(k8sClient.Status().Update(ctx, update)).Should(Succeed())
		Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
		admission, err := s.chooseScaleDownReplicant(round, instance, current)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(admission).Should(And(
			HaveField("Reason", ContainSubstring("not available")),
			HaveField("Pod", BeNil()),
		))
	})

	It("node evacuation in progress", func() {
		instance.Status.NodeEvacuations = []crd.NodeEvacuationStatus{
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
