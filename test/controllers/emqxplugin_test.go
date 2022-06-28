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

package controller_test

import (
	"context"
	"fmt"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	//+kubebuilder:scaffold:imports
)

var _ = Describe("Check default plugin", func() {
	pluginList := []string{"emqx_management", "emqx_dashboard", "emqx_rule_engine", "emqx_retainer"}
	loadedPlugins := "emqx_management.\nemqx_dashboard.\nemqx_retainer.\nemqx_rule_engine.\n"
	ports := []corev1.ServicePort{{
		Name:       "management-listener-http",
		Port:       8081,
		Protocol:   corev1.ProtocolTCP,
		TargetPort: intstr.FromInt(8081),
	}}

	Context("Check default plugins", func() {
		It("should create a EMQX plugin custom resource", func() {
			check_plugin(broker, pluginList)
			check_plugin(enterprise, append(pluginList, "emqx_modules"))
		})

		It("should create a configMap with plugins config", func() {
			check_plugins_config(broker, pluginList)
			check_plugin(enterprise, append(pluginList, "emqx_modules"))
		})

		It("should create a configMap with loaded plugins", func() {
			check_loaded_plugins(broker, loadedPlugins)
			check_loaded_plugins(enterprise, fmt.Sprintf("%semqx_modules.\n", loadedPlugins))
		})

		It("should add management port for handless service", func() {
			check_service_ports(types.NamespacedName{Name: "emqx-headless", Namespace: broker.Namespace}, ports)
			check_service_ports(types.NamespacedName{Name: "emqx-ee-headless", Namespace: enterprise.Namespace}, ports)
		})
	})
})

