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

package v1beta4

import (
	"context"
	"fmt"

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	"github.com/emqx/emqx-operator/pkg/handler"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var emqxBroker = &appsv1beta4.EmqxBroker{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "emqx",
		Namespace: "default",
		Labels: map[string]string{
			"test": "e2e",
		},
	},
	Spec: appsv1beta4.EmqxBrokerSpec{
		Template: appsv1beta4.EmqxTemplate{
			Spec: appsv1beta4.EmqxTemplateSpec{
				EmqxContainer: appsv1beta4.EmqxContainer{
					Name:  "emqx",
					Image: "emqx/emqx:4.4.9",
				},
			},
		},
	},
}

var emqxEnterprise = &appsv1beta4.EmqxEnterprise{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "emqx-ee",
		Namespace: "default",
		Labels: map[string]string{
			"test": "e2e",
		},
	},
	Spec: appsv1beta4.EmqxEnterpriseSpec{
		Template: appsv1beta4.EmqxTemplate{
			Spec: appsv1beta4.EmqxTemplateSpec{
				EmqxContainer: appsv1beta4.EmqxContainer{
					Name:  "emqx",
					Image: "emqx/emqx-ee:4.4.9",
				},
			},
		},
	},
}

var lwm2m = &appsv1beta4.EmqxPlugin{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "apps.emqx.io/v1beta3",
		Kind:       "EmqxPlugin",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "lwm2m",
		Namespace: "default",
		Labels: map[string]string{
			"test": "e2e",
		},
	},
	Spec: appsv1beta4.EmqxPluginSpec{
		PluginName: "emqx_lwm2m",
		Selector: map[string]string{
			"test": "e2e",
		},
		Config: map[string]string{
			"lwm2m.lifetime_min": "1s",
			"lwm2m.lifetime_max": "86400s",
			"lwm2m.bind.udp.1":   "0.0.0.0:5685",
			"lwm2m.bind.udp.2":   "0.0.0.0:5686",
			"lwm2m.bind.dtls.1":  "0.0.0.0:5687",
			"lwm2m.bind.dtls.2":  "0.0.0.0:5688",
			"lwm2m.xml_dir":      "/opt/emqx/etc/lwm2m_xml",
		},
	},
}

