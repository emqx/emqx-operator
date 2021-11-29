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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/emqx/emqx-operator/api/v1alpha2"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var _ = Describe("", func() {
	Context("Check service", func() {
		BeforeEach(func() {
			ctx := context.Background()

			sa, role, roleBinding := GenerateRBAC(BrokerName, BrokerNameSpace)
			Expect(k8sClient.Create(ctx, sa)).Should(Succeed())
			Expect(k8sClient.Create(ctx, role)).Should(Succeed())
			Expect(k8sClient.Create(ctx, roleBinding)).Should(Succeed())
		})

		It("Check headless service", func() {
			ctx := context.Background()
			broker := GenerateEmqxBroker(BrokerName, BrokerNameSpace)
			Expect(k8sClient.Create(ctx, broker)).Should(Succeed())

			svc := &corev1.Service{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: broker.GetHeadlessServiceName(), Namespace: broker.GetNamespace()}, svc)
				if err != nil {
					return false
				}
				return true
			}, Timeout, Interval).Should(BeTrue())

			Expect(svc.Spec.Type).Should(Equal(corev1.ServiceTypeClusterIP))
			Expect(svc.Spec.ClusterIP).Should(Equal(corev1.ClusterIPNone))
			Expect(svc.Spec.Ports).Should(
				ConsistOf([]corev1.ServicePort{
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
				}),
			)
		})

		It("Check listener service", func() {
			ctx := context.Background()
			broker := GenerateEmqxBroker(BrokerName, BrokerNameSpace)
			broker.Spec.Listener.Ports.MQTTS = int32(8884)
			Expect(k8sClient.Create(ctx, broker)).Should(Succeed())

			svc := &corev1.Service{}
			Eventually(func() bool {
				err := k8sClient.Get(
					ctx,
					types.NamespacedName{
						Name:      broker.GetName(),
						Namespace: broker.GetNamespace(),
					},
					svc,
				)
				if err != nil {
					return false
				}
				return true
			}, Timeout, Interval).Should(BeTrue())

			Expect(svc.Spec.Ports).Should(ContainElements(corev1.ServicePort{
				Name:     "mqtts",
				Port:     8884,
				Protocol: "TCP",
				TargetPort: intstr.IntOrString{
					IntVal: 8884,
				},
			}))
		})

		AfterEach(func() {
			emqx := &v1alpha2.EmqxBroker{
				ObjectMeta: metav1.ObjectMeta{
					Name:      BrokerName,
					Namespace: BrokerNameSpace,
				},
			}
			Expect(DeleteAll(emqx)).ToNot(HaveOccurred())
			Eventually(func() bool {
				return EnsureDeleteAll(emqx)
			}, Timeout, Interval).Should(BeTrue())
		})
	})
})
