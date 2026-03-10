package controller

import (
	"time"

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
				Name: "controller-v2beta1-add-emqx-core-test",
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
		// Mock instance status:
		instance.Status.Conditions = []metav1.Condition{
			{
				Type:               crdv2.Ready,
				Status:             metav1.ConditionTrue,
				Reason:             crdv2.Ready,
				LastTransitionTime: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
			},
			{
				Type:               crdv2.CoreNodesReady,
				Status:             metav1.ConditionTrue,
				Reason:             crdv2.CoreNodesReady,
				LastTransitionTime: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
			},
			{
				Type:               crdv2.Initialized,
				Status:             metav1.ConditionTrue,
				Reason:             crdv2.Initialized,
				LastTransitionTime: metav1.Time{Time: time.Now().AddDate(0, 0, -10)},
			},
		}
		// Instantiate reconciler:
		a = &addCoreSet{emqxReconciler}
		round = newReconcileRound()
		round.state = loadReconcileState(ctx, k8sClient, instance)
	})

	It("should create statefulSet", func() {
		Eventually(a.reconcile).WithArguments(round, instance).
			WithTimeout(timeout).
			WithPolling(interval).
			Should(Equal(subResult{}))

		Eventually(func() []appsv1.StatefulSet {
			list := &appsv1.StatefulSetList{}
			_ = k8sClient.List(ctx, list,
				client.InNamespace(instance.Namespace),
				client.MatchingLabels(instance.DefaultLabelsWith(crdv2.CoreLabels())),
			)
			return list.Items
		}).Should(ConsistOf(
			HaveField("Spec.Template.Spec.Containers", ConsistOf(HaveField("Image", Equal(instance.Spec.Image)))),
		))
	})

	It("change image updates existing statefulSet in place", func() {
		instance.Spec.Image = "emqx/emqx"
		instance.Spec.UpdateStrategy.InitialDelaySeconds = int32(999999999)
		Eventually(a.reconcile).WithArguments(round, instance).
			WithTimeout(timeout).
			WithPolling(interval).
			Should(Equal(subResult{}))

		Eventually(func() []appsv1.StatefulSet {
			list := &appsv1.StatefulSetList{}
			_ = k8sClient.List(ctx, list,
				client.InNamespace(instance.Namespace),
				client.MatchingLabels(instance.DefaultLabelsWith(crdv2.CoreLabels())),
			)
			return list.Items
		}).WithTimeout(timeout).WithPolling(interval).Should(ConsistOf(
			HaveField("Spec.Template.Spec.Containers", ConsistOf(HaveField("Image", Equal("emqx/emqx")))),
		))

		Eventually(func() *crdv2.EMQX {
			_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
			return instance
		}).Should(And(
			WithTransform(
				func(emqx *crdv2.EMQX) *metav1.Condition {
					return emqx.Status.GetLastTrueCondition()
				},
				HaveField("Type", Equal(crdv2.Initialized)),
			),
			HaveCondition(crdv2.Ready, HaveField("Status", Equal(metav1.ConditionFalse))),
			HaveCondition(crdv2.CoreNodesReady, HaveField("Status", Equal(metav1.ConditionFalse))),
		))
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})
})