var _ = Describe("Check custom plugin", func() {
	pluginList := []string{"emqx_management", "emqx_dashboard", "emqx_rule_engine", "emqx_retainer", "emqx_lwm2m"}
	ports := []corev1.ServicePort{
		{
			Name:       "lwm2m-bind-udp-1",
			Protocol:   corev1.ProtocolUDP,
			Port:       5683,
			TargetPort: intstr.FromInt(5683),
		},
		{
			Name:       "lwm2m-bind-udp-2",
			Protocol:   corev1.ProtocolUDP,
			Port:       5684,
			TargetPort: intstr.FromInt(5684),
		},
		{
			Name:       "lwm2m-bind-dtls-1",
			Protocol:   corev1.ProtocolUDP,
			Port:       5685,
			TargetPort: intstr.FromInt(5685),
		},
		{
			Name:       "lwm2m-bind-dtls-2",
			Protocol:   corev1.ProtocolUDP,
			Port:       5686,
			TargetPort: intstr.FromInt(5686),
		},
	}
	JustBeforeEach(func() {
		lwm2m := &v1beta3.EmqxPlugin{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps.emqx.io/v1beta3",
				Kind:       "EmqxPlugin",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", broker.GetName(), "lwm2m"),
				Namespace: broker.GetNamespace(),
				Labels:    broker.GetLabels(),
			},
			Spec: v1beta3.EmqxPluginSpec{
				PluginName: "emqx_lwm2m",
				Selector:   broker.GetLabels(),
				Config: map[string]string{
					"lwm2m.lifetime_min": "1s",
					"lwm2m.lifetime_max": "86400s",
					"lwm2m.bind.udp.1":   "0.0.0.0:5683",
					"lwm2m.bind.udp.2":   "0.0.0.0:5684",
					"lwm2m.bind.dtls.1":  "0.0.0.0:5685",
					"lwm2m.bind.dtls.2":  "0.0.0.0:5686",
					"lwm2m.xml_dir":      "/opt/emqx/etc/lwm2m_xml",
				},
			},
		}
		Expect(k8sClient.Create(context.Background(), lwm2m)).Should(Succeed())

		Eventually(func() bool {
			_ = k8sClient.Get(context.Background(), types.NamespacedName{Name: lwm2m.GetName(), Namespace: lwm2m.GetNamespace()}, lwm2m)
			return lwm2m.Status.Phase == v1beta3.EmqxPluginStatusLoaded
		}, timeout, interval).Should(BeTrue())

		check_plugin(broker, pluginList)
		check_plugins_config(broker, pluginList)
		check_loaded_plugins(broker, "emqx_management.\nemqx_dashboard.\nemqx_retainer.\nemqx_rule_engine.\nemqx_lwm2m.\n")
		check_service_ports(types.NamespacedName{Name: "emqx", Namespace: broker.Namespace}, ports)
	})
	It("check update external plugin", func() {
		ports = []corev1.ServicePort{
			{
				Name:       "lwm2m-bind-dtls-1",
				Protocol:   corev1.ProtocolUDP,
				Port:       5685,
				TargetPort: intstr.FromInt(5685),
			},
			{
				Name:       "lwm2m-bind-dtls-2",
				Protocol:   corev1.ProtocolUDP,
				Port:       5686,
				TargetPort: intstr.FromInt(5686),
			},
			{
				Name:       "lwm2m-bind-udp-1",
				Protocol:   corev1.ProtocolUDP,
				Port:       5687,
				TargetPort: intstr.FromInt(5687),
			},
			{
				Name:       "lwm2m-bind-udp-2",
				Protocol:   corev1.ProtocolUDP,
				Port:       5688,
				TargetPort: intstr.FromInt(5688),
			},
		}

		Eventually(func() error {
			plugin := &v1beta3.EmqxPlugin{}
			err := k8sClient.Get(
				context.Background(),
				types.NamespacedName{
					Name:      fmt.Sprintf("%s-%s", broker.GetName(), "lwm2m"),
					Namespace: broker.GetNamespace(),
				}, plugin,
			)
			if err != nil {
				return err
			}
			plugin.Spec.Config["lwm2m.bind.udp.1"] = "5687"
			plugin.Spec.Config["lwm2m.bind.udp.2"] = "5688"
			return k8sClient.Update(context.Background(), plugin)
		}, timeout, interval).Should(Succeed())

		check_service_ports(types.NamespacedName{Name: "emqx", Namespace: broker.Namespace}, ports)
	})
	JustAfterEach(func() {
		ports = []corev1.ServicePort{
			{
				Name:       "lwm2m-bind-dtls-1",
				Protocol:   corev1.ProtocolUDP,
				Port:       5685,
				TargetPort: intstr.FromInt(5685),
			},
			{
				Name:       "lwm2m-bind-dtls-2",
				Protocol:   corev1.ProtocolUDP,
				Port:       5686,
				TargetPort: intstr.FromInt(5686),
			},
			{
				Name:       "lwm2m-bind-udp-1",
				Protocol:   corev1.ProtocolUDP,
				Port:       5687,
				TargetPort: intstr.FromInt(5687),
			},
			{
				Name:       "lwm2m-bind-udp-2",
				Protocol:   corev1.ProtocolUDP,
				Port:       5688,
				TargetPort: intstr.FromInt(5688),
			},
		}
		Eventually(func() error {
			plugin := &v1beta3.EmqxPlugin{}
			err := k8sClient.Get(
				context.Background(),
				types.NamespacedName{
					Name:      fmt.Sprintf("%s-%s", broker.GetName(), "lwm2m"),
					Namespace: broker.GetNamespace(),
				}, plugin,
			)
			if err != nil {
				if k8sErrors.IsNotFound(err) {
					return nil
				}
				return err
			}
			return k8sClient.Delete(context.Background(), plugin)
		}, timeout, interval).Should(Succeed())

		Eventually(func() string {
			cm := &corev1.ConfigMap{}
			_ = k8sClient.Get(
				context.Background(),
				types.NamespacedName{
					Name:      fmt.Sprintf("%s-%s", broker.GetName(), "loaded-plugins"),
					Namespace: broker.GetNamespace(),
				}, cm,
			)
			return cm.Data["loaded_plugins"]
		}, timeout, interval).ShouldNot(ContainSubstring("{emqx_lwm2m, true}"))

		Eventually(func() map[string]string {
			cm := &corev1.ConfigMap{}
			_ = k8sClient.Get(
				context.Background(),
				types.NamespacedName{
					Name:      fmt.Sprintf("%s-%s", broker.GetName(), "plugins-config"),
					Namespace: broker.GetNamespace(),
				}, cm,
			)
			return cm.Data
		}, timeout, interval).ShouldNot(HaveKey("emqx_lwm2m.conf"))

		Eventually(func() []corev1.ServicePort {
			svc := &corev1.Service{}
			_ = k8sClient.Get(
				context.Background(),
				types.NamespacedName{
					Name:      broker.GetName(),
					Namespace: broker.GetNamespace(),
				},
				svc,
			)
			return svc.Spec.Ports
		}).ShouldNot(ContainElements(ports))
	})
})

func check_plugin(emqx v1beta3.Emqx, pluginList []string) {
	Eventually(func() []string {
		list := v1beta3.EmqxPluginList{}
		_ = k8sClient.List(
			context.Background(),
			&list,
			client.InNamespace(emqx.GetNamespace()),
			client.MatchingLabels(emqx.GetLabels()),
		)
		l := []string{}
		for _, plugin := range list.Items {
			l = append(l, plugin.Spec.PluginName)
		}
		return l
	}, timeout, interval).Should(ConsistOf(pluginList))
}

func check_plugins_config(emqx v1beta3.Emqx, pluginList []string) {
	for _, pluginName := range pluginList {
		Eventually(func() map[string]string {
			cm := &corev1.ConfigMap{}
			_ = k8sClient.Get(
				context.Background(),
				types.NamespacedName{
					Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "plugins-config"),
					Namespace: emqx.GetNamespace(),
				}, cm,
			)
			return cm.Data
		}, timeout, interval).Should(HaveKey(pluginName + ".conf"))
	}
}

func check_loaded_plugins(emqx v1beta3.Emqx, loadedPlugins string) {
	Eventually(func() map[string]string {
		cm := &corev1.ConfigMap{}
		_ = k8sClient.Get(
			context.Background(),
			types.NamespacedName{
				Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "loaded-plugins"),
				Namespace: emqx.GetNamespace(),
			}, cm,
		)
		return cm.Data
	}, timeout, interval).Should(Equal(
		map[string]string{
			"loaded_plugins": loadedPlugins,
		},
	))
}
