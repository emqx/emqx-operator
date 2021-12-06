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

package suites_test

import (
	"context"

	"github.com/emqx/emqx-operator/api/v1beta1"
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
		BeforeEach(func() {
			// for _, emqx := range emqxList() {
			// 	Expect(k8sClient.Create(context.Background(), emqx)).Should(Succeed())
			// }
		})

		servicePorts, containerPorts, env := ports()

		It("Check headless service", func() {
			for _, emqx := range emqxList() {
				Expect(k8sClient.Create(context.Background(), emqx)).Should(Succeed())

				svc := &corev1.Service{}
				Eventually(func() bool {
					err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      emqx.GetHeadlessServiceName(),
							Namespace: emqx.GetNamespace(),
						},
						svc,
					)
					return err == nil
				}, tuneout, interval).Should(BeTrue())

				Expect(svc.Spec.Type).Should(Equal(corev1.ServiceTypeClusterIP))
				Expect(svc.Spec.ClusterIP).Should(Equal(corev1.ClusterIPNone))
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
				}, tuneout, interval).Should(BeTrue())
				Expect(sts.Spec.Template.Spec.Containers[0].Ports).Should(ConsistOf(containerPorts))
				Expect(sts.Spec.Template.Spec.Containers[0].Env).Should(ContainElements(env))
			}
		})

		It("Check listener service", func() {
			for _, emqx := range emqxList() {
				if broker, ok := emqx.(*v1beta1.EmqxBroker); ok {
					broker.Spec.Listener.Ports.MQTTS = int32(8884)
					Expect(k8sClient.Create(context.Background(), broker)).Should(Succeed())
				}
				if enterprise, ok := emqx.(*v1beta1.EmqxEnterprise); ok {
					enterprise.Spec.Listener.Ports.MQTTS = int32(8884)
					Expect(k8sClient.Create(context.Background(), enterprise)).Should(Succeed())
				}

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
				}, tuneout, interval).Should(BeTrue())

				Expect(svc.Spec.Ports).Should(
					ContainElements(
						corev1.ServicePort{
							Name:     "mqtts",
							Port:     8884,
							Protocol: "TCP",
							TargetPort: intstr.IntOrString{
								IntVal: 8884,
							},
						},
					),
				)

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
				}, tuneout, interval).Should(BeTrue())
				Expect(sts.Spec.Template.Spec.Containers[0].Ports).Should(
					ConsistOf(
						corev1.ContainerPort{
							Name:          "mqtts",
							Protocol:      "TCP",
							ContainerPort: 8884,
						},
					),
				)
				Expect(sts.Spec.Template.Spec.Containers[0].Env).Should(
					ContainElements(
						corev1.EnvVar{
							Name:  "EMQX_LISTENER__SSL__EXTERNAL",
							Value: "8884",
						},
					),
				)
			}
		})

		AfterEach(func() {
			for _, emqx := range emqxList() {
				Expect(deleteAll(emqx)).ToNot(HaveOccurred())
				Eventually(func() bool {
					return ensureDeleteAll(emqx)
				}, tuneout, interval).Should(BeTrue())
			}
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
