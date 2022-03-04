package v1beta3_test

import (
	"testing"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestListenerDefault(t *testing.T) {
	listener := &v1beta3.Listener{}
	listener.Default()

	assert.Equal(t, corev1.ServiceTypeClusterIP, listener.Type)
	assert.Equal(t, int32(8081), listener.API.Port)
	assert.Equal(t, int32(18083), listener.Dashboard.Port)
	assert.Equal(t, int32(1883), listener.MQTT.Port)
	assert.Equal(t, int32(8883), listener.MQTTS.Port)
	assert.Equal(t, int32(8083), listener.WS.Port)
	assert.Equal(t, int32(8084), listener.WSS.Port)

	listener = &v1beta3.Listener{
		Type: corev1.ServiceTypeNodePort,
		MQTTS: v1beta3.Port{
			Port:     int32(8885),
			NodePort: int32(8885),
		},
	}
	listener.Default()

	assert.Equal(t, corev1.ServiceTypeNodePort, listener.Type)
	assert.Equal(t, int32(8081), listener.API.Port)
	assert.Equal(t, int32(8885), listener.MQTTS.Port)
	assert.Equal(t, int32(8885), listener.MQTTS.NodePort)
	assert.Equal(t, int32(0), listener.Dashboard.Port)
	assert.Equal(t, int32(0), listener.MQTT.Port)
	assert.Equal(t, int32(0), listener.WS.Port)
	assert.Equal(t, int32(0), listener.WSS.Port)

}
