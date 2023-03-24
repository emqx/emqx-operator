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
	"sort"

	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	appscontrollersv2alpha1 "github.com/emqx/emqx-operator/controllers/apps/v2alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegaTypes "github.com/onsi/gomega/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("Base Test", func() {
	BeforeEach(func() {
		createResource(emqx.DeepCopy())
	})

	AfterEach(func() {
		deleteResource(emqx.DeepCopy())
	})

	Context("Check EMQX Custom Resource", Label("base"), func() {
		instance := &appsv2alpha1.EMQX{}

		It("Base Check", func() {
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), instance)).Should(Succeed())

			By("Checking the EMQX Custom Resource's Pod and EndpointSlice", func() {
				checkPodAndEndpointSlices(instance, 2)
			})

			By("Checking the EMQX Custom Resource's Service", func() {
				checkService(instance)
			})

			By("Checking the EMQX Custom Resource's Status", func() {
				checkRunning(instance.DeepCopy())
			})
		})
	})

	Context("Direct Update", Label("update"), func() {
		instance := &appsv2alpha1.EMQX{}
		JustBeforeEach(func() {
			By("Wait EMQX cluster ready")
			checkRunning(emqx.DeepCopy())

			By("change replicas, will trigger direct update")
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), instance)).Should(Succeed())
			instance.Spec.ReplicantTemplate.Spec.Replicas = pointer.Int32(3)
			Expect(k8sClient.Update(context.TODO(), instance)).Should(Succeed())
		})

		It("Check Direct Update", func() {
			By("Checking just once deployment", func() {
				var deployments *appsv1.DeploymentList
				Eventually(func() int {
					deployments = &appsv1.DeploymentList{}
					_ = k8sClient.List(context.TODO(), deployments,
						client.InNamespace(instance.Namespace),
						client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
					)
					return len(deployments.Items)
				}, timeout, interval).Should(Equal(1))

				Expect(deployments.Items[0].Status.Replicas).Should(Equal(instance.Status.ReplicantNodeReplicas))
			})

			By("Checking the EMQX Custom Resource's Pod and EndpointSlice", func() {
				checkPodAndEndpointSlices(instance, 3)
			})

			By("Checking the EMQX Custom Resource's Service", func() {
				checkService(instance)
			})
		})
	})

	Context("Blue Green Update", Label("blue"), func() {
		instance := &appsv2alpha1.EMQX{}
		currentImage := "emqx:5"
		JustBeforeEach(func() {
			By("Wait EMQX cluster ready")
			checkRunning(emqx.DeepCopy())

			By("Change image, will trigger blue green update")
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), instance)).Should(Succeed())
			instance.Spec.Image = currentImage
			Expect(k8sClient.Update(context.TODO(), instance)).Should(Succeed())
		})

		It("Check Blue Green Update", func() {
			By("Checking statefulSet image", func() {
				sts := &appsv1.StatefulSet{}
				Eventually(func() string {
					_ = k8sClient.Get(context.TODO(), client.ObjectKey{
						Namespace: instance.Namespace,
						Name:      instance.Spec.CoreTemplate.Name,
					}, sts)
					return sts.Spec.Template.Spec.Containers[0].Image
				}, timeout, interval).Should(Equal(currentImage))
			})

			By("Checking deployment list", func() {
				var dList []*appsv1.Deployment
				Eventually(func() int {
					deployments := &appsv1.DeploymentList{}
					_ = k8sClient.List(context.TODO(), deployments,
						client.InNamespace(instance.Namespace),
						client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
					)
					dList = []*appsv1.Deployment{}
					for _, d := range deployments.Items {
						dList = append(dList, d.DeepCopy())
					}
					return len(dList)
				}, timeout, interval).Should(Equal(2))

				sort.Sort(appscontrollersv2alpha1.DeploymentsByCreationTimestamp(dList))

				old := dList[0].DeepCopy()
				Expect(old.Spec.Template.Spec.Containers[0].Image).Should(Equal(emqx.DeepCopy().Spec.Image))
				Eventually(func() int32 {
					_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(old), old)
					return old.Status.Replicas
				}, timeout, interval).Should(Equal(int32(0)))

				new := dList[1].DeepCopy()
				Expect(new.Spec.Template.Spec.Containers[0].Image).Should(Equal(currentImage))
				Eventually(func() int32 {
					_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(new), new)
					return new.Status.Replicas
				}, timeout, interval).Should(Equal(instance.Status.ReplicantNodeReplicas))
			})

			By("Checking endpointScales list", func() {
				checkPodAndEndpointSlices(instance, 2)
			})

			By("Checking the EMQX Custom Resource's Service", func() {
				checkService(instance)
			})

			By("Checking the EMQX Custom Resource status", func() {
				Eventually(func() string {
					_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(instance), instance)
					return instance.Status.CurrentImage
				}).Should(Equal("emqx:5"))
				checkRunning(instance.DeepCopy())
			})
		})
	})
})

