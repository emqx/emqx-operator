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

	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("E2E Test", func() {
	listenerPorts := []corev1.ServicePort{
		{
			Name:       "tcp-default",
			Port:       1883,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromInt(1883),
		},
		{
			Name:       "ssl-default",
			Port:       8883,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromInt(8883),
		},
		{
			Name:       "ws-default",
			Port:       8083,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromInt(8083),
		},
		{
			Name:       "wss-default",
			Port:       8084,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromInt(8084),
		},
	}
	Context("Replicant not exist", func() {
		instance := &appsv2alpha1.EMQX{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "e2e-test",
				Namespace: "default",
			},
			Spec: appsv2alpha1.EMQXSpec{
				Image: "emqx/emqx:5.0.3",
			},
		}
		BeforeEach(func() {
			instance.Default()
			Expect(k8sClient.Create(context.TODO(), instance)).Should(Succeed())
		})
		It("Check EMQX Status", func() {
			Eventually(func() corev1.ConditionStatus {
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test", Namespace: "default"}, instance)
				running := corev1.ConditionFalse
				for _, c := range instance.Status.Conditions {
					if c.Type == appsv2alpha1.ClusterRunning {
						running = c.Status
					}
				}
				return running
			}, timeout, interval).Should(Equal(corev1.ConditionTrue))

			Expect(instance.Status.NodeStatuses).Should(HaveLen(3))
			Expect(instance.Status.CoreReplicas).Should(Equal(int32(3)))
			Expect(instance.Status.ReadyCoreReplicas).Should(Equal(int32(3)))
			Expect(instance.Status.ReplicantReplicas).Should(Equal(int32(0)))
			Expect(instance.Status.ReadyReplicantReplicas).Should(Equal(int32(0)))

			svc := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test-listeners", Namespace: "default"}, svc)
			}, timeout, interval).Should(Succeed())

			Expect(svc.Spec.Ports).Should(ConsistOf(listenerPorts))
			Expect(svc.Spec.Selector).Should(HaveKeyWithValue("apps.emqx.io/db-role", "core"))
		})
		AfterEach(func() {
			Expect(k8sClient.Delete(context.TODO(), instance)).Should(Succeed())
		})
	})

	Context("Replicant exist", func() {
		replicant := int32(3)
		instance := &appsv2alpha1.EMQX{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "e2e-test",
				Namespace: "default",
			},
			Spec: appsv2alpha1.EMQXSpec{
				Image: "emqx/emqx:5.0.3",
				ReplicantTemplate: appsv2alpha1.EMQXReplicantTemplate{
					Spec: appsv2alpha1.EMQXReplicantTemplateSpec{
						Replicas: &replicant,
					},
				},
			},
		}
		BeforeEach(func() {
			instance.Default()
			Expect(k8sClient.Create(context.TODO(), instance)).Should(Succeed())
		})
		It("Check EMQX Status", func() {
			Eventually(func() corev1.ConditionStatus {
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test", Namespace: "default"}, instance)
				running := corev1.ConditionFalse
				for _, c := range instance.Status.Conditions {
					if c.Type == appsv2alpha1.ClusterRunning {
						running = c.Status
					}
				}
				return running
			}, timeout, interval).Should(Equal(corev1.ConditionTrue))

			Expect(instance.Status.NodeStatuses).Should(HaveLen(6))
			Expect(instance.Status.CoreReplicas).Should(Equal(int32(3)))
			Expect(instance.Status.ReadyCoreReplicas).Should(Equal(int32(3)))
			Expect(instance.Status.ReplicantReplicas).Should(Equal(int32(3)))
			Expect(instance.Status.ReadyReplicantReplicas).Should(Equal(int32(3)))

			svc := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test-listeners", Namespace: "default"}, svc)
			}, timeout, interval).Should(Succeed())

			Expect(svc.Spec.Ports).Should(ConsistOf(listenerPorts))
			Expect(svc.Spec.Selector).Should(HaveKeyWithValue("apps.emqx.io/db-role", "replicant"))
		})
		AfterEach(func() {
			Expect(k8sClient.Delete(context.TODO(), instance)).Should(Succeed())
		})
	})
})
