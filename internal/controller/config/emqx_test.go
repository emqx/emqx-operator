/*
Copyright 2025.

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

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestCopy(t *testing.T) {
	t.Run("copy config", func(t *testing.T) {
		config, err := EMQXConfig(`
		durable_sessions.enable = true
		listeners.tcp.default.bind = 11883
		`)
		assert.Nil(t, err)
		got := config.Copy()
		assert.NotSame(t, config.Config, got.Config)
		got.StripReadOnlyConfig()
		assert.NotEqual(t, config.JSON(), got.JSON())
	})
}

func TestNodeCookie(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		config, err := EMQXConfig("")
		assert.Nil(t, err)
		got := config.GetNodeCookie()
		assert.Equal(t, "", got)
	})

	t.Run("simple cookie", func(t *testing.T) {
		config, err := EMQXConfig(`node.cookie = COOKIE`)
		assert.Nil(t, err)
		got := config.GetNodeCookie()
		assert.Equal(t, "COOKIE", got)
	})

	t.Run("cookie with space and special characters", func(t *testing.T) {
		config, err := EMQXConfig(`node.cookie = "COOKIE #!@#$"`)
		assert.Nil(t, err)
		got := config.GetNodeCookie()
		assert.Equal(t, "COOKIE #!@#$", got)
	})

	t.Run("triple-quoted cookie", func(t *testing.T) {
		config, err := EMQXConfig(`node.cookie = """~
			COOKIE
			LONGER THAN
			NEEDED~"""`)
		assert.Nil(t, err)
		got := config.GetNodeCookie()
		assert.Equal(t, "COOKIE\nLONGER THAN\nNEEDED", got)
	})
}

func TestStripReadOnlyConfig(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		config, err := EMQXConfig("")
		assert.Nil(t, err)
		got := config.StripReadOnlyConfig()
		assert.Empty(t, got)
	})

	t.Run("non-empty config", func(t *testing.T) {
		config, err := EMQXConfig(`
		node.cookie = COOKIE
		cluster.name = emqx
		listeners {
			tcp.default.bind = 18083
			ssl.default.bind = 18084
		}
		durable_sessions {
			enable = true
		}
		`)
		assert.Nil(t, err)
		got := config.StripReadOnlyConfig()
		assert.ElementsMatch(t, got, []string{
			"node",
			"cluster.name",
			"durable_sessions",
		})
	})

	t.Run("wrong config", func(t *testing.T) {
		_, err := EMQXConfig("hello world")
		assert.Error(t, err)
	})
}

func TestGetDashboardPortMap(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		config, err := EMQXConfig("")
		assert.Nil(t, err)
		got := config.GetDashboardPortMap()
		assert.Equal(t, map[string]int{
			"dashboard": 18083,
		}, got)
	})

	t.Run("a single http port", func(t *testing.T) {
		config, err := EMQXConfig(`dashboard.listeners.http.bind = 18083`)
		assert.Nil(t, err)
		got := config.GetDashboardPortMap()
		assert.Equal(t, map[string]int{
			"dashboard": 18083,
		}, got)
	})

	t.Run("a single IPV4 http port", func(t *testing.T) {
		config, err := EMQXConfig(`dashboard.listeners.http.bind = "0.0.0.0:18083"`)
		assert.Nil(t, err)
		got := config.GetDashboardPortMap()
		assert.Equal(t, map[string]int{
			"dashboard": 18083,
		}, got)
	})

	t.Run("a single IPV6 http port", func(t *testing.T) {
		config, err := EMQXConfig(`dashboard.listeners.http.bind = "[::]:18083"`)
		assert.Nil(t, err)
		got := config.GetDashboardPortMap()
		assert.Equal(t, map[string]int{
			"dashboard": 18083,
		}, got)
	})

	t.Run("a single https port", func(t *testing.T) {
		config, err := EMQXConfig(`dashboard.listeners.https.bind = 18084`)
		assert.Nil(t, err)
		got := config.GetDashboardPortMap()
		assert.Equal(t, map[string]int{
			"dashboard":       18083, // default http port
			"dashboard-https": 18084,
		}, got)
	})

	t.Run("disable http port and a single https port", func(t *testing.T) {
		config, err := EMQXConfig(`
			dashboard.listeners.http.bind = 0
			dashboard.listeners.https.bind = 18084
		`)
		assert.Nil(t, err)
		got := config.GetDashboardPortMap()
		assert.Equal(t, map[string]int{
			"dashboard-https": 18084,
		}, got)
	})

	t.Run("disable all port", func(t *testing.T) {
		config, err := EMQXConfig(`
			dashboard.listeners.http.bind = 0
			dashboard.listeners.https.bind = 0
		`)
		assert.Nil(t, err)
		got := config.GetDashboardPortMap()
		assert.Empty(t, got)
	})
}

func TestGetDashboardServicePorts(t *testing.T) {
	expect := []corev1.ServicePort{
		{
			Name:       "dashboard",
			Protocol:   corev1.ProtocolTCP,
			Port:       int32(18083),
			TargetPort: intstr.Parse("18083"),
		},
	}

	t.Run("empty config with defaults", func(t *testing.T) {
		config, err := EMQXConfigWithDefaults("")
		assert.Nil(t, err)
		got := config.GetDashboardServicePorts()
		assert.Equal(t, expect, got)
	})

	t.Run("a single port", func(t *testing.T) {
		config, err := EMQXConfig(`dashboard.listeners.http.bind = 18083`)
		assert.Nil(t, err)
		got := config.GetDashboardServicePorts()
		assert.Equal(t, expect, got)
	})

	t.Run("ipv4 address", func(t *testing.T) {
		config, err := EMQXConfig(`dashboard.listeners.http.bind = "0.0.0.0:18083"`)
		assert.Nil(t, err)
		got := config.GetDashboardServicePorts()
		assert.Equal(t, expect, got)
	})

	t.Run("ipv6 address", func(t *testing.T) {
		config, err := EMQXConfig(`dashboard.listeners.http.bind = "[::]:18083"`)
		assert.Nil(t, err)
		got := config.GetDashboardServicePorts()
		assert.Equal(t, expect, got)
	})

	t.Run("empty config", func(t *testing.T) {
		config, err := EMQXConfig("")
		assert.Nil(t, err)
		got := config.GetDashboardServicePorts()
		assert.Equal(t, expect, got)
	})
}

func TestGetListenersServicePorts(t *testing.T) {
	t.Run("check listeners", func(t *testing.T) {
		config, err := EMQXConfig(`
			listeners.tcp.default.bind = "0.0.0.0:1883"
			listeners.ssl.default.bind = "0.0.0.0:8883"
			listeners.ws.default.bind = "0.0.0.0:8083"
			listeners.wss.default.bind = "0.0.0.0:8084"
			listeners.quic.default.bind = "0.0.0.0:14567"
		`)
		assert.Nil(t, err)
		got := config.GetListenersServicePorts()
		assert.ElementsMatch(t, []corev1.ServicePort{
			{
				Name:       "tcp-default",
				Protocol:   corev1.ProtocolTCP,
				Port:       1883,
				TargetPort: intstr.Parse("1883"),
			},
			{
				Name:       "ssl-default",
				Protocol:   corev1.ProtocolTCP,
				Port:       8883,
				TargetPort: intstr.Parse("8883"),
			},
			{
				Name:       "ws-default",
				Protocol:   corev1.ProtocolTCP,
				Port:       8083,
				TargetPort: intstr.Parse("8083"),
			},
			{
				Name:       "wss-default",
				Protocol:   corev1.ProtocolTCP,
				Port:       8084,
				TargetPort: intstr.Parse("8084"),
			},
			{
				Name:       "quic-default",
				Protocol:   corev1.ProtocolUDP,
				Port:       14567,
				TargetPort: intstr.Parse("14567"),
			},
		}, got)
	})

	t.Run("check gateway listeners", func(t *testing.T) {
		config, err := EMQXConfig(`
			gateway.coap.listeners.udp.default.bind = "5683"
			gateway.exporto.listeners.tcp.default.bind = "7993"
			gateway.lwm2w.listeners.udp.default.bind = "5783"
			gateway.mqttsn.listeners.udp.default.bind = "1884"
			gateway.stomp.listeners.tcp.default.bind = "61613"
		`)
		assert.Nil(t, err)
		got := config.GetListenersServicePorts()
		assert.ElementsMatch(t, []corev1.ServicePort{
			{
				Name:       "coap-udp-default",
				Protocol:   corev1.ProtocolUDP,
				Port:       5683,
				TargetPort: intstr.Parse("5683"),
			},
			{
				Name:       "exporto-tcp-default",
				Protocol:   corev1.ProtocolTCP,
				Port:       7993,
				TargetPort: intstr.Parse("7993"),
			},
			{
				Name:       "lwm2w-udp-default",
				Protocol:   corev1.ProtocolUDP,
				Port:       5783,
				TargetPort: intstr.Parse("5783"),
			},
			{
				Name:       "mqttsn-udp-default",
				Protocol:   corev1.ProtocolUDP,
				Port:       1884,
				TargetPort: intstr.Parse("1884"),
			},
			{
				Name:       "stomp-tcp-default",
				Protocol:   corev1.ProtocolTCP,
				Port:       61613,
				TargetPort: intstr.Parse("61613"),
			},
		}, got)
	})
}

func TestJSON(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		config, err := EMQXConfig("")
		assert.Nil(t, err)
		got := config.JSON()
		assert.Equal(t, "", got)
	})

	t.Run("arrays", func(t *testing.T) {
		config, err := EMQXConfig(`
			node.name = "emqx@127.0.0.1"
			cluster.core_nodes = ["emqx@node1.emqx.io", "emqx@node2.emqx.io"]
		`)
		assert.Nil(t, err)
		got := config.JSON()
		expected := `{"cluster":{"core_nodes":["emqx@node1.emqx.io","emqx@node2.emqx.io"]},"node":{"name":"emqx@127.0.0.1"}}`
		assert.JSONEq(t, expected, got)
	})

	t.Run("empty arrays", func(t *testing.T) {
		config, err := EMQXConfig(`cluster.core_nodes = []`)
		assert.Nil(t, err)
		got := config.JSON()
		expected := `{"cluster":{"core_nodes":[]}}`
		assert.Equal(t, expected, got)
	})

	t.Run("nested arrays", func(t *testing.T) {
		config, err := EMQXConfig(`cluster.seed_nodes = [["emqx@node1.emqx.io"], []]`)
		assert.Nil(t, err)
		got := config.JSON()
		expected := `{"cluster":{"seed_nodes":[["emqx@node1.emqx.io"],[]]}}`
		assert.Equal(t, expected, got)
	})
}

func TestStrip(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		config, err := EMQXConfig("")
		assert.Nil(t, err)
		got := config.Strip("dashboard.listeners.http.bind")
		assert.Equal(t, got, false)
		assert.Equal(t, config.JSON(), "")
	})

	t.Run("delete non-existent key", func(t *testing.T) {
		config, err := EMQXConfig(`
		dashboard.listeners.http.bind = 18083
		`)
		assert.Nil(t, err)
		got := config.Strip("dashboard.config.file")
		assert.Equal(t, got, false)
		assert.JSONEq(t, `{"dashboard":{"listeners":{"http":{"bind":18083}}}}`, config.JSON())
	})

	t.Run("delete whole root", func(t *testing.T) {
		config, err := EMQXConfig(`
		dashboard.listeners { http.bind = 18083, https.bind = 18084 }
		`)
		assert.Nil(t, err)
		got := config.Strip("dashboard")
		assert.Equal(t, got, true)
		assert.Equal(t, config.JSON(), "")
	})

	t.Run("delete empty leftover objects", func(t *testing.T) {
		config, err := EMQXConfig(`
		dashboard.listeners { http.bind = 18083, https.bind = 18084 }
		`)
		assert.Nil(t, err)
		got := config.Strip("dashboard.listeners.https.bind")
		assert.Equal(t, got, true)
		assert.JSONEq(t, `{"dashboard":{"listeners":{"http":{"bind":18083}}}}`, config.JSON())
	})
}
