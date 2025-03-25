package v2beta1

import (
	"time"

	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	. "github.com/emqx/emqx-operator/internal/test"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Check add core controller", Ordered, Label("core"), func() {
	var a *addCore
	var ns *corev1.Namespace
	var instance *appsv2beta1.EMQX

	BeforeEach(func() {
		a = &addCore{emqxReconciler}
		a.LoadEMQXConf(emqx)

		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-v2beta1-add-emqx-core-test",
				Labels: map[string]string{
					"test": "e2e",
				},
			},
		}

		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Status.Conditions = []metav1.Condition{
			{
				Type:               appsv2beta1.Ready,
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
			},
			{
				Type:               appsv2beta1.CoreNodesReady,
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
			},
		}
	})

	It("create namespace", func() {
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
	})

	It("create EMQX CR", func() {
		Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
	})

	It("should create statefulSet", func() {
		Eventually(a.reconcile(ctx, logger, instance, nil)).WithTimeout(timeout).WithPolling(interval).Should(Equal(subResult{}))
		Eventually(func() []appsv1.StatefulSet {
			list := &appsv1.StatefulSetList{}
			_ = k8sClient.List(ctx, list,
				client.InNamespace(instance.Namespace),
				client.MatchingLabels(appsv2beta1.DefaultCoreLabels(instance)),
			)
			return list.Items
		}).Should(ConsistOf(
			HaveField("Spec.Template.Spec.Containers", ConsistOf(HaveField("Image", Equal(instance.Spec.Image)))),
		))
	})

	It("change image creates new statefulSet", func() {
		instance.Spec.Image = "emqx/emqx"
		instance.Spec.UpdateStrategy.InitialDelaySeconds = int32(999999999)
		Eventually(a.reconcile(ctx, logger, instance, nil)).WithTimeout(timeout).WithPolling(interval).Should(Equal(subResult{}))
		Eventually(func() []appsv1.StatefulSet {
			list := &appsv1.StatefulSetList{}
			_ = k8sClient.List(ctx, list,
				client.InNamespace(instance.Namespace),
				client.MatchingLabels(appsv2beta1.DefaultCoreLabels(instance)),
			)
			return list.Items
		}).WithTimeout(timeout).WithPolling(interval).Should(ConsistOf(
			HaveField("Spec.Template.Spec.Containers", ConsistOf(HaveField("Image", Equal("emqx")))),
			HaveField("Spec.Template.Spec.Containers", ConsistOf(HaveField("Image", Equal("emqx/emqx")))),
		))

		Eventually(func() *appsv2beta1.EMQX {
			_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
			return instance
		}).Should(And(
			WithTransform(func(emqx *appsv2beta1.EMQX) string { return emqx.Status.GetLastTrueCondition().Type }, Equal(appsv2beta1.CoreNodesProgressing)),
			HaveCondition(appsv2beta1.Ready, BeNil()),
			HaveCondition(appsv2beta1.Available, BeNil()),
			HaveCondition(appsv2beta1.CoreNodesReady, BeNil()),
		))
	})

	It("delete namespace", func() {
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})

})
