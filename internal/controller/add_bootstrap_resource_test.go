package controller

import (
	"strings"
	"testing"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	resources "github.com/emqx/emqx-operator/internal/controller/resources"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	instance := &crd.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
	}

	got := generateBootstrapAPIKeySecret(instance)
	assert.Equal(t, "emqx-bootstrap-api-key", got.Name)
	data, ok := got.StringData["bootstrap_api_key"]
	assert.True(t, ok)

	parts := strings.SplitN(data, ":", 2)
	assert.Equal(t, resources.DefaultBootstrapAPIKey, parts[0])
	assert.NotEmpty(t, parts[1])
}
