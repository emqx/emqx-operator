package v2alpha2

import (
	"context"
	"errors"

	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
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
	var req *innerReq.Requester = &innerReq.Requester{}
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
	})

	It("create namespace", func() {
		Expect(k8sClient.Create(context.TODO(), ns)).Should(Succeed())
	})

	It("should create statefulSet", func() {
		Eventually(a.reconcile(ctx, instance, req)).Should(Equal(subResult{}))
		Eventually(func() []appsv1.StatefulSet {
			list := &appsv1.StatefulSetList{}
			_ = k8sClient.List(ctx, list,
				client.InNamespace(instance.Namespace),
				client.MatchingLabels(instance.Spec.CoreTemplate.Labels),
			)
			return list.Items
		}).Should(And(
			HaveLen(1),
			ContainElements(
				WithTransform(func(s appsv1.StatefulSet) string { return s.Spec.Template.Spec.Containers[0].Image }, Equal(instance.Spec.Image)),
			),
		))
	})

	Context("change replicas count", func() {
		JustBeforeEach(func() {
			instance.Spec.CoreTemplate.Spec.Replicas = pointer.Int32(4)
		})

		It("should update statefulSet", func() {
			Eventually(a.reconcile(ctx, instance, req)).Should(Equal(subResult{}))
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
		})
	})

	Context("change image", func() {
		JustBeforeEach(func() {
			instance.Spec.Image = "emqx/emqx"
			instance.Spec.UpdateStrategy.InitialDelaySeconds = int32(999999999)
		})

		It("should create new statefulSet", func() {
			Eventually(a.reconcile(ctx, instance, req)).Should(Equal(subResult{}))
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
		})
	})

	Context("can be scale down", func() {
		var old *appsv1.StatefulSet = new(appsv1.StatefulSet)
		var realReplicas int32

		JustBeforeEach(func() {
			Eventually(func() error {
				list := getStateFulSetList(ctx, a.Client,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(instance.Spec.CoreTemplate.Labels),
				)
				if len(list) == 0 {
					return errors.New("not found")
				}
				old = list[0].DeepCopy()
				return nil
			}).Should(Succeed())

			realReplicas = *old.Spec.Replicas
			instance.Spec.UpdateStrategy.InitialDelaySeconds = int32(0)
			instance.Spec.UpdateStrategy.EvacuationStrategy.WaitTakeover = int32(0)
		})
		It("should scale down", func() {
			for realReplicas > 1 {
				//mock statefulSet status
				old.Status.Replicas = realReplicas
				old.Status.ReadyReplicas = realReplicas
				Expect(k8sClient.Status().Update(ctx, old)).Should(Succeed())
				Eventually(func() *appsv1.StatefulSet {
					_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(old), old)
					return old
				}).WithTimeout(timeout).WithPolling(interval).Should(And(
					WithTransform(func(s *appsv1.StatefulSet) int32 { return s.Status.Replicas }, Equal(realReplicas)),
					WithTransform(func(s *appsv1.StatefulSet) int32 { return s.Status.ReadyReplicas }, Equal(realReplicas)),
				))

				// retry it because update the statefulSet maybe will conflict
				Eventually(a.reconcile(ctx, instance, req)).WithTimeout(timeout).WithPolling(interval).Should(Equal(subResult{}))
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(old), old)
				Expect(*old.Spec.Replicas).Should(Equal(realReplicas - 1))

				realReplicas--
			}
		})
	})

	It("delete namespace", func() {
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})
})
