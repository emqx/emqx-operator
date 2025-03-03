package controller

import (
	"time"

	appsv2beta1 "github.com/emqx/emqx-operator/api/v2beta1"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Check add core controller", Ordered, Label("core"), func() {
	var a *addCore
	var fakeR *innerReq.FakeRequester = &innerReq.FakeRequester{}
	var ns *corev1.Namespace = &corev1.Namespace{}

	var instance *appsv2beta1.EMQX = new(appsv2beta1.EMQX)

	BeforeEach(func() {
		a = &addCore{emqxReconciler}

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

	It("should create statefulSet", func() {
		Eventually(a.reconcile).WithArguments(ctx, logger, instance, fakeR).WithTimeout(timeout).WithPolling(interval).Should(Equal(subResult{}))
		Eventually(func() []appsv1.StatefulSet {
			list := &appsv1.StatefulSetList{}
			_ = k8sClient.List(ctx, list,
				client.InNamespace(instance.Namespace),
				client.MatchingLabels(appsv2beta1.DefaultCoreLabels(instance)),
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
					client.MatchingLabels(appsv2beta1.DefaultCoreLabels(instance)),
				)
				return list.Items
			}).Should(HaveLen(1))
			sts := list.Items[0].DeepCopy()
			sts.Status.Replicas = 2
			Expect(k8sClient.Status().Update(ctx, sts)).Should(Succeed())
			Eventually(func() *appsv1.StatefulSet {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(sts), sts)
				return sts
			}).WithTimeout(timeout).WithPolling(interval).Should(
				WithTransform(func(s *appsv1.StatefulSet) int32 { return s.Status.Replicas }, Equal(int32(2))),
			)

			instance.Status.CoreNodesStatus.UpdateRevision = sts.Labels[appsv2beta1.LabelsPodTemplateHashKey]
			instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(4))
		})

		It("should update statefulSet", func() {
			Eventually(a.reconcile).WithArguments(ctx, logger, instance, fakeR).WithTimeout(timeout).WithPolling(interval).Should(Equal(subResult{}))
			Eventually(func() []appsv1.StatefulSet {
				list := &appsv1.StatefulSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(appsv2beta1.DefaultCoreLabels(instance)),
				)
				return list.Items
			}).Should(ConsistOf(
				WithTransform(func(s appsv1.StatefulSet) int32 { return *s.Spec.Replicas }, Equal(*instance.Spec.CoreTemplate.Spec.Replicas)),
			))

			Eventually(func() *appsv2beta1.EMQX {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
				return instance
			}).Should(And(
				WithTransform(func(emqx *appsv2beta1.EMQX) string { return emqx.Status.GetLastTrueCondition().Type }, Equal(appsv2beta1.CoreNodesProgressing)),
				WithTransform(func(emqx *appsv2beta1.EMQX) bool { return emqx.Status.IsConditionTrue(appsv2beta1.Ready) }, BeFalse()),
				WithTransform(func(emqx *appsv2beta1.EMQX) bool { return emqx.Status.IsConditionTrue(appsv2beta1.Available) }, BeFalse()),
				WithTransform(func(emqx *appsv2beta1.EMQX) bool { return emqx.Status.IsConditionTrue(appsv2beta1.CoreNodesReady) }, BeFalse()),
			))
		})
	})

	Context("change image", func() {
		JustBeforeEach(func() {
			instance.Spec.Image = "emqx/emqx"
			instance.Spec.UpdateStrategy.InitialDelaySeconds = int32(999999999)
		})

		It("should create new statefulSet", func() {
			Eventually(a.reconcile).WithArguments(ctx, logger, instance, fakeR).WithTimeout(timeout).WithPolling(interval).Should(Equal(subResult{}))
			Eventually(func() []appsv1.StatefulSet {
				list := &appsv1.StatefulSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(appsv2beta1.DefaultCoreLabels(instance)),
				)
				return list.Items
			}).WithTimeout(timeout).WithPolling(interval).Should(ConsistOf(
				WithTransform(func(s appsv1.StatefulSet) string { return s.Spec.Template.Spec.Containers[0].Image }, Equal(emqx.Spec.Image)),
				WithTransform(func(s appsv1.StatefulSet) string { return s.Spec.Template.Spec.Containers[0].Image }, Equal(instance.Spec.Image)),
			))

			Eventually(func() *appsv2beta1.EMQX {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
				return instance
			}).Should(And(
				WithTransform(func(emqx *appsv2beta1.EMQX) string { return emqx.Status.GetLastTrueCondition().Type }, Equal(appsv2beta1.CoreNodesProgressing)),
				WithTransform(func(emqx *appsv2beta1.EMQX) bool { return emqx.Status.IsConditionTrue(appsv2beta1.Ready) }, BeFalse()),
				WithTransform(func(emqx *appsv2beta1.EMQX) bool { return emqx.Status.IsConditionTrue(appsv2beta1.Available) }, BeFalse()),
				WithTransform(func(emqx *appsv2beta1.EMQX) bool { return emqx.Status.IsConditionTrue(appsv2beta1.CoreNodesReady) }, BeFalse()),
			))
		})
	})

	It("delete namespace", func() {
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})
})
