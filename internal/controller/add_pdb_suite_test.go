package controller

import (
	crdv2 "github.com/emqx/emqx-operator/api/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Reconciler addPdb", Ordered, func() {
	var ns *corev1.Namespace
	var instance *crdv2.EMQX

	BeforeAll(func() {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-v2-add-pdb-test",
				Labels: map[string]string{
					"test": "e2e",
				},
			},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
	})

	BeforeEach(func() {
		instance = emqx.DeepCopy()
		instance.Namespace = ns.Name
		instance.Spec.ReplicantTemplate = &crdv2.EMQXReplicantTemplate{
			Spec: crdv2.EMQXReplicantTemplateSpec{
				Replicas: ptr.To(int32(1)),
			},
		}
	})

	It("skips PDBs when minAvailable and maxUnavailable are both nil", func() {
		a := &addPdb{emqxReconciler}

		Eventually(a.reconcile).WithArguments(newReconcileRound(), instance).
			Should(Equal(subResult{}))
		Consistently(podDisruptionBudgets).WithArguments(instance).
			WithTimeout(interval * 3).
			WithPolling(interval).
			Should(BeEmpty())
	})

	It("creates and removes PDBs when transitioning from and to both nil", func() {
		a := &addPdb{emqxReconciler}

		maxUnavailable := intstr.FromInt32(1)
		minAvailable := intstr.FromString("50%")
		instance.Spec.CoreTemplate.Spec.MaxUnavailable = &maxUnavailable
		instance.Spec.ReplicantTemplate.Spec.MinAvailable = &minAvailable
		Eventually(a.reconcile).WithArguments(newReconcileRound(), instance).
			Should(Equal(subResult{}))

		Eventually(podDisruptionBudgets).WithArguments(instance).
			Should(ConsistOf(
				And(
					HaveField("Name", Equal(instance.CoreName())),
					HaveField("Spec.MaxUnavailable", HaveValue(Equal(maxUnavailable))),
				),
				And(
					HaveField("Name", Equal(instance.ReplicantName())),
					HaveField("Spec.MinAvailable", HaveValue(Equal(minAvailable))),
				),
			))

		instance.Spec.CoreTemplate.Spec.MaxUnavailable = nil
		instance.Spec.ReplicantTemplate.Spec.MinAvailable = nil
		Eventually(a.reconcile).WithArguments(newReconcileRound(), instance).
			Should(Equal(subResult{}))

		Eventually(podDisruptionBudgets).WithArguments(instance).
			Should(BeEmpty())
	})
})

func podDisruptionBudgets(instance *crdv2.EMQX) []policyv1.PodDisruptionBudget {
	list := &policyv1.PodDisruptionBudgetList{}
	_ = k8sClient.List(ctx, list,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(instance.DefaultLabels()),
	)
	return list.Items
}
