package v2alpha2

import (
	"context"
	"errors"

	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Check add repl controller", func() {
	var a *addRepl
	var req *innerReq.Requester = &innerReq.Requester{}
	var ns *corev1.Namespace = &corev1.Namespace{}

	BeforeEach(func() {
		a = &addRepl{emqxReconciler}

		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-v2alpah2-test-" + rand.String(5),
				Labels: map[string]string{
					"test": "e2e",
				},
			},
		}
		Expect(k8sClient.Create(context.TODO(), ns)).Should(Succeed())
	})
	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})

	Context("if replicant template is nil", func() {
		var instance *appsv2alpha2.EMQX = new(appsv2alpha2.EMQX)
		JustBeforeEach(func() {
			instance = emqx.DeepCopy()
			instance.Namespace = ns.Name
			instance.Spec.ReplicantTemplate = nil
		})

		It("should do nothing", func() {
			Eventually(a.reconcile(ctx, instance, req)).Should(Equal(subResult{}))
			Eventually(func() []appsv1.ReplicaSet {
				list := &appsv1.ReplicaSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(instance.Labels),
				)
				return list.Items
			}).Should(HaveLen(0))
		})
	})

	Context("if core nodes is not ready", func() {
		var instance *appsv2alpha2.EMQX = new(appsv2alpha2.EMQX)
		JustBeforeEach(func() {
			instance = emqx.DeepCopy()
			instance.Namespace = ns.Name
			instance.Status.RemoveCondition(appsv2alpha2.CodeNodesReady)
		})

		It("should do nothing", func() {
			instance := emqx.DeepCopy()
			instance.Status.RemoveCondition(appsv2alpha2.CodeNodesReady)

			Eventually(a.reconcile(ctx, instance, req)).Should(Equal(subResult{}))
			Eventually(func() []appsv1.ReplicaSet {
				list := &appsv1.ReplicaSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
				)
				return list.Items
			}).Should(HaveLen(0))
		})
	})

	Context("replicant template is not nil, and core code is ready", func() {
		var instance *appsv2alpha2.EMQX = new(appsv2alpha2.EMQX)
		JustBeforeEach(func() {
			instance = emqx.DeepCopy()
			instance.Namespace = ns.Name
		})

		It("should create replicaSet", func() {
			Eventually(a.reconcile(ctx, instance, req)).Should(Equal(subResult{}))
			Eventually(func() []appsv1.ReplicaSet {
				list := &appsv1.ReplicaSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
				)
				return list.Items
			}).Should(And(
				HaveLen(1),
				ContainElements(
					WithTransform(func(rs appsv1.ReplicaSet) string { return rs.Spec.Template.Spec.Containers[0].Image }, Equal(instance.Spec.Image)),
				),
			))
		})

		It("if change replica count, should update replicaSet", func() {
			Eventually(a.reconcile(ctx, instance, req)).Should(Equal(subResult{}))
			Eventually(func() []appsv1.ReplicaSet {
				list := &appsv1.ReplicaSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
				)
				return list.Items
			}).Should(And(
				HaveLen(1),
				ContainElements(
					WithTransform(func(rs appsv1.ReplicaSet) int32 { return *rs.Spec.Replicas }, Equal(int32(3))),
				),
			))

			By("update instance")
			instance.Spec.ReplicantTemplate.Spec.Replicas = pointer.Int32(5)
			Eventually(a.reconcile(ctx, instance, req)).Should(Equal(subResult{}))
			Eventually(func() []appsv1.ReplicaSet {
				list := &appsv1.ReplicaSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
				)
				return list.Items
			}).Should(And(
				HaveLen(1),
				ContainElements(
					WithTransform(func(rs appsv1.ReplicaSet) int32 { return *rs.Spec.Replicas }, Equal(int32(5))),
				),
			))

			By("update instance")
			instance.Spec.ReplicantTemplate.Spec.Replicas = pointer.Int32(0)
			Eventually(a.reconcile(ctx, instance, req)).Should(Equal(subResult{}))
			Eventually(func() []appsv1.ReplicaSet {
				list := &appsv1.ReplicaSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
				)
				return list.Items
			}).Should(And(
				HaveLen(1),
				ContainElements(
					WithTransform(func(rs appsv1.ReplicaSet) int32 { return *rs.Spec.Replicas }, Equal(int32(0))),
				),
			))
		})

		It("if change image, should create new replicaSet", func() {
			Eventually(a.reconcile(ctx, instance, req)).Should(Equal(subResult{}))
			Eventually(func() []appsv1.ReplicaSet {
				list := &appsv1.ReplicaSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
				)
				return list.Items
			}).Should(And(
				HaveLen(1),
				ContainElements(
					WithTransform(func(rs appsv1.ReplicaSet) string { return rs.Spec.Template.Spec.Containers[0].Image }, Equal(instance.Spec.Image)),
				),
			))

			By("update instance")
			instance.Spec.Image = "emqx/emqx"
			Eventually(a.reconcile(ctx, instance, req)).Should(Equal(subResult{}))
			Eventually(func() []appsv1.ReplicaSet {
				list := &appsv1.ReplicaSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
				)
				return list.Items
			}).Should(And(
				HaveLen(2),
				ContainElements(
					WithTransform(func(rs appsv1.ReplicaSet) string { return rs.Spec.Template.Spec.Containers[0].Image }, Equal(instance.Spec.Image)),
				),
			))
		})

	})

	Context("check can be scale up", func() {
		var instance *appsv2alpha2.EMQX = new(appsv2alpha2.EMQX)
		var old *appsv1.ReplicaSet = new(appsv1.ReplicaSet)
		JustBeforeEach(func() {
			instance = emqx.DeepCopy()
			instance.Namespace = ns.Name
			instance.Spec.UpdateStrategy = appsv2alpha2.UpdateStrategy{
				InitialDelaySeconds: 1,
				EvacuationStrategy: appsv2alpha2.EvacuationStrategy{
					WaitTakeover: 999999999,
				},
			}

			// create replicant
			Eventually(a.reconcile(ctx, instance, req)).Should(Equal(subResult{}))
			// get old replicant
			Eventually(func() error {
				list := &appsv1.ReplicaSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
				)
				if len(list.Items) == 0 {
					return errors.New("not found")
				}
				old = list.Items[0].DeepCopy()
				return nil
			}).Should(Succeed())

			// update replicant
			instance.Spec.Image = "emqx/emqx"
			Eventually(a.reconcile(ctx, instance, req)).Should(Equal(subResult{}))

			list := &appsv1.ReplicaSetList{}
			_ = k8sClient.List(ctx, list,
				client.InNamespace(instance.Namespace),
				client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
			)
			for _, rs := range list.Items {
				Eventually(func() error {
					rs.Status.Replicas = *rs.Spec.Replicas
					rs.Status.ReadyReplicas = *rs.Spec.Replicas
					return k8sClient.Status().Update(ctx, rs.DeepCopy())
				}).Should(Succeed())
			}
		})

		It("can be scale down", func() {
			// retry it because update the replicaSet maybe will conflict
			Eventually(a.reconcile(ctx, instance, req)).WithTimeout(timeout).WithPolling(interval).Should(Equal(subResult{}))
			Eventually(func() int32 {
				oldCP := old.DeepCopy()
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(oldCP), oldCP)
				return *oldCP.Spec.Replicas
			}).WithTimeout(timeout).WithPolling(interval).Should(Equal(int32(2)))
		})
	})
})
