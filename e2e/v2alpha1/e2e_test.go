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
				Namespace: "e2e-test",
			},
			Spec: appsv2alpha1.EMQXSpec{
				Image: "emqx/emqx:5.0.9",
				BootstrapConfig: `
				log{
					console_handler: {
						enable: true
						level: debug
					}
				}
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
				Name: "e2e-test",
			},
		})).Should(Succeed())
		Expect(k8sClient.Create(context.TODO(), emqx)).Should(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(context.TODO(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "e2e-test",
			},
		})).Should(Succeed())

		Eventually(func() bool {
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test"}, &corev1.Namespace{})
			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	})

	Context("Check EMQX Custom Resource", func() {
		instance := &appsv2alpha1.EMQX{}

		It("", func() {
			By("Checking the EMQX Custom Resource's Status")
			Eventually(func() []appsv2alpha1.Condition {
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test", Namespace: "e2e-test"}, instance)
				return instance.Status.Conditions
			}, timeout, interval).Should(ConsistOf(conditions))

			Expect(instance.Status.CurrentImage).Should(Equal("emqx/emqx:5.0.9"))
			Expect(instance.Status.EMQXNodes).Should(HaveLen(4))
			Expect(instance.Status.CoreNodeReplicas).Should(Equal(int32(1)))
			Expect(instance.Status.CoreNodeReadyReplicas).Should(Equal(int32(1)))
			Expect(instance.Status.ReplicantNodeReplicas).Should(Equal(int32(3)))
			Expect(instance.Status.ReplicantNodeReadyReplicas).Should(Equal(int32(3)))

			By("Checking the EMQX Custom Resource's Service")
			svc := &corev1.Service{}
			Eventually(func() map[string]string {
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test-listeners", Namespace: "e2e-test"}, svc)
				return svc.Spec.Selector
			}, timeout, interval).Should(HaveKeyWithValue("apps.emqx.io/db-role", "replicant"))

			Expect(svc.Spec.Ports).Should(ConsistOf(listenerPorts))

		})
	})

	Context("Check EMQX Custom Resource when replicant replicas == 0", func() {
		instance := &appsv2alpha1.EMQX{}
		JustBeforeEach(func() {
			By("Update the EMQX Custom Resource, add replicant nodes")
			Eventually(func() error {
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test", Namespace: "e2e-test"}, instance)
				replicant := int32(0)
				instance.Spec.ReplicantTemplate.Spec.Replicas = &replicant
				return k8sClient.Update(context.TODO(), instance)
			}, timeout, interval).Should(Succeed())
		})
		It("", func() {
			By("Checking the EMQX Custom Resource's Status")
			Eventually(func() []appsv2alpha1.Condition {
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test", Namespace: "e2e-test"}, instance)
				return instance.Status.Conditions
			}, timeout, interval).Should(ConsistOf(conditions))

			Expect(instance.Status.CurrentImage).Should(Equal("emqx/emqx:5.0.9"))
			Expect(instance.Status.EMQXNodes).Should(HaveLen(1))
			Expect(instance.Status.CoreNodeReplicas).Should(Equal(int32(1)))
			Expect(instance.Status.CoreNodeReadyReplicas).Should(Equal(int32(1)))
			Expect(instance.Status.ReplicantNodeReplicas).Should(Equal(int32(0)))
			Expect(instance.Status.ReplicantNodeReadyReplicas).Should(Equal(int32(0)))

			By("Checking the EMQX Custom Resource's Service")
			svc := &corev1.Service{}
			Eventually(func() map[string]string {
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test-listeners", Namespace: "e2e-test"}, svc)
				return svc.Spec.Selector
			}, timeout, interval).Should(HaveKeyWithValue("apps.emqx.io/db-role", "core"))

			Expect(svc.Spec.Ports).Should(ConsistOf(listenerPorts))

		})
	})

	Context("Update EMQX Custom Resource, change image", func() {
		instance := &appsv2alpha1.EMQX{}
		JustBeforeEach(func() {
			By("Wait EMQX cluster ready")
			Eventually(func() []appsv2alpha1.Condition {
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test", Namespace: "e2e-test"}, instance)
				return instance.Status.Conditions
			}, timeout, interval).Should(ConsistOf(conditions))
			By("Update the EMQX Custom Resource, change image")
			Eventually(func() error {
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test", Namespace: "e2e-test"}, instance)
				replicant := int32(3)
				instance.Spec.ReplicantTemplate.Spec.Replicas = &replicant
				instance.Spec.Image = "emqx/emqx:5.0.8"
				return k8sClient.Update(context.TODO(), instance)
			}, timeout, interval).Should(Succeed())
		})

		It("Checking the EMQX Custom Resource's Status when update image", func() {
			By("Checking statefulSet image")
			Eventually(func() string {
				sts := &appsv1.StatefulSet{}
				err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test-core", Namespace: "e2e-test"}, sts)
				if err != nil {
					return ""
				}
				return sts.Spec.Template.Spec.Containers[0].Image
			}, timeout, interval).Should(Equal("emqx/emqx:5.0.8"))

			By("Checking deployment image")
			Eventually(func() string {
				deploy := &appsv1.Deployment{}
				err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test-replicant", Namespace: "e2e-test"}, deploy)
				if err != nil {
					return ""
				}
				return deploy.Spec.Template.Spec.Containers[0].Image
			}, timeout, interval).Should(Equal("emqx/emqx:5.0.8"))

			By("Checking the EMQX Custom Resource's Status")
			Eventually(func() []appsv2alpha1.Condition {
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test", Namespace: "e2e-test"}, instance)
				return instance.Status.Conditions
			}, timeout, interval).Should(ConsistOf(conditions))

			Expect(instance.Status.CurrentImage).Should(Equal("emqx/emqx:5.0.8"))
			Expect(instance.Status.EMQXNodes).Should(HaveLen(4))
			Expect(instance.Status.CoreNodeReplicas).Should(Equal(int32(1)))
			Expect(instance.Status.CoreNodeReadyReplicas).Should(Equal(int32(1)))
			Expect(instance.Status.ReplicantNodeReplicas).Should(Equal(int32(3)))
			Expect(instance.Status.ReplicantNodeReadyReplicas).Should(Equal(int32(3)))
		})
	})

})
