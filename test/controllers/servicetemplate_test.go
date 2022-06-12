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

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var _ = Describe("", func() {
	Context("Check service", func() {
		It("Check headless service", func() {
			for _, emqx := range emqxList() {
				names := v1beta3.Names{Object: emqx}
				Eventually(func() []corev1.ServicePort {
					svc := &corev1.Service{}
					_ = k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      names.HeadlessSvc(),
							Namespace: emqx.GetNamespace(),
						},
						svc,
					)
					return svc.Spec.Ports
				}, timeout, interval).Should(ConsistOf([]corev1.ServicePort{{
					Name:       "management-listener-http",
					Port:       8081,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8081),
				}}))
			}
		})

		It("Check listener service", func() {
			for _, emqx := range emqxList() {
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
				}, timeout, interval).Should(ConsistOf([]corev1.ServicePort{
					{
						Name:       "management-listener-http",
						Port:       8081,
						Protocol:   corev1.ProtocolTCP,
						TargetPort: intstr.FromInt(8081),
					},
					{
						Name:       "dashboard-listener-http",
						Port:       18083,
						Protocol:   corev1.ProtocolTCP,
						TargetPort: intstr.FromInt(18083),
					},
					{
						Name:       "listener-tcp-external",
						Port:       1883,
						Protocol:   corev1.ProtocolTCP,
						TargetPort: intstr.FromInt(1883),
					},
					{
						Name:       "listener-ssl-external",
						Port:       8883,
						Protocol:   corev1.ProtocolTCP,
						TargetPort: intstr.FromInt(8883),
					},
					{
						Name:       "listener-ws-external",
						Port:       8083,
						Protocol:   corev1.ProtocolTCP,
						TargetPort: intstr.FromInt(8083),
					},
					{
						Name:       "listener-wss-external",
						Port:       8084,
						Protocol:   corev1.ProtocolTCP,
						TargetPort: intstr.FromInt(8084),
					},
				}))
			}
		})

		It("Update listener service", func() {
			emqx := generateEmqxBroker(brokerName, brokerNameSpace)

			config := make(v1beta3.EmqxConfig)
			config["listener.tcp.external"] = "21883"
			emqx.SetEmqxConfig(config)
			Expect(updateEmqx(emqx)).Should(Succeed())

			svc := &corev1.Service{}
			Eventually(func() []corev1.ServicePort {
				_ = k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Name:      emqx.GetName(),
						Namespace: emqx.GetNamespace(),
					},
					svc,
				)
				return svc.Spec.Ports
			}, timeout, interval).Should(ContainElements([]corev1.ServicePort{
				{
					Name:       "listener-tcp-external",
					Port:       21883,
					Protocol:   "TCP",
					TargetPort: intstr.FromInt(21883),
				},
			}))
		})
	})
})
