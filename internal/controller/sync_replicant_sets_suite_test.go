package controller

import (
	"fmt"
	"slices"

	crd "github.com/emqx/emqx-operator/api/v3alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reconciler syncReplicantSets", Ordered, func() {
	var ns *corev1.Namespace = &corev1.Namespace{}
	var instance *crd.EMQX

	const (
		currentRevision string = "current"
		updateRevision  string = "update"
	)

	updateLabels := emqx.DefaultLabelsWith(
		crd.ReplicantLabels(),
		map[string]string{crd.LabelPodTemplateHash: updateRevision},
	)
	currentLabels := emqx.DefaultLabelsWith(
		crd.ReplicantLabels(),
		map[string]string{crd.LabelPodTemplateHash: currentRevision},
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
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})

	Context("rolling update", func() {
		var update, current *appsv1.ReplicaSet
		var currentReplicants []*corev1.Pod
		var updateReplicant *corev1.Pod
		var resources []client.Object

		BeforeEach(func() {
			instance = emqx.DeepCopy()
			instance.Namespace = ns.Name
			instance.Spec.ReplicantTemplate = &crd.EMQXReplicantTemplate{
				Spec: crd.EMQXReplicantTemplateSpec{
					Replicas: ptr.To(int32(3)),
				},
			}

			resources = []client.Object{}

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
								{Name: "emqx", Image: "emqx"},
							},
						},
					},
				},
			}
			current = update.DeepCopy()
			current.Labels = currentLabels
			current.Spec.Selector.MatchLabels = currentLabels
			current.Spec.Template.Labels = currentLabels
			current.Spec.Replicas = ptr.To(int32(3))
			Expect(k8sClient.Create(ctx, update)).Should(Succeed())
			Expect(k8sClient.Create(ctx, current)).Should(Succeed())

			resources = append(resources, current, update)

			currentReplicants = []*corev1.Pod{}
			for i := 0; i < 3; i++ {
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName:    current.Name + "-" + currentRevision + fmt.Sprint(i) + "-",
						Namespace:       current.Namespace,
						Labels:          current.Spec.Template.Labels,
						OwnerReferences: ownerReferences(current),
					},
					Spec: current.Spec.Template.Spec,
				}
				Expect(k8sClient.Create(ctx, pod)).Should(Succeed())
				currentReplicants = append(currentReplicants, pod)
				resources = append(resources, pod)
			}

			updateReplicant = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName:    update.Name + "-" + updateRevision + "0-",
					Namespace:       update.Namespace,
					Labels:          update.Spec.Template.Labels,
					OwnerReferences: ownerReferences(update),
				},
				Spec: update.Spec.Template.Spec,
			}
			Expect(k8sClient.Create(ctx, updateReplicant)).Should(Succeed())
			resources = append(resources, updateReplicant)

			update.Status.Replicas = 1
			update.Status.ReadyReplicas = 1
			update.Status.AvailableReplicas = 0
			current.Status.Replicas = 3
			current.Status.ReadyReplicas = 3
			current.Status.AvailableReplicas = 3
			Expect(k8sClient.Status().Update(ctx, update)).Should(Succeed())
			Expect(k8sClient.Status().Update(ctx, current)).Should(Succeed())

			instance.Status = crd.EMQXStatus{
				CoreNodesStatus: crd.CoreNodesStatus{
					ReadyReplicas: 1,
				},
				CoreNodes: []crd.EMQXNode{},
				ReplicantNodesStatus: crd.ReplicantNodesStatus{
					UpdateRevision:  updateRevision,
					UpdateReplicas:  1,
					CurrentRevision: currentRevision,
					CurrentReplicas: 3,
					ReadyReplicas:   4,
				},
				ReplicantNodes: []crd.EMQXNode{
					{Name: "emqx@10.0.0.1", PodName: currentReplicants[0].Name, Status: "running"},
					{Name: "emqx@10.0.0.2", PodName: currentReplicants[1].Name, Status: "running"},
					{Name: "emqx@10.0.0.3", PodName: currentReplicants[2].Name, Status: "running"},
				},
			}
		})

		AfterEach(func() {
			slices.Reverse(resources)
			for _, r := range resources {
				k8sClient.Delete(ctx, r)
			}
		})

		It("should scale down current replicant set and annotate pod", func() {
			s := &syncReplicantSets{emqxReconciler}
			round := newReconcileRound()
			Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
			Expect(s.reconcile(round, instance)).To(Equal(subResult{}))
			// Pod should be annotated with deletion cost:
			Expect(actualObject(currentReplicants[0])).To(
				HaveField("Annotations", HaveKey("controller.kubernetes.io/pod-deletion-cost")),
			)
			// Current RS should be scaled down:
			Expect(actualObject(current)).To(
				HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(2))),
			)
		})

		It("drains up to maxUnavailable old pods in one reconcile", func() {
			instance.Spec.UpdateStrategy.MaxUnavailable = ptr.To(intstr.FromInt(3))
			s := &syncReplicantSets{emqxReconciler}
			round := newReconcileRound()
			Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
			Expect(s.reconcile(round, instance)).To(Equal(subResult{}))
			for _, p := range currentReplicants {
				Expect(actualObject(p)).To(
					HaveField("Annotations", HaveKey("controller.kubernetes.io/pod-deletion-cost")),
				)
			}
			Expect(actualObject(current)).To(
				HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(0))),
			)
		})

		It("drains no pods if maxUnavailable reached", func() {
			instance.Spec.UpdateStrategy.MaxUnavailable = ptr.To(intstr.FromInt(3))
			Expect(actualObject(current)).To(Not(BeNil()))
			current.Status.AvailableReplicas = 0
			Expect(k8sClient.Status().Update(ctx, current)).Should(Succeed())
			s := &syncReplicantSets{emqxReconciler}
			round := newReconcileRound()
			Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
			Expect(s.reconcile(round, instance)).To(Equal(subResult{}))
			for _, p := range currentReplicants {
				Expect(actualObject(p)).To(Not(
					HaveField("Annotations", HaveKey("controller.kubernetes.io/pod-deletion-cost")),
				))
			}
			Expect(actualObject(current)).To(
				HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(3))),
			)
		})

		It("surges the update RS gradually with maxSurge", func() {
			instance.Spec.UpdateStrategy.MaxSurge = ptr.To(intstr.FromInt(2))
			s := &syncReplicantSets{emqxReconciler}
			round := newReconcileRound()
			Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())

			// Phase 2: numAllowedReplicas = min(3 + 2 - 3, 3) = 2
			Expect(s.reconcile(round, instance)).To(Equal(subResult{}))
			Expect(actualObject(update)).To(
				HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(2))),
			)

			// Simulate current RS scaled down to 2 (one pod drained).
			Expect(actualObject(current)).To(Not(BeNil()))
			Expect(k8sClient.Delete(ctx, currentReplicants[0])).Should(Succeed())
			current.Spec.Replicas = ptr.To(int32(2))
			Expect(k8sClient.Update(ctx, current)).Should(Succeed())
			current.Status.Replicas = 2
			current.Status.ReadyReplicas = 2
			current.Status.AvailableReplicas = 2
			Expect(k8sClient.Status().Update(ctx, current)).Should(Succeed())

			// Phase 2: numAllowedReplicas = min(3 + 2 - 2, 3) = 2
			round = newReconcileRound()
			Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
			Expect(s.reconcile(round, instance)).To(Equal(subResult{}))
			Expect(actualObject(update)).To(
				HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(3))),
			)

			// Simulate current RS scaled down to 1 (one more pod drained).
			Expect(actualObject(current)).To(Not(BeNil()))
			Expect(k8sClient.Delete(ctx, currentReplicants[1])).Should(Succeed())
			current.Spec.Replicas = ptr.To(int32(1))
			Expect(k8sClient.Update(ctx, current)).Should(Succeed())
			current.Status.Replicas = 1
			current.Status.ReadyReplicas = 1
			current.Status.AvailableReplicas = 1
			Expect(k8sClient.Status().Update(ctx, current)).Should(Succeed())

			// Phase 3: numAllowedReplicas = min(3 + 2 - 1, 3) = still 3
			round = newReconcileRound()
			Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
			Expect(s.reconcile(round, instance)).To(Equal(subResult{}))
			Expect(actualObject(update)).To(
				HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(3))),
			)
		})
	})

	Context("scale-up / scale-down", func() {
		var rs *appsv1.ReplicaSet
		var replicants []*corev1.Pod
		var resources []client.Object

		const revision = "steady"

		BeforeEach(func() {
			instance = emqx.DeepCopy()
			instance.Namespace = ns.Name
			instance.Spec.ReplicantTemplate = &crd.EMQXReplicantTemplate{
				Spec: crd.EMQXReplicantTemplateSpec{
					Replicas: ptr.To(int32(3)),
				},
			}

			resources = []client.Object{}

			steadyLabels := emqx.DefaultLabelsWith(
				crd.ReplicantLabels(),
				map[string]string{crd.LabelPodTemplateHash: revision},
			)
			rs = &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: emqx.Name + "-",
					Namespace:    ns.Name,
					Labels:       steadyLabels,
				},
				Spec: appsv1.ReplicaSetSpec{
					Replicas: ptr.To(int32(3)),
					Selector: &metav1.LabelSelector{
						MatchLabels: steadyLabels,
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: steadyLabels,
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "emqx", Image: "emqx"},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, rs)).Should(Succeed())
			resources = append(resources, rs)

			replicants = []*corev1.Pod{}
			for i := 0; i < 3; i++ {
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName:    rs.Name + "-" + fmt.Sprint(i) + "-",
						Namespace:       ns.Name,
						Labels:          steadyLabels,
						OwnerReferences: ownerReferences(rs),
					},
					Spec: rs.Spec.Template.Spec,
				}
				Expect(k8sClient.Create(ctx, pod)).Should(Succeed())
				replicants = append(replicants, pod)
				resources = append(resources, pod)
			}

			rs.Status.Replicas = 3
			rs.Status.ReadyReplicas = 3
			rs.Status.AvailableReplicas = 3
			Expect(k8sClient.Status().Update(ctx, rs)).Should(Succeed())

			// Steady state: current == update revision.
			instance.Status = crd.EMQXStatus{
				CoreNodesStatus: crd.CoreNodesStatus{
					ReadyReplicas: 1,
				},
				ReplicantNodesStatus: crd.ReplicantNodesStatus{
					UpdateRevision:  revision,
					CurrentRevision: revision,
					CurrentReplicas: 3,
					UpdateReplicas:  3,
					ReadyReplicas:   3,
				},
				ReplicantNodes: []crd.EMQXNode{
					{Name: "emqx@10.0.0.1", PodName: replicants[0].Name, Status: "running"},
					{Name: "emqx@10.0.0.2", PodName: replicants[1].Name, Status: "running"},
					{Name: "emqx@10.0.0.3", PodName: replicants[2].Name, Status: "running"},
				},
			}
		})

		AfterEach(func() {
			slices.Reverse(resources)
			for _, r := range resources {
				k8sClient.Delete(ctx, r)
			}
		})

		It("scales up replicant set when desired > current", func() {
			instance.Spec.ReplicantTemplate.Spec.Replicas = ptr.To(int32(5))
			s := &syncReplicantSets{emqxReconciler}
			round := newReconcileRound()
			Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
			Expect(s.reconcile(round, instance)).To(Equal(subResult{}))
			Expect(actualObject(rs)).To(
				HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(5))),
			)
		})

		It("scales down replicant set up to maxUnavailable pods at a time", func() {
			// Allow 2 maxUnavailable replicas
			instance.Spec.UpdateStrategy.MaxUnavailable = ptr.To(intstr.FromInt(2))
			instance.Spec.ReplicantTemplate.Spec.Replicas = ptr.To(int32(0))

			s := &syncReplicantSets{emqxReconciler}
			round := newReconcileRound()
			Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
			Expect(s.reconcile(round, instance)).To(Equal(subResult{}))

			numAnnotated := 0
			for _, p := range replicants {
				Expect(actualObject(p)).ToNot(BeNil())
				if _, ok := p.Annotations[corev1.PodDeletionCost]; ok {
					numAnnotated++
				}
			}

			Expect(numAnnotated).To(Equal(2))
			Expect(actualObject(rs)).To(
				HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(1))),
			)
		})

		It("respects maxUnavailable budget during scale-down", func() {
			instance.Spec.UpdateStrategy.MaxUnavailable = ptr.To(intstr.FromInt(1))
			instance.Spec.ReplicantTemplate.Spec.Replicas = ptr.To(int32(0))
			instance.Status.ReplicantNodesStatus.ReadyReplicas = 0
			rs.Status.AvailableReplicas = 0
			Expect(k8sClient.Status().Update(ctx, rs)).Should(Succeed())

			s := &syncReplicantSets{emqxReconciler}
			round := newReconcileRound()
			Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
			Expect(s.reconcile(round, instance)).To(Equal(subResult{}))

			numAnnotated := 0
			for _, p := range replicants {
				Expect(actualObject(p)).ToNot(BeNil())
				if _, ok := p.Annotations[corev1.PodDeletionCost]; ok {
					numAnnotated++
				}
			}
			// Budget is 1, so exactly 1 pod annotated out of 3.
			Expect(numAnnotated).To(Equal(1))

			// RS decremented to 2 (3 - 1 annotated).
			Expect(actualObject(rs)).To(
				HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(2))),
			)
		})

		It("is a no-op at desired replica count", func() {
			instance.Spec.ReplicantTemplate.Spec.Replicas = ptr.To(int32(3))
			s := &syncReplicantSets{emqxReconciler}
			round := newReconcileRound()
			Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
			Expect(s.reconcile(round, instance)).To(Equal(subResult{}))

			// No pods annotated.
			for _, p := range replicants {
				Expect(actualObject(p)).To(Not(
					HaveField("Annotations", HaveKey(corev1.PodDeletionCost)),
				))
			}
			// RS replicas unchanged.
			Expect(actualObject(rs)).To(
				HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(3))),
			)
		})

		It("waits for evacuation during scale-down when node has sessions", func() {
			instance.Spec.ReplicantTemplate.Spec.Replicas = ptr.To(int32(2))
			// Give all nodes active sessions.
			for i := range instance.Status.ReplicantNodes {
				instance.Status.ReplicantNodes[i].Sessions = 100
			}
			s := &syncReplicantSets{emqxReconciler}
			round := newReconcileRound()
			Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())

			// The reconcile will attempt to start evacuation, which fails against the mock API
			// (501 response). The error propagation is the expected behavior here.
			result := s.reconcile(round, instance)
			Expect(result.err).To(HaveOccurred())

			// No pod should be annotated (evacuation must complete first).
			for _, p := range replicants {
				Expect(actualObject(p)).To(Not(
					HaveField("Annotations", HaveKey(corev1.PodDeletionCost)),
				))
			}
			// RS replicas unchanged.
			Expect(actualObject(rs)).To(
				HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(3))),
			)
		})
	})
})

