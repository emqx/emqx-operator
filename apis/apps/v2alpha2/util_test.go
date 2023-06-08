/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v2alpha2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestGetDashboardServicePort(t *testing.T) {
	expect := &corev1.ServicePort{
		Name:       "dashboard-listeners-http-bind",
		Protocol:   corev1.ProtocolTCP,
		Port:       int32(18083),
		TargetPort: intstr.FromInt(18083),
	}

	t.Run("a single port", func(t *testing.T) {
		instance := &EMQX{}
		instance.Spec.BootstrapConfig = `dashboard.listeners.http.bind = 18083`
		got, err := GetDashboardServicePort(instance)
		assert.Nil(t, err)
		assert.Equal(t, expect, got)
	})

	t.Run("ipv4 address", func(t *testing.T) {
		instance := &EMQX{}
		instance.Spec.BootstrapConfig = `dashboard.listeners.http.bind = "0.0.0.0:18083"`
		got, err := GetDashboardServicePort(instance)
		assert.Nil(t, err)
		assert.Equal(t, expect, got)
	})

	t.Run("ipv6 address", func(t *testing.T) {
		instance := &EMQX{}
		instance.Spec.BootstrapConfig = `dashboard.listeners.http.bind = "[::]:18083"`
		got, err := GetDashboardServicePort(instance)
		assert.Nil(t, err)
		assert.Equal(t, expect, got)
	})

	t.Run("wrong bootstrap config", func(t *testing.T) {
		instance := &EMQX{}
		instance.Spec.BootstrapConfig = `hello world`
		got, err := GetDashboardServicePort(instance)
		assert.ErrorContains(t, err, "failed to parse")
		assert.Nil(t, got)
	})

	t.Run("empty bootstrap config", func(t *testing.T) {
		instance := &EMQX{}
		got, err := GetDashboardServicePort(instance)
		assert.ErrorContains(t, err, "failed to get dashboard.listeners.http.bind")
		assert.Nil(t, got)
	})

	t.Run("empty dashboard listeners config", func(t *testing.T) {
		instance := &EMQX{}
		instance.Spec.BootstrapConfig = `foo = bar`
		got, err := GetDashboardServicePort(instance)
		assert.ErrorContains(t, err, "failed to get dashboard.listeners.http.bind")
		assert.Nil(t, got)
	})
}

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

func TestMergeMap(t *testing.T) {
	m1 := map[string]string{
		"m0": "test-0",
		"m1": "test-1",
		"m2": "test-2",
	}

	m2 := map[string]string{
		"m0": "test-0",
		"m1": "test-1",
		"m3": "test-3",
	}

	expect := map[string]string{
		"m0": "test-0",
		"m1": "test-1",
		"m2": "test-2",
		"m3": "test-3",
	}
	assert.Equal(t, expect, mergeMap(m1, m2))
}
