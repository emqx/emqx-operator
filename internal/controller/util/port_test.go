package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestParseBindPort(t *testing.T) {
	tests := []struct {
		name string
		bind string
		port int32
	}{
		{name: "bare port", bind: "1883", port: 1883},
		{name: "hostname", bind: "mqtt.example:8883", port: 8883},
		{name: "IPv4", bind: "0.0.0.0:8083", port: 8083},
		{name: "bracketed IPv6", bind: "[::]:8084", port: 8084},
		{name: "zero", bind: "0", port: 0},
		{name: "outside Kubernetes range", bind: "65536", port: 65536},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, err := ParseBindPort(tt.bind)
			assert.NoError(t, err)
			assert.Equal(t, tt.port, port)
		})
	}
}

func TestParseBindPortErrors(t *testing.T) {
	tests := []struct {
		name     string
		bind     string
		contains string
	}{
		{name: "missing", bind: "", contains: "missing port"},
		{name: "non-numeric", bind: "127.0.0.1", contains: `non-numeric port "127.0.0.1"`},
		{name: "host with non-numeric port", bind: "0.0.0.0:nope", contains: `non-numeric port "nope"`},
		{name: "unbracketed IPv6", bind: "::1:1883", contains: "too many colons"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBindPort(tt.bind)
			assert.ErrorContains(t, err, tt.contains)
		})
	}
}

func TestAppendMissingServicePorts(t *testing.T) {
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
		}, AppendMissingServicePorts(ports1, ports2))
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
				Name: "duplicate-mqtt",
				Port: 1883,
			},
			{
				Name: "ws",
				Port: 8083,
			},
		}, AppendMissingServicePorts(ports1, ports2))
	})

	t.Run("preserve duplicates within second slice", func(t *testing.T) {
		ports1 := []corev1.ServicePort{{Name: "custom", Port: 1883}}
		ports2 := []corev1.ServicePort{
			{Name: "custom", Port: 1884},
			{Name: "tcp-a", Port: 1883},
			{Name: "tcp-duplicate", Port: 1885},
			{Name: "tcp-duplicate", Port: 1886},
		}
		assert.Equal(t, []corev1.ServicePort{
			{Name: "custom", Port: 1883},
			{Name: "tcp-a", Port: 1883},
			{Name: "tcp-duplicate", Port: 1885},
			{Name: "tcp-duplicate", Port: 1886},
		}, AppendMissingServicePorts(ports1, ports2))
	})
}

func TestAppendMissingContainerPorts(t *testing.T) {
	ports1 := []corev1.ContainerPort{{Name: "custom", ContainerPort: 1883}}
	ports2 := []corev1.ContainerPort{
		{Name: "custom", ContainerPort: 1884},
		{Name: "tcp-a", ContainerPort: 1883},
		{Name: "tcp-duplicate", ContainerPort: 1885},
		{Name: "tcp-duplicate", ContainerPort: 1886},
	}
	assert.Equal(t, []corev1.ContainerPort{
		{Name: "custom", ContainerPort: 1883},
		{Name: "tcp-a", ContainerPort: 1883},
		{Name: "tcp-duplicate", ContainerPort: 1885},
		{Name: "tcp-duplicate", ContainerPort: 1886},
	}, AppendMissingContainerPorts(ports1, ports2))
}
