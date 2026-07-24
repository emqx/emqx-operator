package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = DescribeClientFaultMatrix("Reconciler addBootstrap", Ordered, func() {
	var ns *corev1.Namespace
	var instance *crd.EMQX

	BeforeAll(func() {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "controller-add-emqx-bootstrap-test",
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
	})

	AfterEach(func() {
		Expect(k8sClient.DeleteAllOf(ctx, &corev1.Secret{}, client.InNamespace(ns.Name))).
			Should(Succeed())
	})

	It("creates bootstrap secrets", func() {
		a := &addBootstrap{emqxReconciler()}
		Eventually(a.reconcile).WithArguments(newReconcileRound(), instance).
			Should(BeSuccessfulReconcile())

		cookieSecret := &corev1.Secret{}
		Expect(k8sClient.Get(ctx, instance.NodeCookieNamespacedName(), cookieSecret)).
			To(Succeed())
		Expect(cookieSecret.Data["node_cookie"]).ShouldNot(BeEmpty())

		bootstrapSecret := &corev1.Secret{}
		Expect(k8sClient.Get(ctx, instance.BootstrapAPIKeyNamespacedName(), bootstrapSecret)).
			To(Succeed())
		Expect(bootstrapSecret.Data["bootstrap_api_key"]).ShouldNot(BeEmpty())
	})
})
