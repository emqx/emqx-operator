package v2beta1

import (
	"context"
	"strings"
	"testing"

	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGenerateNodeCookieSecret(t *testing.T) {
	instance := &appsv2beta1.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
	}

	t.Run("generate node cookie secret", func(t *testing.T) {
		got := generateNodeCookieSecret(instance)
		assert.Equal(t, "emqx-node-cookie", got.Name)
		_, ok := got.StringData["node_cookie"]
		assert.True(t, ok)
	})

	t.Run("generate node cookie when already set node cookie", func(t *testing.T) {
		instance.Spec.Config.Data = "node.cookie = fake"
		got := generateNodeCookieSecret(instance)
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

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Create a context
	ctx := context.Background()

	instance := &appsv2beta1.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: appsv2beta1.EMQXSpec{
			BootstrapAPIKeys: []appsv2beta1.BootstrapAPIKey{
				{
					Key:    "test_key",
					Secret: "secret",
				},
			},
		},
	}

	got := generateBootstrapAPIKeySecret(fakeClient, ctx, instance)
	assert.Equal(t, "emqx-bootstrap-api-key", got.Name)
	data, ok := got.StringData["bootstrap_api_key"]
	assert.True(t, ok)

	users := strings.Split(data, "\n")
	var usernames []string
	for _, user := range users {
		usernames = append(usernames, user[:strings.Index(user, ":")])
	}
	assert.ElementsMatch(t, usernames, []string{appsv2beta1.DefaultBootstrapAPIKey, "test_key"})
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

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	// Create a context
	ctx := context.Background()

	val, err := ReadSecret(fakeClient, ctx, "default", "test-secret", "key")
	if err != nil {
		t.Fatal(err)
	}

	// Check the secret value
	assert.Equal(t, "value", val)
}
