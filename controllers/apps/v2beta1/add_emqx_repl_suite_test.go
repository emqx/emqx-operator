package v2beta1

import (
	"context"
	"time"

	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Check add repl controller", Ordered, Label("repl"), func() {
	var a *addRepl
	var instance *appsv2beta1.EMQX = new(appsv2beta1.EMQX)
	var ns *corev1.Namespace = &corev1.Namespace{}

	BeforeEach(func() {
		a = &addRepl{emqxReconciler}

		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-v2beta1-add-emqx-repl-test",
				Labels: map[string]string{
					"test": "e2e",
				},
			},
		}

		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.ReplicantTemplate = &appsv2beta1.EMQXReplicantTemplate{
			Spec: appsv2beta1.EMQXReplicantTemplateSpec{
				Replicas: pointer.Int32(3),
			},
		}
		instance.Status = appsv2beta1.EMQXStatus{
			ReplicantNodesStatus: &appsv2beta1.EMQXNodesStatus{
				Replicas: 3,
			},
			Conditions: []metav1.Condition{
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
			},
		}
		instance.Default()
	})

	It("create namespace", func() {
		Expect(k8sClient.Create(context.TODO(), ns)).Should(Succeed())
	})

	Context("replicant template is nil", func() {
		JustBeforeEach(func() {
			instance.Spec.ReplicantTemplate = nil
		})

		It("should do nothing", func() {
			Eventually(a.reconcile(ctx, instance, nil)).WithTimeout(timeout).WithPolling(interval).Should(Equal(subResult{}))
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

	Context("core nodes is not ready", func() {
		JustBeforeEach(func() {
			instance.Status.RemoveCondition(appsv2beta1.CoreNodesReady)
		})

		It("should do nothing", func() {
			Eventually(a.reconcile(ctx, instance, nil)).WithTimeout(timeout).WithPolling(interval).Should(Equal(subResult{}))
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
		It("should create replicaSet", func() {
			Eventually(a.reconcile(ctx, instance, nil)).WithTimeout(timeout).WithPolling(interval).Should(Equal(subResult{}))
			Eventually(func() []appsv1.ReplicaSet {
				list := &appsv1.ReplicaSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
				)
				return list.Items
			}).Should(ConsistOf(
				WithTransform(func(rs appsv1.ReplicaSet) string { return rs.Spec.Template.Spec.Containers[0].Image }, Equal(instance.Spec.Image)),
			))
		})
	})

	Context("scale down replicas count", func() {
		JustBeforeEach(func() {
			list := &appsv1.ReplicaSetList{}
			Eventually(func() []appsv1.ReplicaSet {
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
				)
				return list.Items
			}).Should(HaveLen(1))
			rs := list.Items[0].DeepCopy()
			rs.Status.Replicas = 3
			Expect(k8sClient.Status().Update(ctx, rs)).Should(Succeed())
			Eventually(func() *appsv1.ReplicaSet {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(rs), rs)
				return rs
			}).WithTimeout(timeout).WithPolling(interval).Should(
				WithTransform(func(s *appsv1.ReplicaSet) int32 { return s.Status.Replicas }, Equal(int32(3))),
			)

			instance.Status.ReplicantNodesStatus.UpdateRevision = rs.Labels[appsv2beta1.LabelsPodTemplateHashKey]
			instance.Spec.ReplicantTemplate.Spec.Replicas = pointer.Int32(0)
		})

		It("should update replicaSet", func() {
			Eventually(a.reconcile(ctx, instance, nil)).WithTimeout(timeout).WithPolling(interval).WithTimeout(timeout).WithPolling(interval).Should(Equal(subResult{}))
			Eventually(func() []appsv1.ReplicaSet {
				list := &appsv1.ReplicaSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
				)
				return list.Items
			}).Should(ConsistOf(
				WithTransform(func(rs appsv1.ReplicaSet) int32 { return *rs.Spec.Replicas }, Equal(*instance.Spec.ReplicantTemplate.Spec.Replicas)),
			))
		})
	})

	Context("scale up replicas count", func() {
		JustBeforeEach(func() {
			list := &appsv1.ReplicaSetList{}
			Eventually(func() []appsv1.ReplicaSet {
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
				)
				return list.Items
			}).Should(HaveLen(1))
			instance.Status.ReplicantNodesStatus.UpdateRevision = list.Items[0].Labels[appsv2beta1.LabelsPodTemplateHashKey]

			instance.Spec.ReplicantTemplate.Spec.Replicas = pointer.Int32(4)
		})

		It("should update replicaSet", func() {
			Eventually(a.reconcile(ctx, instance, nil)).WithTimeout(timeout).WithPolling(interval).Should(Equal(subResult{}))
			Eventually(func() []appsv1.ReplicaSet {
				list := &appsv1.ReplicaSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
				)
				return list.Items
			}).Should(ConsistOf(
				WithTransform(func(rs appsv1.ReplicaSet) int32 { return *rs.Spec.Replicas }, Equal(*instance.Spec.ReplicantTemplate.Spec.Replicas)),
			))

			Eventually(func() *appsv2beta1.EMQX {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
				return instance
			}).Should(And(
				WithTransform(func(emqx *appsv2beta1.EMQX) string { return emqx.Status.GetLastTrueCondition().Type }, Equal(appsv2beta1.ReplicantNodesProgressing)),
				WithTransform(func(emqx *appsv2beta1.EMQX) bool { return emqx.Status.IsConditionTrue(appsv2beta1.Ready) }, BeFalse()),
				WithTransform(func(emqx *appsv2beta1.EMQX) bool { return emqx.Status.IsConditionTrue(appsv2beta1.Available) }, BeFalse()),
				WithTransform(func(emqx *appsv2beta1.EMQX) bool {
					return emqx.Status.IsConditionTrue(appsv2beta1.ReplicantNodesReady)
				}, BeFalse()),
			))
		})
	})

	Context("change image", func() {
		JustBeforeEach(func() {
			instance.Spec.Image = "emqx/emqx"
			instance.Spec.UpdateStrategy.InitialDelaySeconds = int32(999999999)
		})

		It("should create new replicaSet", func() {
			Eventually(a.reconcile(ctx, instance, nil)).WithTimeout(timeout).WithPolling(interval).Should(Equal(subResult{}))
			Eventually(func() []appsv1.ReplicaSet {
				list := &appsv1.ReplicaSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
				)
				return list.Items
			}).Should(ConsistOf(
				WithTransform(func(rs appsv1.ReplicaSet) string { return rs.Spec.Template.Spec.Containers[0].Image }, Equal(emqx.Spec.Image)),
				WithTransform(func(rs appsv1.ReplicaSet) string { return rs.Spec.Template.Spec.Containers[0].Image }, Equal(instance.Spec.Image)),
			))

			Eventually(func() *appsv2beta1.EMQX {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
				return instance
			}).Should(And(
				WithTransform(func(emqx *appsv2beta1.EMQX) string { return emqx.Status.GetLastTrueCondition().Type }, Equal(appsv2beta1.ReplicantNodesProgressing)),
				WithTransform(func(emqx *appsv2beta1.EMQX) bool { return emqx.Status.IsConditionTrue(appsv2beta1.Ready) }, BeFalse()),
				WithTransform(func(emqx *appsv2beta1.EMQX) bool { return emqx.Status.IsConditionTrue(appsv2beta1.Available) }, BeFalse()),
				WithTransform(func(emqx *appsv2beta1.EMQX) bool {
					return emqx.Status.IsConditionTrue(appsv2beta1.ReplicantNodesReady)
				}, BeFalse()),
			))
		})
	})

	It("delete namespace", func() {
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})
})
