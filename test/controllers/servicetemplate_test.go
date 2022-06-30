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
var _ = Describe("Check service", func() {
	Context("Check service", func() {
		ports := []corev1.ServicePort{
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
		It("should create a service", func() {
			check_service_ports(types.NamespacedName{Name: "emqx", Namespace: broker.Namespace}, ports)
			check_service_ports(types.NamespacedName{Name: "emqx-ee", Namespace: enterprise.Namespace}, ports)
		})
	})

	Context("Check update service", func() {
		var ports []corev1.ServicePort
		JustBeforeEach(func() {
			ports = []corev1.ServicePort{
				{
					Name:       "mqtt-tcp-11883",
					Port:       11883,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(11883),
				},
				{
					Name:       "mqtt-tcp-21883",
					Port:       21883,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(21883),
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
			for _, emqx := range []v1beta3.Emqx{broker, enterprise} {
				config := emqx.GetEmqxConfig()
				config["listener.tcp.internal"] = "11883"
				config["listener.tcp.external"] = "21883"
				emqx.SetEmqxConfig(config)
				updateEmqx(emqx)
			}
		})

		It("should create a service", func() {
			check_service_ports(types.NamespacedName{Name: "emqx", Namespace: broker.Namespace}, ports)
			check_service_ports(types.NamespacedName{Name: "emqx-ee", Namespace: enterprise.Namespace}, ports)
		})
	})
})

func check_service_ports(namespacedName types.NamespacedName, ports []corev1.ServicePort) {
	Eventually(func() []corev1.ServicePort {
		svc := &corev1.Service{}
		_ = k8sClient.Get(
			context.Background(),
			namespacedName,
			svc,
		)
		return svc.Spec.Ports
	}, timeout, interval).Should(ContainElements(ports))
}
