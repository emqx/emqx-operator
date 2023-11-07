package v2beta1

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

	AfterEach(func() {
		// Clean up bootstrap_api_key secret
		bootstrapSecret := &corev1.Secret{}
		err := k8sClient.Get(context.TODO(), client.ObjectKey{
			Namespace: ns.Name,
			Name:      instance.BootstrapAPIKeyNamespacedName().Name,
		}, bootstrapSecret)
		if err == nil {
			// If the secret exists, delete it
			Expect(k8sClient.Delete(context.TODO(), bootstrapSecret)).Should(Succeed())
		} else if !errors.IsNotFound(err) {
			// If the error is not a NotFound error, fail the test
			Expect(err).NotTo(HaveOccurred())
		}
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

	It("should contain key and secret in bootstrap secret given initial values", func() {
		// Given
		instance.Spec.BootstrapAPIKeys = []appsv2beta1.BootstrapAPIKey{
			{
				Key:    "test_key",
				Secret: "test_secret",
			},
		}

		// Call the reconciler.
		result := a.reconcile(ctx, instance, nil)

		// Make sure there were no errors.
		Expect(result.err).NotTo(HaveOccurred())

		// Check the created secrets.
		bootstrapSecret := &corev1.Secret{}
		err := k8sClient.Get(context.Background(), client.ObjectKey{
			Namespace: ns.Name,
			Name:      instance.BootstrapAPIKeyNamespacedName().Name,
		}, bootstrapSecret)
		Expect(err).NotTo(HaveOccurred())

		// Verify that the bootstrap API key contains the initial key and secret.
		Expect(string(bootstrapSecret.Data["bootstrap_api_key"])).Should(ContainSubstring("test_key:test_secret"))
	})

	It("should contain key and secret in bootstrap secret given SecretRef values", func() {
		// Given
		instance.Spec.BootstrapAPIKeys = []appsv2beta1.BootstrapAPIKey{
			{
				SecretRef: &appsv2beta1.SecretRef{
					Key: appsv2beta1.KeyRef{
						// Note: a lowercase RFC 1123 subdomain must consist of lower case alphanumeric characters
						SecretName: "test-key-secret",
						SecretKey:  "key",
					},
					Secret: appsv2beta1.KeyRef{
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
		Expect(k8sClient.Create(context.TODO(), keySecret)).Should(Succeed())

		secretSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      "test-value-secret",
			},
			StringData: map[string]string{
				"secret": "test_secret",
			},
		}
		Expect(k8sClient.Create(context.TODO(), secretSecret)).Should(Succeed())

		// Call the reconciler.
		result := a.reconcile(ctx, instance, nil)

		// Make sure there were no errors.
		Expect(result.err).NotTo(HaveOccurred())

		// Check the created secrets.
		bootstrapSecret := &corev1.Secret{}
		err := k8sClient.Get(context.Background(), client.ObjectKey{
			Namespace: ns.Name,
			Name:      instance.BootstrapAPIKeyNamespacedName().Name,
		}, bootstrapSecret)
		Expect(err).NotTo(HaveOccurred())

		// Verify that the bootstrap API key contains the initial key and secret.
		Expect(string(bootstrapSecret.Data["bootstrap_api_key"])).Should(ContainSubstring("test_key:test_secret"))
	})
})