func createResource(instance *appsv2alpha1.EMQX) {
	instance.Default()
	Expect(instance.ValidateCreate()).Should(Succeed())
	Expect(k8sClient.Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: instance.GetNamespace(),
			Labels: map[string]string{
				"test": "e2e",
			},
		},
	})).Should(Succeed())
	Expect(k8sClient.Create(context.TODO(), instance)).Should(Succeed())
}

func deleteResource(instance *appsv2alpha1.EMQX) {
	Expect(k8sClient.Delete(context.TODO(), instance)).Should(Succeed())

	Expect(k8sClient.Delete(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: instance.GetNamespace(),
		},
	})).Should(Succeed())

	Eventually(func() bool {
		err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: instance.GetNamespace()}, &corev1.Namespace{})
		return k8sErrors.IsNotFound(err)
	}, timeout, interval).Should(BeTrue())
}

func checkPodAndEndpointSlices(instance *appsv2alpha1.EMQX, count int) {
	podList := &corev1.PodList{}
	Eventually(func() []corev1.Pod {
		_ = k8sClient.List(context.TODO(), podList,
			client.InNamespace(instance.Namespace),
			client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
		)
		return podList.Items
	}, timeout, interval).Should(
		And(
			HaveLen(count),
			HaveEach(
				HaveField("Status", And(
					HaveField("Phase", corev1.PodRunning),
					HaveField("Conditions", ContainElements(
						HaveField("Type", appsv2alpha1.PodInCluster),
						HaveField("Type", corev1.PodReady),
					)))),
			),
		),
	)

	matchers := []gomegaTypes.GomegaMatcher{}
	for _, p := range podList.Items {
		pod := p.DeepCopy()
		m := And(
			HaveField("Addresses", ConsistOf([]string{pod.Status.PodIP})),
			HaveField("NodeName", HaveValue(Equal(pod.Spec.NodeName))),
			HaveField("Conditions", And(
				HaveField("Ready", HaveValue(BeTrue())),
				HaveField("Serving", HaveValue(BeTrue())),
				HaveField("Terminating", BeNil()),
			)),
			HaveField("TargetRef", And(
				HaveField("Kind", "Pod"),
				HaveField("UID", pod.GetUID()),
				HaveField("Name", pod.GetName()),
				HaveField("Namespace", pod.GetNamespace()),
			)),
		)
		matchers = append(matchers, m)
	}

	Eventually(func() *discoveryv1.EndpointSlice {
		ep := &discoveryv1.EndpointSlice{}
		_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.ListenersServiceTemplate.Name, Namespace: instance.Namespace}, ep)
		return ep
	}, timeout, interval).Should(
		And(
			HaveField("Endpoints", ConsistOf(matchers)),
			HaveField("Ports", ConsistOf([]discoveryv1.EndpointPort{
				{
					Name:     pointer.String("tcp-default"),
					Port:     pointer.Int32(1883),
					Protocol: &[]corev1.Protocol{corev1.ProtocolTCP}[0],
				},
				{
					Name:     pointer.String("lwm2m-udp-default"),
					Port:     pointer.Int32(5783),
					Protocol: &[]corev1.Protocol{corev1.ProtocolUDP}[0],
				},
			})),
		),
	)
}

func checkService(instance *appsv2alpha1.EMQX) {
	svc := &corev1.Service{}
	Eventually(func() []corev1.ServicePort {
		_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.ListenersServiceTemplate.Name, Namespace: instance.Namespace}, svc)
		return svc.Spec.Ports
	}, timeout, interval).Should(ConsistOf([]corev1.ServicePort{
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
	}))
}

func checkRunning(instance *appsv2alpha1.EMQX) {
	Eventually(func() appsv2alpha1.EMQXStatus {
		_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(instance), instance)
		return instance.Status
	}, timeout, interval).Should(
		And(
			HaveField("EMQXNodes", HaveLen(4)),
			HaveField("CoreNodeReplicas", Equal(int32(2))),
			HaveField("CoreNodeReadyReplicas", Equal(int32(2))),
			HaveField("ReplicantNodeReplicas", Equal(int32(2))),
			HaveField("ReplicantNodeReadyReplicas", Equal(int32(2))),
			HaveField("Conditions", ConsistOf(
				HaveField("Type", appsv2alpha1.ClusterRunning),
				HaveField("Type", appsv2alpha1.ClusterCoreReady),
				HaveField("Type", appsv2alpha1.ClusterCoreUpdating),
				HaveField("Type", appsv2alpha1.ClusterCreating),
			)),
		),
	)
}
