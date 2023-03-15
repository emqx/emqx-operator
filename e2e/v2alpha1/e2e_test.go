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

package v2alpha1

import (
	"context"

	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	gomegaTypes "github.com/onsi/gomega/types"

	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("E2E Test", func() {
	conditions := []gomegaTypes.GomegaMatcher{
		HaveField("Type", appsv2alpha1.ClusterRunning),
		HaveField("Type", appsv2alpha1.ClusterCoreReady),
		HaveField("Type", appsv2alpha1.ClusterCoreUpdating),
		HaveField("Type", appsv2alpha1.ClusterCreating),
	}
	listenerPorts := []corev1.ServicePort{
		{
			Name:       "tcp-default",
			Port:       1883,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromInt(1883),
		},
		{
			Name:       "lwm2m-udp-default",
			Port:       5783,
			Protocol:   corev1.ProtocolUDP,
			TargetPort: intstr.FromInt(5783),
		},
	}

	BeforeEach(func() {
		emqx := &appsv2alpha1.EMQX{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "e2e-test",
				Namespace: "e2e-test-v2alpha1",
			},
			Spec: appsv2alpha1.EMQXSpec{
				Image: "emqx:5.0",
				BootstrapAPIKeys: []appsv2alpha1.BootsrapAPIKey{
					{
						Key:    "test_key",
						Secret: "secret",
					},
				},
				BootstrapConfig: `
				gateway {
					"lwm2m" {
					  auto_observe = true
					  enable_stats = true
					  idle_timeout = "30s"
					  lifetime_max = "86400s"
					  lifetime_min = "1s"
					  listeners {
						udp {
						  default {
							bind = "5783"
							max_conn_rate = 1000
							max_connections = 1024000
						  }
						}
					  }
					  mountpoint = ""
					  qmode_time_window = "22s"
					  translators {
						command {qos = 0, topic = "dn/#"}
						notify {qos = 0, topic = "up/notify"}
						register {qos = 0, topic = "up/resp"}
						response {qos = 0, topic = "up/resp"}
						update {qos = 0, topic = "up/update"}
					  }
					  update_msg_publish_condition = "contains_object_list"
					  xml_dir = "etc/lwm2m_xml/"
					}
				  }
				`,
			},
		}
		emqx.Default()
		Expect(emqx.ValidateCreate()).Should(Succeed())

		Expect(k8sClient.Create(context.TODO(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "e2e-test-v2alpha1",
			},
		})).Should(Succeed())
		Expect(k8sClient.Create(context.TODO(), emqx)).Should(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(context.TODO(), &appsv2alpha1.EMQX{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "e2e-test",
				Namespace: "e2e-test-v2alpha1",
			},
		})).Should(Succeed())

		Expect(k8sClient.Delete(context.TODO(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "e2e-test-v2alpha1",
			},
		})).Should(Succeed())

		Eventually(func() bool {
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test-v2alpha1"}, &corev1.Namespace{})
			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	})

	Context("Check EMQX Custom Resource", func() {
		instance := &appsv2alpha1.EMQX{}

		It("", func() {
			By("Checking the EMQX Custom Resource's Status")
			for _, matcher := range []gomegaTypes.GomegaMatcher{
				HaveField("Conditions", ConsistOf(conditions)),
				HaveField("CurrentImage", Equal("emqx:5.0")),
				HaveField("EMQXNodes", HaveLen(4)),
				HaveField("CoreNodeReplicas", Equal(int32(1))),
				HaveField("CoreNodeReadyReplicas", Equal(int32(1))),
				HaveField("ReplicantNodeReplicas", Equal(int32(3))),
				HaveField("ReplicantNodeReadyReplicas", Equal(int32(3))),
			} {
				Eventually(func() appsv2alpha1.EMQXStatus {
					_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test", Namespace: "e2e-test-v2alpha1"}, instance)
					return instance.Status
				}, timeout, interval).Should(matcher)
			}

			By("Checking the EMQX Custom Resource's Service")
			svc := &corev1.Service{}
			Eventually(func() []corev1.ServicePort {
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test-listeners", Namespace: "e2e-test-v2alpha1"}, svc)
				return svc.Spec.Ports
			}, timeout, interval).Should(ConsistOf(listenerPorts))

			By("Checking the EMQX Custom Resource's EndpointSlice")
			ep := &discoveryv1.EndpointSlice{}
			Eventually(func() []discoveryv1.Endpoint {
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test-listeners", Namespace: "e2e-test-v2alpha1"}, ep)
				return ep.Endpoints
			}, timeout, interval).Should(HaveLen(3))
			Eventually(func() []discoveryv1.EndpointPort {
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test-listeners", Namespace: "e2e-test-v2alpha1"}, ep)
				return ep.Ports
			}, timeout, interval).Should(ConsistOf([]discoveryv1.EndpointPort{
				{
					Name:     &[]string{listenerPorts[0].Name}[0],
					Port:     &[]int32{listenerPorts[0].Port}[0],
					Protocol: &[]corev1.Protocol{listenerPorts[0].Protocol}[0],
				},
				{
					Name:     &[]string{listenerPorts[1].Name}[0],
					Port:     &[]int32{listenerPorts[1].Port}[0],
					Protocol: &[]corev1.Protocol{listenerPorts[1].Protocol}[0],
				},
			}))
		})
	})

	Context("Update EMQX Custom Resource, change image", func() {
		instance := &appsv2alpha1.EMQX{}
		JustBeforeEach(func() {
			By("Wait EMQX cluster ready")
			Eventually(func() []appsv2alpha1.Condition {
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test", Namespace: "e2e-test-v2alpha1"}, instance)
				return instance.Status.Conditions
			}, timeout, interval).Should(ConsistOf(conditions))
			By("Update the EMQX Custom Resource, change image")
			Eventually(func() error {
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test", Namespace: "e2e-test-v2alpha1"}, instance)
				replicant := int32(3)
				instance.Spec.ReplicantTemplate.Spec.Replicas = &replicant
				instance.Spec.Image = "emqx:5"
				return k8sClient.Update(context.TODO(), instance)
			}, timeout, interval).Should(Succeed())
		})

		It("Checking the EMQX Custom Resource's Status when update image", func() {
			By("Checking statefulSet image")
			Eventually(func() string {
				sts := &appsv1.StatefulSet{}
				err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test-core", Namespace: "e2e-test-v2alpha1"}, sts)
				if err != nil {
					return ""
				}
				return sts.Spec.Template.Spec.Containers[0].Image
			}, timeout, interval).Should(Equal("emqx:5"))

			By("Checking deployment image")
			Eventually(func() string {
				deploy := &appsv1.Deployment{}
				err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test-replicant", Namespace: "e2e-test-v2alpha1"}, deploy)
				if err != nil {
					return ""
				}
				return deploy.Spec.Template.Spec.Containers[0].Image
			}, timeout, interval).Should(Equal("emqx:5"))

			By("Checking the EMQX Custom Resource's Status")
			for _, matcher := range []gomegaTypes.GomegaMatcher{
				HaveField("Conditions", ConsistOf(conditions)),
				HaveField("CurrentImage", Equal("emqx:5")),
				HaveField("EMQXNodes", HaveLen(4)),
				HaveField("CoreNodeReplicas", Equal(int32(1))),
				HaveField("CoreNodeReadyReplicas", Equal(int32(1))),
				HaveField("ReplicantNodeReplicas", Equal(int32(3))),
				HaveField("ReplicantNodeReadyReplicas", Equal(int32(3))),
			} {
				Eventually(func() appsv2alpha1.EMQXStatus {
					_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test", Namespace: "e2e-test-v2alpha1"}, instance)
					return instance.Status
				}, timeout, interval).Should(matcher)
			}
		})
	})

})
