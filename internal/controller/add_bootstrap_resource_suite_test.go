package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	crd "github.com/emqx/emqx-operator/api/v3alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Reconciler addBootstrap", Ordered, func() {
	var instance *crd.EMQX = &crd.EMQX{}
	var ns *corev1.Namespace = &corev1.Namespace{}
	var a *addBootstrap

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

	AfterEach(func() {
		// Clean up bootstrap_api_key secret
		bootstrapSecret := &corev1.Secret{}
		err := k8sClient.Get(ctx, client.ObjectKey{
			Namespace: ns.Name,
			Name:      instance.BootstrapAPIKeyNamespacedName().Name,
		}, bootstrapSecret)
		if err == nil {
			// If the secret exists, delete it
			Expect(k8sClient.Delete(ctx, bootstrapSecret)).Should(Succeed())
		} else if !errors.IsNotFound(err) {
			// If the error is not a NotFound error, fail the test
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("create namespace", func() {
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
	})

	It("should create bootstrap secrets", func() {
		// Wait until the bootstrap secrets are created
		// Call the reconciler.
		result := a.reconcile(newReconcileRound(), instance)

		// Make sure there were no errors.
		Expect(result.err).NotTo(HaveOccurred())
		// Check the created secrets.
		cookieSecret := &corev1.Secret{}
		err := k8sClient.Get(ctx, client.ObjectKey{
			Namespace: ns.Name,
			Name:      instance.NodeCookieNamespacedName().Name,
		}, cookieSecret)
		Expect(err).NotTo(HaveOccurred())
		Expect(cookieSecret.Data["node_cookie"]).ShouldNot(BeEmpty())

		bootstrapSecret := &corev1.Secret{}
		err = k8sClient.Get(ctx, client.ObjectKey{
			Namespace: ns.Name,
			Name:      instance.BootstrapAPIKeyNamespacedName().Name,
		}, bootstrapSecret)
		Expect(err).NotTo(HaveOccurred())
		Expect(bootstrapSecret.Data["bootstrap_api_key"]).ShouldNot(BeEmpty())
	})

	It("should contain key and secret in bootstrap secret given initial values", func() {
		// Given
		instance.Spec.BootstrapAPIKeys = []crd.BootstrapAPIKey{
			{
				Key:    "test_key",
				Secret: "test_secret",
			},
		}

		// Call the reconciler.
		result := a.reconcile(newReconcileRound(), instance)

		// Make sure there were no errors.
		Expect(result.err).NotTo(HaveOccurred())

		// Check the created secrets.
		bootstrapSecret := &corev1.Secret{}
		err := k8sClient.Get(ctx, client.ObjectKey{
			Namespace: ns.Name,
			Name:      instance.BootstrapAPIKeyNamespacedName().Name,
		}, bootstrapSecret)
		Expect(err).NotTo(HaveOccurred())

		// Verify that the bootstrap API key contains the initial key and secret.
		Expect(string(bootstrapSecret.Data["bootstrap_api_key"])).Should(ContainSubstring("test_key:test_secret"))
	})

	It("should contain key and secret in bootstrap secret given SecretRef values", func() {
		// Given
		instance.Spec.BootstrapAPIKeys = []crd.BootstrapAPIKey{
			{
				SecretRef: &crd.SecretRef{
					Key: crd.KeyRef{
						// Note: a lowercase RFC 1123 subdomain must consist of lower case alphanumeric characters
						SecretName: "test-key-secret",
						SecretKey:  "key",
					},
					Secret: crd.KeyRef{
						SecretName: "test-value-secret",
						SecretKey:  "secret",
					},
				},
			},
		}

		// Create referenced secrets
		keySecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "test-key-secret",
			},
			StringData: map[string]string{
				"key": "test_key",
			},
		}
		Expect(k8sClient.Create(ctx, keySecret)).Should(Succeed())

		secretSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "test-value-secret",
			},
			StringData: map[string]string{
				"secret": "test_secret",
			},
		}
		Expect(k8sClient.Create(ctx, secretSecret)).Should(Succeed())

		// Call the reconciler.
		result := a.reconcile(newReconcileRound(), instance)

		// Make sure there were no errors.
		Expect(result.err).NotTo(HaveOccurred())

		// Check the created secrets.
		bootstrapSecret := &corev1.Secret{}
		err := k8sClient.Get(ctx, client.ObjectKey{
			Namespace: ns.Name,
			Name:      instance.BootstrapAPIKeyNamespacedName().Name,
		}, bootstrapSecret)
		Expect(err).NotTo(HaveOccurred())

		// Verify that the bootstrap API key contains the initial key and secret.
		Expect(string(bootstrapSecret.Data["bootstrap_api_key"])).Should(ContainSubstring("test_key:test_secret"))
	})
})
