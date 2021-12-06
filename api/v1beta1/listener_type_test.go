package v1beta1_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/emqx/emqx-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

func TestGeneratListener(t *testing.T) {
	broker := v1beta1.EmqxBroker{}
	assert.NotNil(t, broker.GetListener())

	broker.Spec.Listener.Type = "NodePort"
	assert.Equal(t, broker.GetListener().Type, corev1.ServiceType("NodePort"))
	assert.NotNil(t, broker.GetListener().Ports)

	broker.Spec.Listener.Ports.MQTTS = 8884
	assert.Equal(t, broker.GetListener().Ports.MQTTS, int32(8884))
	assert.Equal(t, broker.GetListener().Ports.API, int32(8081))
}
