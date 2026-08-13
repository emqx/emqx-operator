package controller

import (
	"time"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	. "github.com/emqx/emqx-operator/test/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = DescribeClientFaultMatrix("Reconciler addCoreSet", Ordered, func() {
	var ns *corev1.Namespace
	var instance *crd.EMQX

	listCoreSets := func() []appsv1.StatefulSet {
		list := &appsv1.StatefulSetList{}
		_ = k8sClient.List(ctx, list,
			client.InNamespace(instance.Namespace),
			client.MatchingLabels(instance.DefaultLabelsWith(crd.CoreLabels())),
		)
		return list.Items
	}

	BeforeAll(func() {
		// Create namespace:
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "controller-add-emqx-core-test",
				Labels:       map[string]string{"test": "e2e"},
			},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
		// Create instance:
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(2))
		Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.DeleteAllOf(ctx, &appsv1.StatefulSet{}, client.InNamespace(ns.Name))).
			To(Succeed())
	})

	It("should create statefulSet", func() {
		round := newReconcileRound()
		a := &addCoreSet{emqxReconciler()}
		Eventually(runRoundReconcile).WithArguments(round, instance, a).
			Should(BeSuccessfulReconcile())
		Expect(listCoreSets()).To(ConsistOf(
			And(
				HaveField("Spec.Template.Spec.Containers", ConsistOf(
					HaveField("Image", Equal(instance.Spec.Image)))),
				HaveField("Annotations", And(
					HaveKey(crd.AnnotationCoreRetirementOrdinalWatermark),
					HaveKey(crd.AnnotationCoreRetirementOrdinalUpdatedAt),
				)),
			),
		))
	})

	It("change image updates existing statefulSet in place", func() {
		// Create initial coreSet:
		round := newReconcileRound()
		a := &addCoreSet{emqxReconciler()}
		Eventually(runRoundReconcile).WithArguments(round, instance, a).
			Should(BeSuccessfulReconcile())
		// Simulate scaling and retirement activities in the meantime:
		coreSets := listCoreSets()
		Expect(coreSets).To(HaveLen(1))
		coreSet := &coreSets[0]
		retirement := &coreSetRetirement{
			watermark: 42,
			updatedAt: time.Now().Add(-time.Minute),
		}
		coreSet.Spec.Replicas = ptr.To(int32(5))
		util.AttachAnnotations(coreSet, retirement.annotations())
		Expect(k8sClient.Update(ctx, coreSet)).To(Succeed())
		// Update container image:
		instance.Spec.Image = "emqx/emqx"
		Eventually(runRoundReconcile).WithArguments(round, instance, a).
			Should(BeSuccessfulReconcile())
		Expect(actualObject(instance)).To(And(
			HaveCondition(crd.Ready, HaveField("Status", Equal(metav1.ConditionFalse))),
			HaveCondition(crd.CoreNodesProgressing, HaveField("Status", Equal(metav1.ConditionTrue))),
		))
		Eventually(ensureReconcileState).WithArguments(round, instance).
			Should(Succeed())
		// Postconditions:
		// - container image is updated in place
		// - number of replicas is unchanged
		// - coreSet retirement annotations preserved
		Expect(listCoreSets()).To(ConsistOf(And(
			HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(5))),
			HaveField("Spec.Template.Spec.Containers", ConsistOf(
				HaveField("Image", Equal("emqx/emqx")))),
			HaveField("Annotations", And(
				HaveKeyWithValue(
					crd.AnnotationCoreRetirementOrdinalWatermark,
					"42",
				),
				HaveKeyWithValue(
					crd.AnnotationCoreRetirementOrdinalUpdatedAt,
					retirement.updatedAt.UTC().Format(time.RFC3339Nano),
				),
			)),
		)))
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})
})
