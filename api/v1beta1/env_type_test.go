package v1beta1_test

import (
	"testing"

	"github.com/emqx/emqx-operator/api/v1beta1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestGenerateEnv(t *testing.T) {
	env := []corev1.EnvVar{
		{
			Name:  "EMQX_NAME",
			Value: "foo",
		},
		{
			Name:  "EMQX_CLUSTER__K8S__NAMESPACE",
			Value: "bar",
		},
		{
			Name:  "EMQX_FOO",
			Value: "bar",
		},
	}

	emqxBroker := v1beta1.EmqxBroker{
		Spec: v1beta1.EmqxBrokerSpec{
			Env: env,
		},
	}

	assert.Contains(t,
		emqxBroker.GetEnv(),
		corev1.EnvVar{
			Name:  "EMQX_NAME",
			Value: "foo",
		},
	)
	assert.Contains(t,
		emqxBroker.GetEnv(),
		corev1.EnvVar{
			Name:  "EMQX_CLUSTER__K8S__NAMESPACE",
			Value: "bar",
		},
	)
	assert.Contains(t,
		emqxBroker.GetEnv(),
		corev1.EnvVar{
			Name:  "EMQX_FOO",
			Value: "bar",
		},
	)
}
