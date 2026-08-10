package controller

import (
	"time"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = DescribeClientFaultMatrix("Reconciler loadCoreSetRetirement", Ordered, func() {
	var ns *corev1.Namespace
	var instance *crd.EMQX

	mkCoreSet := func(name string, replicas int32) *appsv1.StatefulSet {
		labels := instance.DefaultLabelsWith(crd.CoreLabels())
		return &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: instance.Namespace,
				Labels:    labels,
			},
			Spec: appsv1.StatefulSetSpec{
				ServiceName: name,
				Replicas:    ptr.To(replicas),
				Selector:    &metav1.LabelSelector{MatchLabels: labels},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: labels},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "emqx", Image: "emqx"}},
					},
				},
			},
		}
	}

	BeforeAll(func() {
		ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			GenerateName: "controller-core-set-retirement-test-",
			Labels:       map[string]string{"test": "e2e"},
		}}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.DeleteAllOf(ctx, &appsv1.StatefulSet{}, client.InNamespace(ns.Name))).
			To(Succeed())
	})

	DescribeTable("initializes missing annotations",
		func(name string, annotations map[string]string) {
			coreSet := mkCoreSet(name, 3)
			coreSet.Annotations = annotations
			Expect(k8sClient.Create(ctx, coreSet)).To(Succeed())

			round := newReconcileRound()
			Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
			reconciler := &loadCoreSetRetirement{emqxReconciler()}
			Eventually(reconciler.reconcile).WithArguments(round, instance).
				Should(BeSuccessfulReconcile())

			actual, err := actualObject(coreSet)
			Expect(err).NotTo(HaveOccurred())
			state, err := readCoreSetRetirementState(actual)
			Expect(err).NotTo(HaveOccurred())
			Expect(state).NotTo(BeNil())
			Expect(state.watermark).To(Equal(int32(3)))
			Expect(state.updatedAt).NotTo(BeZero())
		},
		Entry("when both are absent", "emqx-core-missing-pair", nil),
		Entry("when the pair is incomplete", "emqx-core-incomplete-pair", map[string]string{
			crd.AnnotationCoreRetirementOrdinalWatermark: "1",
		}),
	)

	It("normalizes a watermark below the live replica count", func() {
		coreSet := mkCoreSet("emqx-core-low-watermark", 3)
		startedAt := time.Now().Add(-time.Hour).Truncate(time.Nanosecond)
		retirement := &coreSetRetirement{watermark: 1, updatedAt: startedAt}
		util.AttachAnnotations(coreSet, retirement.annotations())
		Expect(k8sClient.Create(ctx, coreSet)).To(Succeed())

		round := newReconcileRound()
		Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
		reconciler := &loadCoreSetRetirement{emqxReconciler()}
		Eventually(reconciler.reconcile).WithArguments(round, instance).
			Should(BeSuccessfulReconcile())

		actual, err := actualObject(coreSet)
		Expect(err).NotTo(HaveOccurred())
		state, err := readCoreSetRetirementState(actual)
		Expect(err).NotTo(HaveOccurred())
		Expect(state.watermark).To(Equal(int32(3)))
		Expect(state.updatedAt).To(BeTemporally(">", startedAt))
	})

	It("rejects invalid annotations", func() {
		coreSet := mkCoreSet("emqx-core-invalid-retirement-state", 3)
		coreSet.Annotations = map[string]string{
			crd.AnnotationCoreRetirementOrdinalWatermark: "invalid",
			crd.AnnotationCoreRetirementOrdinalUpdatedAt: "invalid",
		}
		Expect(k8sClient.Create(ctx, coreSet)).To(Succeed())

		round := newReconcileRound()
		Expect(reloadReconcileState(round, k8sClient, instance)).To(Succeed())
		reconciler := &loadCoreSetRetirement{emqxReconciler()}
		result := reconciler.reconcile(round, instance)
		Expect(result.err).To(MatchError(ContainSubstring("invalid reusable core pod ordinal watermark")))
		Expect(round.coreRetirement).To(BeNil())
	})
})
