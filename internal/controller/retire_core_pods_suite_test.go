package controller

import (
	"fmt"
	"time"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	"github.com/emqx/emqx-operator/internal/emqx/api"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = DescribeClientFaultMatrix("Reconciler retireCorePods", Ordered, func() {
	var ns *corev1.Namespace
	var instance *crd.EMQX

	var pvcSpec = corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		Resources: corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{
			corev1.ResourceStorage: resource.MustParse("1Mi"),
		}},
	}

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
			GenerateName: "controller-retire-core-pods-test-",
			Labels:       map[string]string{"test": "e2e"},
		}}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Status.CoreNodes = []crd.EMQXNode{{
			Name: "emqx@surviving-core", PodName: "surviving-core", Role: crd.RoleCore,
		}}
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
	})

	DescribeTable("advances the watermark only after PVC deletion",
		func(name string, pvcPresent bool, expectedWatermark int32, expectRequeue bool) {
			coreSet := mkCoreSet(name, 4)
			coreSet.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{{
				ObjectMeta: metav1.ObjectMeta{Name: "emqx-core-data"},
				Spec:       pvcSpec,
			}}
			util.AttachAnnotations(coreSet, newCoreSetRetirementState(5).annotations())
			Expect(k8sClient.Create(ctx, coreSet)).To(Succeed())
			DeferCleanupObject(coreSet)

			if pvcPresent {
				podName := fmt.Sprintf("%s-4", coreSet.Name)
				pvc := &corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{Name: "emqx-core-data-" + podName, Namespace: ns.Name},
					Spec:       pvcSpec,
				}
				Expect(k8sClient.Create(ctx, pvc)).To(Succeed())
				DeferCleanupObject(pvc)
			}

			round := newReconcileRound()
			round.dsCluster = &api.DSCluster{}
			reconciler := &retireCorePods{emqxReconciler()}
			Eventually(runRoundReconcile).WithArguments(round, instance, reconciler).
				Should(BeSuccessfulReconcile(Postponed(expectRequeue)))

			actual, err := actualObject(coreSet)
			Expect(err).NotTo(HaveOccurred())
			state, err := readCoreSetRetirementState(actual)
			Expect(err).NotTo(HaveOccurred())
			Expect(state.watermark).To(Equal(expectedWatermark))
		},
		Entry("when the PVC is absent", "emqx-core-pvc-absent", false, int32(4), false),
		Entry("waits while the PVC exists", "emqx-core-pvc-present", true, int32(5), true),
	)

	It("drains the highest ordinal and preserves the timestamp", func() {
		coreSet := mkCoreSet("emqx-core-drain-highest", 3)
		coreSet.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{{
			ObjectMeta: metav1.ObjectMeta{Name: "emqx-core-data"},
			Spec:       pvcSpec,
		}}
		startedAt := time.Now().Add(-corePodForcedRetirementTimeout[condFallback] - time.Second)
		retirement := &coreSetRetirement{5, startedAt}
		util.AttachAnnotations(coreSet, retirement.annotations())
		Expect(k8sClient.Create(ctx, coreSet)).To(Succeed())
		DeferCleanupObject(coreSet)

		round := newReconcileRound()
		reconciler := &retireCorePods{emqxReconciler()}
		Eventually(runRoundReconcile).WithArguments(round, instance, reconciler).
			Should(BeSuccessfulReconcile())

		actual, err := actualObject(coreSet)
		Expect(err).NotTo(HaveOccurred())
		state, err := readCoreSetRetirementState(actual)
		Expect(err).NotTo(HaveOccurred())
		Expect(state.watermark).To(Equal(int32(4)))
		Expect(state.updatedAt).To(Equal(startedAt.UTC()))
	})

	It("waits for pod absence without deleting the pod", func() {
		coreSet := mkCoreSet("emqx-core-pod-present", 4)
		coreSet.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{{
			ObjectMeta: metav1.ObjectMeta{Name: "emqx-core-data"},
			Spec:       pvcSpec,
		}}
		startedAt := time.Now().Add(-corePodForcedRetirementTimeout[condFallback] - time.Second)
		retirement := &coreSetRetirement{5, startedAt}
		util.AttachAnnotations(coreSet, retirement.annotations())
		Expect(k8sClient.Create(ctx, coreSet)).To(Succeed())
		DeferCleanupObject(coreSet)

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      coreSet.Name + "-4",
				Namespace: ns.Name,
				Labels:    instance.DefaultLabelsWith(crd.CoreLabels()),
			},
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "emqx", Image: "emqx"}}},
		}
		Expect(k8sClient.Create(ctx, pod)).To(Succeed())
		DeferCleanupObject(pod)

		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "emqx-core-data-" + pod.Name, Namespace: ns.Name},
			Spec:       pvcSpec,
		}
		Expect(k8sClient.Create(ctx, pvc)).To(Succeed())
		DeferCleanupObject(pvc)

		round := newReconcileRound()
		Expect(ensureReconcileState(round, instance)).To(Succeed())
		reconciler := &retireCorePods{emqxReconciler()}
		result := reconciler.reconcile(round, instance)

		Expect(result).To(BeSuccessfulReconcile())
		Expect(result.needRequeue).To(BeTrue())
		Expect(actualObject(pod)).NotTo(BeNil())
		actual, err := actualObject(coreSet)
		Expect(err).NotTo(HaveOccurred())
		state, err := readCoreSetRetirementState(actual)
		Expect(err).NotTo(HaveOccurred())
		Expect(state.watermark).To(Equal(int32(5)))
	})
})
