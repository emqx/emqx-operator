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

package e2e

import (
	"context"
	"fmt"

	appsv1beta3 "github.com/emqx/emqx-operator/apis/apps/v1beta3"
	"github.com/emqx/emqx-operator/pkg/handler"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Check EMQX Custom Resource", func() {
	var headlessPort corev1.ServicePort
	var ports, pluginPorts []corev1.ServicePort
	var pluginList []string

	Context("check EMQX Custom Resources", func() {
		BeforeEach(func() {
			headlessPort = corev1.ServicePort{
				Name:       "http-management-8081",
				Port:       8081,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(8081),
			}
			ports = []corev1.ServicePort{
				{
					Name:       "mqtt-tcp-1883",
					Port:       1883,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(1883),
				},
				{
					Name:       "mqtt-ssl-8883",
					Port:       8883,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8883),
				},
				{
					Name:       "mqtt-ws-8083",
					Port:       8083,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8083),
				},
				{
					Name:       "mqtt-wss-8084",
					Port:       8084,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8084),
				},
			}
			pluginPorts = []corev1.ServicePort{
				{
					Name:       "lwm2m-udp-5683",
					Protocol:   corev1.ProtocolUDP,
					Port:       5683,
					TargetPort: intstr.FromInt(5683),
				},
				{
					Name:       "lwm2m-udp-5684",
					Protocol:   corev1.ProtocolUDP,
					Port:       5684,
					TargetPort: intstr.FromInt(5684),
				},
				{
					Name:       "lwm2m-dtls-5685",
					Protocol:   corev1.ProtocolUDP,
					Port:       5685,
					TargetPort: intstr.FromInt(5685),
				},
				{
					Name:       "lwm2m-dtls-5686",
					Protocol:   corev1.ProtocolUDP,
					Port:       5686,
					TargetPort: intstr.FromInt(5686),
				},
			}
			pluginList = []string{"emqx_rule_engine", "emqx_retainer", "emqx_lwm2m"}
		})

		It("check default resource", func() {
			By("should check EMQX CR Status")
			check_emqx_status(broker)
			check_emqx_status(enterprise)

			By("should check StatefulSet annotations")
			check_pod_annotations(broker)
			check_pod_annotations(enterprise)

			By("should create EMQX Plugins")
			check_plugin(broker, pluginList)
			check_plugin(enterprise, append(pluginList, "emqx_modules"))

			By("should bind listener ports to service")
			check_service_ports(broker, append(ports, pluginPorts...), headlessPort)
			check_service_ports(enterprise, append(ports, pluginPorts...), headlessPort)

		})
	})

	Context("check update EMQX Plugins", func() {
		BeforeEach(func() {
			pluginPorts = []corev1.ServicePort{
				{
					Name:       "lwm2m-dtls-5685",
					Protocol:   corev1.ProtocolUDP,
					Port:       5685,
					TargetPort: intstr.FromInt(5685),
				},
				{
					Name:       "lwm2m-dtls-5686",
					Protocol:   corev1.ProtocolUDP,
					Port:       5686,
					TargetPort: intstr.FromInt(5686),
				},
				{
					Name:       "lwm2m-udp-5687",
					Protocol:   corev1.ProtocolUDP,
					Port:       5687,
					TargetPort: intstr.FromInt(5687),
				},
				{
					Name:       "lwm2m-udp-5688",
					Protocol:   corev1.ProtocolUDP,
					Port:       5688,
					TargetPort: intstr.FromInt(5688),
				},
			}

			update_lwm2m(broker)
			update_lwm2m(enterprise)
		})
		It("", func() {
			By("should bind ports to service")
			check_service_ports(broker, append(ports, pluginPorts...), headlessPort)
			check_service_ports(enterprise, append(ports, pluginPorts...), headlessPort)
		})

		AfterEach(func() {
			Eventually(func() error {
				plugin := &appsv1beta3.EmqxPlugin{}
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

			By("should delete plugin resource")

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
			}, timeout, interval).ShouldNot(ContainSubstring("emqx_lwm2m"))

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
			}, timeout, interval).ShouldNot(ContainElements(pluginPorts))
		})
	})
})

