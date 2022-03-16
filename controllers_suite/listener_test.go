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

package controller_suite_test

import (
	"context"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var _ = Describe("", func() {
	Context("Check service", func() {
		servicePorts, containerPorts, env := ports()

		It("Check headless service", func() {
			for _, emqx := range emqxList() {
				names := v1beta3.Names{Object: emqx}
				svc := &corev1.Service{}
				Eventually(func() bool {
					err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      names.HeadlessSvc(),
							Namespace: emqx.GetNamespace(),
						},
						svc,
					)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				Expect(svc.Spec.Type).Should(Equal(corev1.ServiceTypeClusterIP))
				Expect(svc.Spec.ClusterIP).Should(Equal(corev1.ClusterIPNone))
			}
		})

		It("Check listener service", func() {
			for _, emqx := range emqxList() {
				svc := &corev1.Service{}
				Eventually(func() bool {
					err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      emqx.GetName(),
							Namespace: emqx.GetNamespace(),
						},
						svc,
					)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				Expect(svc.Spec.Type).Should(Equal(corev1.ServiceTypeClusterIP))
				Expect(svc.Spec.Ports).Should(ConsistOf(servicePorts))

				sts := &appsv1.StatefulSet{}
				Eventually(func() bool {
					err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      emqx.GetName(),
							Namespace: emqx.GetNamespace(),
						},
						sts,
					)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				Expect(sts.Spec.Template.Spec.Containers[0].Ports).Should(ConsistOf(containerPorts))
				Expect(sts.Spec.Template.Spec.Containers[0].Env).Should(ContainElements(env))
				Expect(sts.Spec.Template.Spec.Containers[0].ReadinessProbe.ProbeHandler.HTTPGet.Port).Should(
					Equal(intstr.IntOrString{IntVal: int32(8081)}),
				)
				Expect(sts.Spec.Template.Spec.Containers[0].LivenessProbe.ProbeHandler.HTTPGet.Port).Should(
					Equal(intstr.IntOrString{IntVal: int32(8081)}),
				)
			}
		})

		It("Update listener service", func() {
			emqx := generateEmqxBroker(brokerName, brokerNameSpace)
			servicePorts := []corev1.ServicePort{
				{
					Name:     "api",
					Port:     28081,
					NodePort: 30001,
					Protocol: "TCP",
					TargetPort: intstr.IntOrString{
						IntVal: 28081,
					},
				},
				{
					Name:     "mqtt",
					Port:     21883,
					NodePort: 30002,
					Protocol: "TCP",
					TargetPort: intstr.IntOrString{
						IntVal: 21883,
					},
				},
			}

			containerPorts := []corev1.ContainerPort{
				{
					Name:          "mqtt",
					Protocol:      "TCP",
					ContainerPort: 21883,
				},
				{
					Name:          "api",
					Protocol:      "TCP",
					ContainerPort: 28081,
				},
			}

			env := []corev1.EnvVar{
				{
					Name:  "EMQX_LISTENER__TCP__EXTERNAL",
					Value: "21883",
				},
				{
					Name:  "EMQX_MANAGEMENT__LISTENER__HTTP",
					Value: "28081",
				},
			}

			listener := v1beta3.Listener{
				Type:        corev1.ServiceTypeNodePort,
				Labels:      map[string]string{"test/labels": "service"},
				Annotations: map[string]string{"test/annotations": "service"},
				API: v1beta3.ListenerPort{
					Port:     int32(28081),
					NodePort: int32(30001),
				},
				MQTT: v1beta3.ListenerPort{
					Port:     int32(21883),
					NodePort: int32(30002),
				},
			}
			emqx.SetListener(listener)
			Expect(updateEmqx(emqx)).Should(Succeed())

			svc := &corev1.Service{}
			Eventually(func() corev1.ServiceType {
				_ = k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Name:      emqx.GetName(),
						Namespace: emqx.GetNamespace(),
					},
					svc,
				)
				return svc.Spec.Type
			}, timeout, interval).Should(Equal(corev1.ServiceTypeNodePort))
			Expect(svc.Spec.Ports).Should(ConsistOf(servicePorts))
			Expect(svc.Annotations).Should(HaveKeyWithValue("test/annotations", "service"))
			Expect(svc.Labels).Should(HaveKeyWithValue("test/labels", "service"))
			Expect(svc.Labels).Should(HaveKeyWithValue("cluster", "emqx"))

			sts := &appsv1.StatefulSet{}
			Eventually(func() []corev1.ContainerPort {
				_ = k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Name:      emqx.GetName(),
						Namespace: emqx.GetNamespace(),
					},
					sts,
				)
				return sts.Spec.Template.Spec.Containers[0].Ports
			}, timeout, interval).Should(ConsistOf(containerPorts))
			Expect(sts.Spec.Template.Spec.Containers[0].Env).Should(ContainElements(env))
		})
	})
})

func ports() ([]corev1.ServicePort, []corev1.ContainerPort, []corev1.EnvVar) {
	servicePorts := []corev1.ServicePort{
		{
			Name:     "mqtt",
			Port:     1883,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				IntVal: 1883,
			},
		},
		{
			Name:     "mqtts",
			Port:     8883,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				IntVal: 8883,
			},
		},
		{
			Name:     "ws",
			Port:     8083,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				IntVal: 8083,
			},
		},
		{
			Name:     "wss",
			Port:     8084,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				IntVal: 8084,
			},
		},
		{
			Name:     "dashboard",
			Port:     18083,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				IntVal: 18083,
			},
		},
		{
			Name:     "api",
			Port:     8081,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				IntVal: 8081,
			},
		},
	}

	containerPorts := []corev1.ContainerPort{
		{
			Name:          "mqtt",
			Protocol:      "TCP",
			ContainerPort: 1883,
		},
		{
			Name:          "mqtts",
			Protocol:      "TCP",
			ContainerPort: 8883,
		},
		{
			Name:          "ws",
			Protocol:      "TCP",
			ContainerPort: 8083,
		},
		{
			Name:          "wss",
			Protocol:      "TCP",
			ContainerPort: 8084,
		},
		{
			Name:          "dashboard",
			Protocol:      "TCP",
			ContainerPort: 18083,
		},
		{
			Name:          "api",
			Protocol:      "TCP",
			ContainerPort: 8081,
		},
	}

	env := []corev1.EnvVar{
		{
			Name:  "EMQX_LISTENER__TCP__EXTERNAL",
			Value: "1883",
		},
		{
			Name:  "EMQX_LISTENER__SSL__EXTERNAL",
			Value: "8883",
		},
		{
			Name:  "EMQX_LISTENER__WS__EXTERNAL",
			Value: "8083",
		},
		{
			Name:  "EMQX_LISTENER__WSS__EXTERNAL",
			Value: "8084",
		},
		{
			Name:  "EMQX_DASHBOARD__LISTENER__HTTP",
			Value: "18083",
		},
		{
			Name:  "EMQX_MANAGEMENT__LISTENER__HTTP",
			Value: "8081",
		},
	}

	return servicePorts, containerPorts, env
}
