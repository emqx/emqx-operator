package v2alpha2

import (
	"context"
	"time"

	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Check add core controller", Ordered, Label("core"), func() {
	var a *addCore
	var ns *corev1.Namespace = &corev1.Namespace{}

	var instance *appsv2alpha2.EMQX = new(appsv2alpha2.EMQX)

	BeforeEach(func() {
		a = &addCore{emqxReconciler}

		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-v2alpha2-add-emqx-core-test",
				Labels: map[string]string{
					"test": "e2e",
				},
			},
		}

		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Status.Conditions = []metav1.Condition{
			{
				Type:               appsv2alpha2.Ready,
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
			},
			{
				Type:               appsv2alpha2.CoreNodesReady,
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
			},
		}
	})

	It("create namespace", func() {
		Expect(k8sClient.Create(context.TODO(), ns)).Should(Succeed())
	})

	It("should create statefulSet", func() {
		Eventually(a.reconcile(ctx, instance, nil)).Should(Equal(subResult{}))
		Eventually(func() []appsv1.StatefulSet {
			list := &appsv1.StatefulSetList{}
			_ = k8sClient.List(ctx, list,
				client.InNamespace(instance.Namespace),
				client.MatchingLabels(instance.Spec.CoreTemplate.Labels),
			)
			return list.Items
		}).Should(ConsistOf(
			WithTransform(func(s appsv1.StatefulSet) string { return s.Spec.Template.Spec.Containers[0].Image }, Equal(instance.Spec.Image)),
		))
	})

	Context("change replicas count", func() {
		JustBeforeEach(func() {
			list := &appsv1.StatefulSetList{}
			Eventually(func() []appsv1.StatefulSet {
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(instance.Spec.CoreTemplate.Labels),
				)
				return list.Items
			}).Should(HaveLen(1))
			instance.Status.CoreNodesStatus.CurrentRevision = list.Items[0].Labels[appsv2alpha2.PodTemplateHashLabelKey]

			instance.Spec.CoreTemplate.Spec.Replicas = pointer.Int32(4)
		})

		It("should update statefulSet", func() {
			Eventually(a.reconcile(ctx, instance, nil)).Should(Equal(subResult{}))
			Eventually(func() []appsv1.StatefulSet {
				list := &appsv1.StatefulSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(instance.Spec.CoreTemplate.Labels),
				)
				return list.Items
			}).Should(ConsistOf(
				WithTransform(func(s appsv1.StatefulSet) int32 { return *s.Spec.Replicas }, Equal(*instance.Spec.CoreTemplate.Spec.Replicas)),
			))

			Eventually(func() *appsv2alpha2.EMQX {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
				return instance
			}).Should(And(
				WithTransform(func(emqx *appsv2alpha2.EMQX) string { return emqx.Status.GetLastTrueCondition().Type }, Equal(appsv2alpha2.CoreNodesProgressing)),
				WithTransform(func(emqx *appsv2alpha2.EMQX) bool { return emqx.Status.IsConditionTrue(appsv2alpha2.Ready) }, BeFalse()),
				WithTransform(func(emqx *appsv2alpha2.EMQX) bool { return emqx.Status.IsConditionTrue(appsv2alpha2.Available) }, BeFalse()),
				WithTransform(func(emqx *appsv2alpha2.EMQX) bool { return emqx.Status.IsConditionTrue(appsv2alpha2.CoreNodesReady) }, BeFalse()),
			))
		})
	})

	Context("change image", func() {
		JustBeforeEach(func() {
			instance.Spec.Image = "emqx/emqx"
			instance.Spec.UpdateStrategy.InitialDelaySeconds = int32(999999999)
		})

		It("should create new statefulSet", func() {
			Eventually(a.reconcile(ctx, instance, nil)).Should(Equal(subResult{}))
			Eventually(func() []appsv1.StatefulSet {
				list := &appsv1.StatefulSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(instance.Spec.CoreTemplate.Labels),
				)
				return list.Items
			}).WithTimeout(timeout).WithPolling(interval).Should(ConsistOf(
				WithTransform(func(s appsv1.StatefulSet) string { return s.Spec.Template.Spec.Containers[0].Image }, Equal(emqx.Spec.Image)),
				WithTransform(func(s appsv1.StatefulSet) string { return s.Spec.Template.Spec.Containers[0].Image }, Equal(instance.Spec.Image)),
			))

			Eventually(func() *appsv2alpha2.EMQX {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
				return instance
			}).Should(And(
				WithTransform(func(emqx *appsv2alpha2.EMQX) string { return emqx.Status.GetLastTrueCondition().Type }, Equal(appsv2alpha2.CoreNodesProgressing)),
				WithTransform(func(emqx *appsv2alpha2.EMQX) bool { return emqx.Status.IsConditionTrue(appsv2alpha2.Ready) }, BeFalse()),
				WithTransform(func(emqx *appsv2alpha2.EMQX) bool { return emqx.Status.IsConditionTrue(appsv2alpha2.Available) }, BeFalse()),
				WithTransform(func(emqx *appsv2alpha2.EMQX) bool { return emqx.Status.IsConditionTrue(appsv2alpha2.CoreNodesReady) }, BeFalse()),
			))
		})
	})

	It("delete namespace", func() {
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})
})
