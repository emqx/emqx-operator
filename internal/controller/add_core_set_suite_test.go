package controller

import (
	crdv2 "github.com/emqx/emqx-operator/api/v2"
	. "github.com/emqx/emqx-operator/test/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reconciler addCoreSet", Ordered, func() {
	var ns *corev1.Namespace
	var instance *crdv2.EMQX
	var a *addCoreSet
	var round *reconcileRound

	BeforeAll(func() {
		// Create namespace:
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-add-emqx-core-test",
				Labels: map[string]string{
					"test": "e2e",
				},
			},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
		// Create instance:
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
	})

	BeforeEach(func() {
		// Instantiate reconciler:
		a = &addCoreSet{emqxReconciler}
		round = newReconcileRound()
		Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
	})

	It("should create statefulSet", func() {
		result := a.reconcile(round, instance)
		Expect(result.err).ToNot(HaveOccurred())
		Expect(coreSets(instance)).To(ConsistOf(
			HaveField("Spec.Template.Spec.Containers", ConsistOf(
				HaveField("Image", Equal(instance.Spec.Image)))),
		))
	})

	It("change image updates existing statefulSet in place", func() {
		instance.Spec.Image = "emqx/emqx"
		instance.Spec.UpdateStrategy.MinReadySeconds = int32(999999999)
		result := a.reconcile(round, instance)
		Expect(result.err).ToNot(HaveOccurred())
		Expect(actualObject(instance)).To(And(
			HaveCondition(crdv2.Ready, HaveField("Status", Equal(metav1.ConditionFalse))),
			HaveCondition(crdv2.CoreNodesProgressing, HaveField("Status", Equal(metav1.ConditionTrue))),
		))
		Expect(coreSets(instance)).To(ConsistOf(
			HaveField("Spec.Template.Spec.Containers", ConsistOf(
				HaveField("Image", Equal("emqx/emqx")))),
		))
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})
})

func coreSets(instance *crdv2.EMQX) []appsv1.StatefulSet {
	list := &appsv1.StatefulSetList{}
	_ = k8sClient.List(ctx, list,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(instance.DefaultLabelsWith(crdv2.CoreLabels())),
	)
	return list.Items
}
