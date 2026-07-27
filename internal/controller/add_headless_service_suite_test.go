package controller

import (
	crd "github.com/emqx/emqx-operator/api/v3beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = DescribeClientFaultMatrix("Reconciler addHeadlessService", Ordered, func() {
	var ns *corev1.Namespace
	var instance *crd.EMQX

	BeforeAll(func() {
		// Create namespace:
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "controller-add-service-test",
				Labels:       map[string]string{"test": "e2e"},
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
		instance.Spec.CoreTemplate = crd.EMQXCoreTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Labels: instance.DefaultLabelsWith(crd.CoreLabels()),
			},
		}
	})

	It("generates headless service", func() {
		a := addHeadlessService{emqxReconciler()}
		Eventually(a.reconcile).WithArguments(newReconcileRound(), instance).
			Should(BeSuccessfulReconcile())
		Eventually(func() *corev1.Service {
			svc := &corev1.Service{}
			_ = k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: "emqx-headless"}, svc)
			return svc
		}).Should(Not(BeNil()))
	})
})
