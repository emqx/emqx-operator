package controller

import (
	"time"

	crdv2 "github.com/emqx/emqx-operator/api/v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/emqx/emqx-operator/test/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reconciler addReplicantSet", Ordered, func() {
	var ns *corev1.Namespace = &corev1.Namespace{}
	var instance *crdv2.EMQX = &crdv2.EMQX{}
	var coreSet *appsv1.StatefulSet
	var corePod0, corePod1 *corev1.Pod
	var a *addReplicantSet
	var round *reconcileRound

	BeforeAll(func() {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-add-emqx-repl-test",
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

	BeforeEach(func() {
		// Create EMQX instance:
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(2))
		instance.Spec.ReplicantTemplate = &crdv2.EMQXReplicantTemplate{
			Spec: crdv2.EMQXReplicantTemplateSpec{
				Replicas: ptr.To(int32(3)),
			},
		}
		Expect(k8sClient.Create(ctx, instance)).To(Succeed())
		// Simulate core nodes readiness:
		coreLabels := instance.DefaultLabelsWith(crdv2.CoreLabels())
		coreSet = &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instance.Name + "-core",
				Namespace: ns.Name,
				Labels:    coreLabels,
			},
			Spec: appsv1.StatefulSetSpec{
				ServiceName: instance.Name + "-core",
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
		Expect(k8sClient.Create(ctx, coreSet)).To(Succeed())
		corePod0 = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:            coreSet.Name + "-0",
				Namespace:       ns.Name,
				Labels:          coreLabels,
				OwnerReferences: ownerReferences(coreSet),
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "emqx", Image: "emqx"},
				},
			},
		}
		corePod1 = corePod0.DeepCopy()
		corePod1.ObjectMeta.Name = coreSet.Name + "-1"
		Expect(k8sClient.Create(ctx, corePod0)).To(Succeed())
		Expect(k8sClient.Create(ctx, corePod1)).To(Succeed())
		instance.Status.CoreNodesStatus.ReadyReplicas = 2
		coreSet.Status.Replicas = 2
		coreSet.Status.ReadyReplicas = 2
		coreSet.Status.AvailableReplicas = 2
		coreSet.Status.UpdatedReplicas = 2
		coreSet.Status.CurrentReplicas = 2
		corePod0.Status.Conditions = []corev1.PodCondition{
			{
				Type:               corev1.PodReady,
				Status:             corev1.ConditionTrue,
				LastTransitionTime: metav1.NewTime(time.Now().Add(-1 * time.Minute)),
			},
		}
		corePod1.Status.Conditions = corePod0.Status.Conditions
		Expect(k8sClient.Status().Update(ctx, instance)).To(Succeed())
		Expect(k8sClient.Status().Update(ctx, coreSet)).To(Succeed())
		Expect(k8sClient.Status().Update(ctx, corePod0)).To(Succeed())
		Expect(k8sClient.Status().Update(ctx, corePod1)).To(Succeed())
		// Instantiate reconciler and reconcile round:
		a = &addReplicantSet{emqxReconciler}
		round = newReconcileRound()
		round.state, _ = loadReconcileState(ctx, k8sClient, instance)
	})

	AfterEach(func() {
		_ = k8sClient.Delete(ctx, corePod0)
		_ = k8sClient.Delete(ctx, corePod1)
		Expect(k8sClient.Delete(ctx, coreSet)).To(Succeed())
		Expect(k8sClient.Delete(ctx, instance)).To(Succeed())
	})

	When("replicant template is nil", func() {
		It("should do nothing", func() {
			// Clear replicant template:
			instance.Spec.ReplicantTemplate = nil
			// Reconciliation step should do nothing and succeed:
			result := a.reconcile(round, instance)
			Expect(result.err).ToNot(HaveOccurred())
			Expect(replicantSets(instance)).To(BeEmpty())
		})
	})

	When("single core node instance", func() {
		It("should fail to create", func() {
			instanceInvalid := instance.DeepCopy()
			instanceInvalid.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(1))
			instanceInvalid.Spec.ReplicantTemplate.Spec.Replicas = ptr.To(int32(2))
			Expect(k8sClient.Create(ctx, instanceInvalid)).To(HaveOccurred())
		})
	})

	When("no available core pods", func() {
		It("should do nothing", func() {
			coreSet.Status.AvailableReplicas = 0
			Expect(k8sClient.Status().Update(ctx, coreSet)).To(Succeed())
			Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
			// Reconciliation step should succeed but not create any RS:
			result := a.reconcile(round, instance)
			Expect(result.err).ToNot(HaveOccurred())
			Expect(replicantSets(instance)).To(BeEmpty())
		})
	})

	When("core nodes are ready", func() {
		It("should create replicaSet", func() {
			result := a.reconcile(round, instance)
			Expect(result.err).ToNot(HaveOccurred())
			Expect(replicantSets(instance)).To(ConsistOf(
				HaveField("Spec.Template.Spec.Containers", ConsistOf(
					HaveField("Image", Equal(instance.Spec.Image)),
				)),
			))
		})

		AfterAll(func() {
			cleanupReplicantSets(instance)
		})
	})

	When("number of replicas decreases", func() {
		BeforeAll(func() {
			// Start with 3 replicas:
			instance.Spec.ReplicantTemplate.Spec.Replicas = ptr.To(int32(3))
			// Run the reconcile to create replicantSet:
			result := a.reconcile(round, instance)
			Expect(result.err).ToNot(HaveOccurred())
			Expect(replicantSets(instance)).To(ConsistOf(
				HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(3))),
			))
		})

		It("should scale down replicaSet", func() {
			// Set replicas count to 0:
			instance.Spec.ReplicantTemplate.Spec.Replicas = ptr.To(int32(0))
			Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
			// Reconciliation step should succeed:
			result := a.reconcile(round, instance)
			Expect(result.err).ToNot(HaveOccurred())
			// Status conditions should reset:
			Expect(actualObject(instance)).To(And(
				HaveCondition(crdv2.Ready, HaveField("Status", Equal(metav1.ConditionFalse))),
				HaveCondition(crdv2.ReplicantNodesProgressing, HaveField("Status", Equal(metav1.ConditionTrue))),
			))
			// ReplicaSet should be updated in place:
			Expect(replicantSets(instance)).To(ConsistOf(
				HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(0))),
			))
		})

		AfterAll(func() {
			cleanupReplicantSets(instance)
		})
	})

	When("number of replicas increases", func() {
		BeforeAll(func() {
			// Start with 1 replicas:
			instance.Spec.ReplicantTemplate.Spec.Replicas = ptr.To(int32(1))
			// Run the reconcile to create replicantSet:
			result := a.reconcile(round, instance)
			Expect(result.err).ToNot(HaveOccurred())
			Expect(replicantSets(instance)).To(ConsistOf(
				HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(1))),
			))
		})

		It("scales up replicaSet", func() {
			// Set replicas count to 4:
			instance.Spec.ReplicantTemplate.Spec.Replicas = ptr.To(int32(4))
			Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
			// Reconciliation step should succeed:
			result := a.reconcile(round, instance)
			Expect(result.err).ToNot(HaveOccurred())
			// Status conditions should reset:
			Expect(actualObject(instance)).To(And(
				HaveCondition(crdv2.Ready, HaveField("Status", Equal(metav1.ConditionFalse))),
				HaveCondition(crdv2.ReplicantNodesProgressing, HaveField("Status", Equal(metav1.ConditionTrue))),
			))
			// ReplicaSet should be updated:
			Expect(replicantSets(instance)).To(ConsistOf(
				HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(4))),
			))
		})

		AfterAll(func() {
			cleanupReplicantSets(instance)
		})
	})

	When("image is changed", func() {
		BeforeAll(func() {
			// Start with 1 replicas:
			instance.Spec.ReplicantTemplate.Spec.Replicas = ptr.To(int32(1))
			// Run the reconcile to create replicantSet:
			result := a.reconcile(round, instance)
			Expect(result.err).ToNot(HaveOccurred())
			Expect(replicantSets(instance)).To(ConsistOf(
				HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(1))),
			))
		})

		It("should create new replicaSet", func() {
			// Introduce changes that require creating a new replicaSet:
			instance.Spec.Image = "emqx/emqx"
			Expect(k8sClient.Update(ctx, instance)).To(Succeed())
			Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
			// Reconciliation step should succeed:
			result := a.reconcile(round, instance)
			Expect(result.err).ToNot(HaveOccurred())
			// There should be two replicaSets soon:
			Expect(replicantSets(instance)).To(ConsistOf(
				HaveField("Spec.Template.Spec.Containers", ConsistOf(HaveField("Image", Equal(emqx.Spec.Image)))),
				HaveField("Spec.Template.Spec.Containers", ConsistOf(HaveField("Image", Equal("emqx/emqx")))),
			))
			// Update revision should differ from current revision (new RS created):
			Expect(actualObject(instance)).To(
				HaveField("Status.ReplicantNodesStatus", WithTransform(
					func(s crdv2.ReplicantNodesStatus) bool {
						return s.UpdateRevision != "" && s.UpdateRevision != s.CurrentRevision
					},
					BeTrue(),
				)))
		})

		AfterAll(func() {
			cleanupReplicantSets(instance)
		})
	})

})

func replicantSets(instance *crdv2.EMQX) []appsv1.ReplicaSet {
	list := &appsv1.ReplicaSetList{}
	_ = k8sClient.List(ctx, list,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(instance.DefaultLabelsWith(crdv2.ReplicantLabels())),
	)
	return list.Items
}

func cleanupReplicantSets(instance *crdv2.EMQX) {
	replicantSets := replicantSets(instance)
	for _, rs := range replicantSets {
		_ = k8sClient.Delete(ctx, &rs)
	}
}