var _ = Describe("", func() {
	DescribeTable("",
		func(emqx appsv1beta4.Emqx, plugin *appsv1beta4.EmqxPlugin) {
			var pluginList []string
			var pluginPorts []corev1.ServicePort
			var ports []corev1.ServicePort
			var headlessPort corev1.ServicePort

			pluginList = []string{"emqx_rule_engine", "emqx_retainer", "emqx_lwm2m"}
			if _, ok := emqx.(*appsv1beta4.EmqxEnterprise); ok {
				pluginList = append(pluginList, "emqx_modules")
			}

			pluginPorts = []corev1.ServicePort{
				{
					Name:       "lwm2m-udp-5685",
					Protocol:   corev1.ProtocolUDP,
					Port:       5685,
					TargetPort: intstr.FromInt(5685),
				},
				{
					Name:       "lwm2m-udp-5686",
					Protocol:   corev1.ProtocolUDP,
					Port:       5686,
					TargetPort: intstr.FromInt(5686),
				},
				{
					Name:       "lwm2m-dtls-5687",
					Protocol:   corev1.ProtocolUDP,
					Port:       5687,
					TargetPort: intstr.FromInt(5687),
				},
				{
					Name:       "lwm2m-dtls-5688",
					Protocol:   corev1.ProtocolUDP,
					Port:       5688,
					TargetPort: intstr.FromInt(5688),
				},
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

			headlessPort = corev1.ServicePort{
				Name:       "http-management-8081",
				Port:       8081,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(8081),
			}
			By("create EMQX CR")
			emqx.Default()
			Expect(emqx.ValidateCreate()).Should(Succeed())
			Expect(k8sClient.Create(context.TODO(), emqx)).Should(Succeed())
			Eventually(func() bool {
				_ = k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Name:      emqx.GetName(),
						Namespace: emqx.GetNamespace(),
					},
					emqx,
				)
				return emqx.IsRunning()
			}, timeout, interval).Should(BeTrue())

			By("create EMQX Plugin")
			plugin.Labels = emqx.GetLabels()
			Expect(k8sClient.Create(context.TODO(), plugin)).Should(Succeed())

			By("check EMQX CR status")
			Expect(emqx.GetStatus().Replicas).Should(Equal(int32(3)))
			Expect(emqx.GetStatus().ReadyReplicas).Should(Equal(int32(3)))
			Expect(emqx.GetStatus().EmqxNodes).Should(HaveLen(3))

			By("check pod annotations")
			sts := &appsv1.StatefulSet{}
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), sts)).Should(Succeed())
			Expect(sts.Spec.Template.Annotations).Should(HaveKey(handler.ManageContainersAnnotation))

			By("check plugins")
			Eventually(func() []string {
				list := appsv1beta4.EmqxPluginList{}
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
			}

			By("check headless service ports")
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

			By("check service ports")
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
			}, timeout, interval).Should(ContainElements(append(pluginPorts, append(ports, headlessPort)...)))

			By("update EMQX Plugin")
			Eventually(func() error {
				plugin := &appsv1beta4.EmqxPlugin{}
				err := k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Name:      lwm2m.GetName(),
						Namespace: lwm2m.GetNamespace(),
					}, plugin,
				)
				if err != nil {
					return err
				}
				plugin.Spec.Config["lwm2m.bind.udp.1"] = "0.0.0.0:5695"
				plugin.Spec.Config["lwm2m.bind.udp.2"] = "0.0.0.0:5696"
				plugin.Spec.Config["lwm2m.bind.dtls.1"] = "0.0.0.0:5697"
				plugin.Spec.Config["lwm2m.bind.dtls.2"] = "0.0.0.0:5698"
				return k8sClient.Update(context.Background(), plugin)
			}, timeout, interval).Should(Succeed())

			pluginPorts = []corev1.ServicePort{
				{
					Name:       "lwm2m-udp-5695",
					Protocol:   corev1.ProtocolUDP,
					Port:       5695,
					TargetPort: intstr.FromInt(5695),
				},
				{
					Name:       "lwm2m-udp-5696",
					Protocol:   corev1.ProtocolUDP,
					Port:       5696,
					TargetPort: intstr.FromInt(5696),
				},
				{
					Name:       "lwm2m-dtls-5697",
					Protocol:   corev1.ProtocolUDP,
					Port:       5697,
					TargetPort: intstr.FromInt(5697),
				},
				{
					Name:       "lwm2m-dtls-5698",
					Protocol:   corev1.ProtocolUDP,
					Port:       5698,
					TargetPort: intstr.FromInt(5698),
				},
			}

			By("check service ports")
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
			}, timeout, interval).Should(ContainElements(append(pluginPorts, append(ports, headlessPort)...)))

			By("delete EMQX CR and EMQX Plugin")
			finalizer := "apps.emqx.io/finalizer"
			plugins := &appsv1beta4.EmqxPluginList{}
			_ = k8sClient.List(
				context.Background(),
				plugins,
				client.InNamespace("default"),
			)
			for _, plugin := range plugins.Items {
				controllerutil.RemoveFinalizer(&plugin, finalizer)
				Expect(k8sClient.Update(context.Background(), &plugin)).Should(Succeed())
				Expect(k8sClient.Delete(context.Background(), &plugin)).Should(Succeed())
			}
			Expect(k8sClient.Delete(context.TODO(), emqx)).Should(Succeed())
		},
		Entry(nil, emqxBroker.DeepCopy(), lwm2m.DeepCopy()),
		Entry(nil, emqxEnterprise.DeepCopy(), lwm2m.DeepCopy()),
	)
})
