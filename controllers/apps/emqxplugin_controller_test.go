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

package apps_test

import (
	"testing"

	appsv1beta3 "github.com/emqx/emqx-operator/apis/apps/v1beta3"
	appscontrollers "github.com/emqx/emqx-operator/controllers/apps"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var plugins = []appsv1beta3.EmqxPlugin{
	{
		Spec: appsv1beta3.EmqxPluginSpec{
			PluginName: "emqx_management",
			Config: map[string]string{
				"management.listener.http": "8081",
			},
		},
	},
	{
		Spec: appsv1beta3.EmqxPluginSpec{
			PluginName: "emqx_dashboard",
			Config: map[string]string{
				"dashboard.listener.http": "18083",
			},
		},
	},
	{
		Spec: appsv1beta3.EmqxPluginSpec{
			PluginName: "emqx_lwm2m",
			Config: map[string]string{
				"lwm2m.bind.udp.1":  "5683",
				"lwm2m.bind.dtls.1": "0.0.0.0:5684",
			},
		},
	},
	{
		Spec: appsv1beta3.EmqxPluginSpec{
			PluginName: "emqx_coap",
			Config: map[string]string{
				"coap.bind.udp.1":  "0.0.0.0:5685",
				"coap.bind.dtls.1": "0.0.0.0:5686",
			},
		},
	},
	{
		Spec: appsv1beta3.EmqxPluginSpec{
			PluginName: "emqx_sn",
			Config: map[string]string{
				"mqtt.sn.port": "1884",
			},
		},
	},
	{
		Spec: appsv1beta3.EmqxPluginSpec{
			PluginName: "emqx_exproto",
			Config: map[string]string{
				"exproto.server.http.port":   "9100",
				"exproto.server.https.port":  "9101",
				"exproto.listener.protoname": "tcp://0.0.0.0:7993",
			},
		},
	},
	{
		Spec: appsv1beta3.EmqxPluginSpec{
			PluginName: "emqx_stomp",
			Config: map[string]string{
				"stomp.listener": "61613",
			},
		},
	},
	{
		Spec: appsv1beta3.EmqxPluginSpec{
			PluginName: "emqx_jt808",
			Config: map[string]string{
				"jt808.listener.tcp": "6207",
				"jt808.listener.ssl": "18084",
			},
		},
	},
	{
		Spec: appsv1beta3.EmqxPluginSpec{
			PluginName: "emqx_tcp",
			Config: map[string]string{
				"tcp.listener.external": "0.0.0.0:8090",
				"tcp.listener.ssl.fake": "0.0.0.0:8091",
			},
		},
	},
	{
		Spec: appsv1beta3.EmqxPluginSpec{
			PluginName: "emqx_gbt32960",
			Config: map[string]string{
				"gbt32960.listener.tcp": "6208",
				"gbt32960.listener.ssl": "7326",
			},
		},
	},
}

var servicePorts = []corev1.ServicePort{
	{
		Name:       "management-listener-http",
		Protocol:   corev1.ProtocolTCP,
		Port:       8081,
		TargetPort: intstr.FromInt(8081),
	},
	{
		Name:       "dashboard-listener-http",
		Protocol:   corev1.ProtocolTCP,
		Port:       18083,
		TargetPort: intstr.FromInt(18083),
	},
	{
		Name:       "lwm2m-bind-udp-1",
		Protocol:   corev1.ProtocolUDP,
		Port:       5683,
		TargetPort: intstr.FromInt(5683),
	},
	{
		Name:       "lwm2m-bind-dtls-1",
		Protocol:   corev1.ProtocolUDP,
		Port:       5684,
		TargetPort: intstr.FromInt(5684),
	},
	{
		Name:       "coap-bind-udp-1",
		Protocol:   corev1.ProtocolUDP,
		Port:       5685,
		TargetPort: intstr.FromInt(5685),
	},
	{
		Name:       "coap-bind-dtls-1",
		Protocol:   corev1.ProtocolUDP,
		Port:       5686,
		TargetPort: intstr.FromInt(5686),
	},
	{
		Name:       "mqtt-sn-port",
		Protocol:   corev1.ProtocolUDP,
		Port:       1884,
		TargetPort: intstr.FromInt(1884),
	},
	{
		Name:       "exproto-server-http-port",
		Protocol:   corev1.ProtocolTCP,
		Port:       9100,
		TargetPort: intstr.FromInt(9100),
	},
	{
		Name:       "exproto-server-https-port",
		Protocol:   corev1.ProtocolTCP,
		Port:       9101,
		TargetPort: intstr.FromInt(9101),
	},
	{
		Name:       "exproto-listener-protoname",
		Protocol:   corev1.ProtocolTCP,
		Port:       7993,
		TargetPort: intstr.FromInt(7993),
	},
	{
		Name:       "stomp-listener",
		Protocol:   corev1.ProtocolTCP,
		Port:       61613,
		TargetPort: intstr.FromInt(61613),
	},
	{
		Name:       "jt808-listener-tcp",
		Protocol:   corev1.ProtocolTCP,
		Port:       6207,
		TargetPort: intstr.FromInt(6207),
	},
	{
		Name:       "jt808-listener-ssl",
		Protocol:   corev1.ProtocolTCP,
		Port:       18084,
		TargetPort: intstr.FromInt(18084),
	},
	{
		Name:       "tcp-listener-external",
		Protocol:   corev1.ProtocolTCP,
		Port:       8090,
		TargetPort: intstr.FromInt(8090),
	},
	{
		Name:       "tcp-listener-ssl-fake",
		Protocol:   corev1.ProtocolTCP,
		Port:       8091,
		TargetPort: intstr.FromInt(8091),
	},
	{
		Name:       "gbt32960-listener-tcp",
		Protocol:   corev1.ProtocolTCP,
		Port:       6208,
		TargetPort: intstr.FromInt(6208),
	},
	{
		Name:       "gbt32960-listener-ssl",
		Protocol:   corev1.ProtocolTCP,
		Port:       7326,
		TargetPort: intstr.FromInt(7326),
	},
}

func TestInsertServicePorts(t *testing.T) {
	ports := []corev1.ServicePort{}
	for _, plugin := range plugins {
		ports = appscontrollers.InsertServicePorts(&plugin, ports)
	}
	assert.ElementsMatch(t, ports, servicePorts)
}

func TestRemoveServicePorts(t *testing.T) {
	ports := servicePorts
	for _, plugin := range plugins {
		ports = appscontrollers.RemoveServicePorts(&plugin, ports)
	}
	assert.Empty(t, ports)
}