func update_lwm2m(emqx appsv1beta3.Emqx) {
	Eventually(func() error {
		plugin := &appsv1beta3.EmqxPlugin{}
		err := k8sClient.Get(
			context.Background(),
			types.NamespacedName{
				Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "lwm2m"),
				Namespace: emqx.GetNamespace(),
			}, plugin,
		)
		if err != nil {
			return err
		}
		plugin.Spec.Config["lwm2m.bind.udp.1"] = "0.0.0.0:5687"
		plugin.Spec.Config["lwm2m.bind.udp.2"] = "0.0.0.0:5688"
		return k8sClient.Update(context.Background(), plugin)
	}, timeout, interval).Should(Succeed())
}

func check_emqx_status(emqx appsv1beta3.Emqx) {
	Eventually(func() bool {
		_ = k8sClient.Get(
			context.TODO(),
			types.NamespacedName{
				Name:      emqx.GetName(),
				Namespace: emqx.GetNamespace(),
			},
			emqx,
		)
		status := emqx.GetStatus()
		return status.IsRunning()
	}, timeout, interval).Should(BeTrue())

	Expect(emqx.GetStatus().Replicas).Should(Equal(int32(3)))
	Expect(emqx.GetStatus().ReadyReplicas).Should(Equal(int32(3)))
	Expect(emqx.GetStatus().EmqxNodes).Should(HaveLen(3))
}

func check_pod_annotations(emqx appsv1beta3.Emqx) {
	sts := &appsv1.StatefulSet{}
	Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), sts)).Should(Succeed())
	Expect(sts.Spec.Template.Annotations).Should(HaveKey(handler.ManageContainersAnnotation))
	if len(emqx.GetACL()) != 0 {
		Expect(sts.Spec.Template.Annotations).Should(HaveKey("ACL/Base64EncodeConfig"))
	} else {
		Expect(sts.Spec.Template.Annotations).ShouldNot(HaveKey("ACL/Base64EncodeConfig"))
	}

	switch instance := emqx.(type) {
	case *appsv1beta3.EmqxBroker:
		if len(instance.GetModules()) != 0 {
			Expect(sts.Spec.Template.Annotations).Should(HaveKey("LoadedModules/Base64EncodeConfig"))
		} else {
			Expect(sts.Spec.Template.Annotations).ShouldNot(HaveKey("LoadedModules/Base64EncodeConfig"))
		}
	case *appsv1beta3.EmqxEnterprise:
		if len(instance.GetModules()) != 0 {
			Expect(sts.Spec.Template.Annotations).Should(HaveKey("LoadedModules/Base64EncodeConfig"))
		} else {
			Expect(sts.Spec.Template.Annotations).ShouldNot(HaveKey("LoadedModules/Base64EncodeConfig"))
		}
	}
}

func check_service_ports(emqx appsv1beta3.Emqx, ports []corev1.ServicePort, headlessPort corev1.ServicePort) {
	Eventually(func() []corev1.ServicePort {
		svc := &corev1.Service{}
		_ = k8sClient.Get(
			context.Background(),
			types.NamespacedName{
				Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "headless"),
				Namespace: emqx.GetNamespace(),
			},
			svc,
		)
		return svc.Spec.Ports
	}, timeout, interval).Should(ContainElements(headlessPort))
	Eventually(func() []corev1.ServicePort {
		svc := &corev1.Service{}
		_ = k8sClient.Get(
			context.Background(),
			types.NamespacedName{
				Name:      emqx.GetName(),
				Namespace: emqx.GetNamespace(),
			},
			svc,
		)
		return svc.Spec.Ports
	}, timeout, interval).Should(ContainElements(append(ports, headlessPort)))
}

func check_plugin(emqx appsv1beta3.Emqx, pluginList []string) {
	Eventually(func() []string {
		list := appsv1beta3.EmqxPluginList{}
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

		Eventually(func() string {
			cm := &corev1.ConfigMap{}
			_ = k8sClient.Get(
				context.Background(),
				types.NamespacedName{
					Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "loaded-plugins"),
					Namespace: emqx.GetNamespace(),
				}, cm,
			)
			return cm.Data["loaded_plugins"]
		}, timeout, interval).Should(ContainSubstring(pluginName))
	}
}
