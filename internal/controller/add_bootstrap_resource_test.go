package controller

import (
	"strings"
	"testing"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	resources "github.com/emqx/emqx-operator/internal/controller/resources"
	"github.com/emqx/emqx-operator/internal/handler"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGenerateNodeCookieSecret(t *testing.T) {
	instance := &crd.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
	}

	t.Run("generate node cookie secret", func(t *testing.T) {
		conf, _ := config.EMQXConfig(instance.Spec.Config.Data)
		got := generateNodeCookieSecret(instance, conf)
		assert.Equal(t, "emqx-node-cookie", got.Name)
		_, ok := got.StringData["node_cookie"]
		assert.True(t, ok)
	})

	t.Run("generate node cookie when already set node cookie", func(t *testing.T) {
		instance.Spec.Config.Data = "node.cookie = fake"
		conf, _ := config.EMQXConfig(instance.Spec.Config.Data)
		got := generateNodeCookieSecret(instance, conf)
		assert.Equal(t, "emqx-node-cookie", got.Name)
		_, ok := got.StringData["node_cookie"]
		assert.True(t, ok)
		assert.Equal(t, "fake", got.StringData["node_cookie"])
	})
}

func TestGenerateBootstrapAPIKeySecret(t *testing.T) {
	// Create a fake client
	scheme := runtime.NewScheme()
	err := corev1.AddToScheme(scheme)
	if err != nil {
		t.Fatal(err)
	}

	a := &addBootstrap{
		EMQXReconciler: &EMQXReconciler{
			Handler: &handler.Handler{
				Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			},
		},
	}

	// Create a context
	ctx := ctx

	instance := &crd.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: crd.EMQXSpec{
			BootstrapAPIKeys: []crd.BootstrapAPIKey{
				{
					Key:    "test_key",
					Secret: "test_secret",
				},
			},
		},
	}

	str, err := a.getAPIKeyString(ctx, instance)
	if err != nil {
		t.Fatal(err)
	}

	got := generateBootstrapAPIKeySecret(instance, str)
	assert.Equal(t, "emqx-bootstrap-api-key", got.Name)
	data, ok := got.StringData["bootstrap_api_key"]
	assert.True(t, ok)

	users := strings.Split(data, "\n")
	usernames := make([]string, 0, len(users))
	secrets := make([]string, 0, len(users))
	for _, user := range users {
		usernames = append(usernames, user[:strings.Index(user, ":")])
		secrets = append(secrets, user[strings.Index(user, ":")+1:])
	}
	assert.ElementsMatch(t, usernames, []string{resources.DefaultBootstrapAPIKey, "test_key"})
	assert.Contains(t, secrets, "test_secret")
}

func TestGenerateBootstrapAPIKeySecretWithSecretRef(t *testing.T) {
	// Create a fake client
	scheme := runtime.NewScheme()
	err := corev1.AddToScheme(scheme)
	if err != nil {
		t.Fatal(err)
	}

	keySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-key-secret",
			Namespace: "emqx",
		},
		Data: map[string][]byte{
			"key": []byte("test_key"),
		},
	}
	valueSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-value-secret",
			Namespace: "emqx",
		},
		Data: map[string][]byte{
			"secret": []byte("test_secret"),
		},
	}

	a := &addBootstrap{
		EMQXReconciler: &EMQXReconciler{
			Handler: &handler.Handler{
				Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			},
		},
	}

	// Create a context
	ctx := ctx

	// Add secrets to the fake client
	err = a.Client.Create(ctx, keySecret)
	if err != nil {
		t.Fatal(err)
	}
	err = a.Client.Create(ctx, valueSecret)
	if err != nil {
		t.Fatal(err)
	}

	instance := &crd.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: crd.EMQXSpec{
			BootstrapAPIKeys: []crd.BootstrapAPIKey{
				{
					SecretRef: &crd.SecretRef{
						Key: crd.KeyRef{
							SecretName: "test-key-secret",
							SecretKey:  "key",
						},
						Secret: crd.KeyRef{
							SecretName: "test-value-secret",
							SecretKey:  "secret",
						},
					},
				},
			},
		},
	}

	str, err := a.getAPIKeyString(ctx, instance)
	if err != nil {
		t.Fatal(err)
	}

	got := generateBootstrapAPIKeySecret(instance, str)
	assert.Equal(t, "emqx-bootstrap-api-key", got.Name)
	data, ok := got.StringData["bootstrap_api_key"]
	assert.True(t, ok)

	users := strings.Split(data, "\n")
	usernames := make([]string, 0, len(users))
	secrets := make([]string, 0, len(users))
	for _, user := range users {
		usernames = append(usernames, user[:strings.Index(user, ":")])
		secrets = append(secrets, user[strings.Index(user, ":")+1:])
	}
	assert.ElementsMatch(t, usernames, []string{resources.DefaultBootstrapAPIKey, "test_key"})
	assert.Contains(t, secrets, "test_secret")
}

func TestReadSecret(t *testing.T) {
	// Create a fake client
	scheme := runtime.NewScheme()
	err := corev1.AddToScheme(scheme)
	if err != nil {
		t.Fatal(err)
	}

	// Define the secret data
	secretData := map[string][]byte{
		"key": []byte("value"),
	}

	// Create a secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: secretData,
	}

	a := &addBootstrap{
		EMQXReconciler: &EMQXReconciler{
			Handler: &handler.Handler{
				Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build(),
			},
		},
	}

	// Create a context
	instance := &crd.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "default",
		},
	}
	val, err := a.readSecret(ctx, instance, "test-secret", "key")
	if err != nil {
		t.Fatal(err)
	}

	// Check the secret value
	assert.Equal(t, "value", val)
}
