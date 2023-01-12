package v1beta4

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestMergeServicePorts(t *testing.T) {
	t.Run("duplicate name", func(t *testing.T) {
		ports1 := []corev1.ServicePort{
			{
				Name: "mqtt",
				Port: 1883,
			},
			{
				Name: "mqtts",
				Port: 8883,
			},
		}

		ports2 := []corev1.ServicePort{
			{
				Name: "mqtt",
				Port: 11883,
			},
			{
				Name: "ws",
				Port: 8083,
			},
		}

		assert.Equal(t, []corev1.ServicePort{
			{
				Name: "mqtt",
				Port: 1883,
			},
			{
				Name: "mqtts",
				Port: 8883,
			},
			{
				Name: "ws",
				Port: 8083,
			},
		}, MergeServicePorts(ports1, ports2))
	})

	t.Run("duplicate port", func(t *testing.T) {
		ports1 := []corev1.ServicePort{
			{
				Name: "mqtt",
				Port: 1883,
			},
			{
				Name: "mqtts",
				Port: 8883,
			},
		}
		ports2 := []corev1.ServicePort{
			{
				Name: "duplicate-mqtt",
				Port: 1883,
			},
			{
				Name: "ws",
				Port: 8083,
			},
		}
		assert.Equal(t, []corev1.ServicePort{
			{
				Name: "mqtt",
				Port: 1883,
			},
			{
				Name: "mqtts",
				Port: 8883,
			},
			{
				Name: "ws",
				Port: 8083,
			},
		}, MergeServicePorts(ports1, ports2))
	})
}
