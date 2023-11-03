package v2beta1

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("AddBootstrap", Ordered, Label("bootstrap"), func() {
	var (
		instance *appsv2beta1.EMQX
		a        *addBootstrap
		ns       *corev1.Namespace
	)
	instance = new(appsv2beta1.EMQX)
	ns = &corev1.Namespace{}

	BeforeEach(func() {
		a = &addBootstrap{emqxReconciler}
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-v2beta1-add-emqx-bootstrap-test",
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

	It("should create bootstrap secrets", func() {
		// Wait until the bootstrap secrets are created
		// Call the reconciler.
		result := a.reconcile(ctx, instance, nil)

		// Make sure there were no errors.
		Expect(result.err).NotTo(HaveOccurred())
		// Check the created secrets.
		cookieSecret := &corev1.Secret{}
		err := k8sClient.Get(context.Background(), client.ObjectKey{
			Namespace: ns.Name,
			Name:      instance.NodeCookieNamespacedName().Name,
		}, cookieSecret)
		Expect(err).NotTo(HaveOccurred())
		Expect(cookieSecret.Data["node_cookie"]).ShouldNot(BeEmpty())

		bootstrapSecret := &corev1.Secret{}
		err = k8sClient.Get(context.Background(), client.ObjectKey{
			Namespace: ns.Name,
			Name:      instance.BootstrapAPIKeyNamespacedName().Name,
		}, bootstrapSecret)
		Expect(err).NotTo(HaveOccurred())
		Expect(bootstrapSecret.Data["bootstrap_api_key"]).ShouldNot(BeEmpty())
	})
})
