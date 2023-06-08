package v2alpha2

import (
	"strings"
	"testing"

	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenerateNodeCookieSecret(t *testing.T) {
	instance := &appsv2alpha2.EMQX{
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
		instance.Spec.BootstrapConfig = "node.cookie = fake"
		got := generateNodeCookieSecret(instance)
		assert.Equal(t, "emqx-node-cookie", got.Name)
		_, ok := got.StringData["node_cookie"]
		assert.True(t, ok)
		assert.Equal(t, "fake", got.StringData["node_cookie"])
	})
}

func TestGenerateBootstrapUserSecret(t *testing.T) {
	instance := &appsv2alpha2.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: appsv2alpha2.EMQXSpec{
			BootstrapAPIKeys: []appsv2alpha2.BootstrapAPIKey{
				{
					Key:    "test_key",
					Secret: "secret",
				},
			},
		},
	}

	got := generateBootstrapUserSecret(instance)
	assert.Equal(t, "emqx-bootstrap-user", got.Name)
	data, ok := got.StringData["bootstrap_user"]
	assert.True(t, ok)

	users := strings.Split(data, "\n")
	var usernames []string
	for _, user := range users {
		usernames = append(usernames, user[:strings.Index(user, ":")])
	}
	assert.ElementsMatch(t, usernames, []string{defUsername, "test_key"})
}

func TestGenerateBootstrapConfigMap(t *testing.T) {
	instance := &appsv2alpha2.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
	}

	got := generateBootstrapConfigMap(instance)
	assert.Equal(t, "emqx-bootstrap-config", got.Name)
	_, ok := got.Data["emqx.conf"]
	assert.True(t, ok)
}