var _ = Describe("Reconciler syncReplicantSets admission", Ordered, func() {
	var ns *corev1.Namespace = &corev1.Namespace{}
	var instance *crd.EMQX

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
		round = newReconcileRound()
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(update), update)).Should(Succeed())
		update.Status.AvailableReplicas = 1
		Expect(k8sClient.Status().Update(ctx, update)).Should(Succeed())
		Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
	})

	It("replicants not available", func() {
		update.Status.AvailableReplicas = 0
		Expect(k8sClient.Status().Update(ctx, update)).Should(Succeed())
		Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
		s := &syncReplicantSets{emqxReconciler}
		admissions := s.outdatedReplicantAdmissions(round, instance)
		Expect(admissions).Should(BeEmpty())
	})

	It("node evacuation in progress", func() {
		instance.Status.NodeEvacuations = []crd.NodeEvacuationStatus{
			{NodeName: "emqx@10.0.0.1", State: "fake"},
		}
		admission := checkReplicantPodRemoval(instance, currentPod)
		Expect(admission).Should(And(
			HaveField("Action", Equal(admissionWait)),
			HaveField("Reason", ContainSubstring("evacuation")),
		))
	})

	It("node session > 0", func() {
		instance.Status.ReplicantNodes[0].Sessions = 99999
		// With Replicas=1 and MaxSurge=0 the single-replica guard returns admissionRemove
		// (nowhere to evacuate). Set Replicas=2 to exercise the evacuation path.
		instance.Spec.ReplicantTemplate.Spec.Replicas = ptr.To(int32(2))
		admission := checkReplicantPodRemoval(instance, currentPod)
		Expect(admission).Should(And(
			HaveField("Action", Equal(admissionEvacuate)),
			HaveField("Reason", ContainSubstring("active sessions")),
		))
	})

	It("node session > 0 & single-node replicant cluster", func() {
		instance.Status.ReplicantNodes[0].Sessions = 99999
		admission := checkReplicantPodRemoval(instance, currentPod)
		Expect(admission).Should(And(
			HaveField("Action", Equal(admissionRemove)),
			HaveField("Reason", ContainSubstring("nowhere to evacuate")),
		))
	})

	It("node session is 0", func() {
		instance.Status.ReplicantNodes[0].Sessions = 0
		admission := checkReplicantPodRemoval(instance, currentPod)
		Expect(admission).Should(And(
			HaveField("Action", Equal(admissionRemove)),
			HaveField("Reason", ContainSubstring("safe to stop")),
		))
	})
})
