package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Reconciler addBootstrap", Ordered, func() {
	var instance *crd.EMQX = &crd.EMQX{}
	var ns *corev1.Namespace = &corev1.Namespace{}

	BeforeAll(func() {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-add-emqx-bootstrap-test",
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
	})

	AfterEach(func() {
		bootstrapSecret := &corev1.Secret{}
		err := k8sClient.Get(ctx, client.ObjectKey{
			Namespace: ns.Name,
			Name:      instance.BootstrapAPIKeyNamespacedName().Name,
		}, bootstrapSecret)
		if err == nil {
			Expect(k8sClient.Delete(ctx, bootstrapSecret)).Should(Succeed())
		} else if !errors.IsNotFound(err) {
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("should create bootstrap secrets", func() {
		a := &addBootstrap{emqxReconciler}
		result := a.reconcile(newReconcileRound(), instance)
		Expect(result.err).NotTo(HaveOccurred())

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
