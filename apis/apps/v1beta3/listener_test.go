package v1beta3_test

import (
	"testing"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestListenerDefaultForWebhook(t *testing.T) {
	broker := &v1beta3.EmqxBroker{}
	broker.Default()

	assert.Equal(t, corev1.ServiceTypeClusterIP, broker.Spec.EmqxTemplate.Listener.Type)
	assert.Equal(t, int32(8081), broker.Spec.EmqxTemplate.Listener.API.Port)
	assert.Equal(t, int32(18083), broker.Spec.EmqxTemplate.Listener.Dashboard.Port)
	assert.Equal(t, int32(1883), broker.Spec.EmqxTemplate.Listener.MQTT.Port)
	assert.Equal(t, int32(8883), broker.Spec.EmqxTemplate.Listener.MQTTS.Port)
	assert.Equal(t, int32(8083), broker.Spec.EmqxTemplate.Listener.WS.Port)
	assert.Equal(t, int32(8084), broker.Spec.EmqxTemplate.Listener.WSS.Port)

	enterprise := &v1beta3.EmqxEnterprise{
		Spec: v1beta3.EmqxEnterpriseSpec{
			EmqxTemplate: v1beta3.EmqxEnterpriseTemplate{
				Listener: v1beta3.Listener{
					Type: corev1.ServiceTypeNodePort,
					MQTTS: v1beta3.Port{
						Port:     int32(8885),
						NodePort: int32(8885),
					},
				},
			},
		},
	}
	enterprise.Default()

	assert.Equal(t, corev1.ServiceTypeNodePort, enterprise.Spec.EmqxTemplate.Listener.Type)
	assert.Equal(t, int32(8081), enterprise.Spec.EmqxTemplate.Listener.API.Port)
	assert.Equal(t, int32(8885), enterprise.Spec.EmqxTemplate.Listener.MQTTS.Port)
	assert.Equal(t, int32(8885), enterprise.Spec.EmqxTemplate.Listener.MQTTS.NodePort)
	assert.Equal(t, int32(0), enterprise.Spec.EmqxTemplate.Listener.Dashboard.Port)
	assert.Equal(t, int32(0), enterprise.Spec.EmqxTemplate.Listener.MQTT.Port)
	assert.Equal(t, int32(0), enterprise.Spec.EmqxTemplate.Listener.WS.Port)
	assert.Equal(t, int32(0), enterprise.Spec.EmqxTemplate.Listener.WSS.Port)
}

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
