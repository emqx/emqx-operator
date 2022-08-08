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
	appsv1 "k8s.io/api/apps/v1"
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

	BeforeEach(func() {
		emqx := &appsv2alpha1.EMQX{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "e2e-test",
				Namespace: "default",
			},
			Spec: appsv2alpha1.EMQXSpec{
				Image: "emqx/emqx:5.0.4",
			},
		}
		emqx.Default()
		Expect(k8sClient.Create(context.TODO(), emqx)).Should(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(context.TODO(), &appsv2alpha1.EMQX{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "e2e-test",
				Namespace: "default",
			},
		})).Should(Succeed())
	})

	Context("Check EMQX Custom Resource when replicant replicas == 0", func() {
		instance := &appsv2alpha1.EMQX{}
		It("", func() {
			By("Checking the EMQX Custom Resource's Status")
			Eventually(func() bool {
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test", Namespace: "default"}, instance)
				return instance.Status.IsRunning()
			}, timeout, interval).Should(BeTrue())

			Expect(instance.Status.EmqxNodes).Should(HaveLen(3))
			Expect(instance.Status.CoreReplicas).Should(Equal(int32(3)))
			Expect(instance.Status.ReadyCoreReplicas).Should(Equal(int32(3)))
			Expect(instance.Status.ReplicantReplicas).Should(Equal(int32(0)))
			Expect(instance.Status.ReadyReplicantReplicas).Should(Equal(int32(0)))

			By("Checking the EMQX Custom Resource's Service")
			svc := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test-listeners", Namespace: "default"}, svc)
			}, timeout, interval).Should(Succeed())

			Expect(svc.Spec.Ports).Should(ConsistOf(listenerPorts))
			Expect(svc.Spec.Selector).Should(HaveKeyWithValue("apps.emqx.io/db-role", "core"))
		})
	})

	Context("Check EMQX Custom Resource when replicant replicas != 0", func() {
		instance := &appsv2alpha1.EMQX{}
		JustBeforeEach(func() {
			By("Update the EMQX Custom Resource, add replicant nodes")
			Eventually(func() error {
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test", Namespace: "default"}, instance)
				replicant := int32(3)
				instance.Spec.ReplicantTemplate.Spec.Replicas = &replicant
				return k8sClient.Update(context.TODO(), instance)
			}, timeout, interval).Should(Succeed())
		})

		It("", func() {
			By("Checking the EMQX Custom Resource's Status when replicant nodes exist")
			Eventually(func() bool {
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test", Namespace: "default"}, instance)
				return instance.Status.IsRunning()
			}, timeout, interval).Should(BeTrue())

			Expect(instance.Status.EmqxNodes).Should(HaveLen(6))
			Expect(instance.Status.CoreReplicas).Should(Equal(int32(3)))
			Expect(instance.Status.ReadyCoreReplicas).Should(Equal(int32(3)))
			Expect(instance.Status.ReplicantReplicas).Should(Equal(int32(3)))
			Expect(instance.Status.ReadyReplicantReplicas).Should(Equal(int32(3)))

			By("Checking the EMQX Custom Resource's Service when replicant nodes exist")
			svc := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test-listeners", Namespace: "default"}, svc)
			}, timeout, interval).Should(Succeed())

			Expect(svc.Spec.Ports).Should(ConsistOf(listenerPorts))
			Expect(svc.Spec.Selector).Should(HaveKeyWithValue("apps.emqx.io/db-role", "replicant"))
		})
	})

	Context("Update EMQX Custom Resource, change image version", func() {
		instance := &appsv2alpha1.EMQX{}
		JustBeforeEach(func() {
			Eventually(func() error {
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test", Namespace: "default"}, instance)
				replicant := int32(3)
				instance.Spec.ReplicantTemplate.Spec.Replicas = &replicant
				instance.Spec.Image = "emqx/emqx:5.0.3"
				return k8sClient.Update(context.TODO(), instance)
			}, timeout, interval).Should(Succeed())
		})

		It("Checking the EMQX Custom Resource's Status when update image", func() {
			Eventually(func() bool {
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test", Namespace: "default"}, instance)
				return instance.Status.IsRunning()
			}, timeout, interval).Should(BeTrue())

			Expect(instance.Status.Conditions).Should(ContainElement(HaveField("status", appsv2alpha1.ClusterCoreUpdating)))
			Expect(instance.Status.Conditions).Should(ContainElement(HaveField("status", appsv2alpha1.ClusterReplicantUpdating)))
			Expect(instance.Status.CurrentImage).Should(Equal("emqx/emqx:5.0.3"))
			Expect(instance.Status.OriginalImage).Should(Equal("emqx/emqx:5.0.3"))

			Eventually(func() string {
				sts := &appsv1.StatefulSet{}
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test-core", Namespace: "default"}, sts)
				return sts.Spec.Template.Spec.Containers[0].Image
			}, interval, timeout).Should(Equal("emqx/emqx:5.0.3"))

			Eventually(func() string {
				deploy := &appsv1.Deployment{}
				_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: "e2e-test-replicant", Namespace: "default"}, deploy)
				return deploy.Spec.Template.Spec.Containers[0].Image
			}, interval, timeout).Should(Equal("emqx/emqx:5.0.3"))
		})
	})

})
