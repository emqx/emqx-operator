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

package v2alpha2

import (
	"context"
	"sort"

	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	appscontrollersv2alpha2 "github.com/emqx/emqx-operator/controllers/apps/v2alpha2"
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
		instance := &appsv2alpha2.EMQX{}

		It("Base Check", func() {
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), instance)).Should(Succeed())

			By("Checking the EMQX Custom Resource's Pod and EndpointSlice", func() {
				checkPodAndEndpointsAndEndpointSlices(instance, 2)
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
		instance := &appsv2alpha2.EMQX{}
		JustBeforeEach(func() {
			By("Wait EMQX cluster ready")
			checkRunning(emqx.DeepCopy())

			By("change replicas, will trigger direct update")
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), instance)).Should(Succeed())
			instance.Spec.ReplicantTemplate.Spec.Replicas = pointer.Int32(3)
			Expect(k8sClient.Update(context.TODO(), instance)).Should(Succeed())
		})

		It("Check Direct Update", func() {
			By("Checking just once replicaSet", func() {
				var replicaSets *appsv1.ReplicaSetList
				Eventually(func() int {
					replicaSets = &appsv1.ReplicaSetList{}
					_ = k8sClient.List(context.TODO(), replicaSets,
						client.InNamespace(instance.Namespace),
						client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
					)
					return len(replicaSets.Items)
				}, timeout, interval).Should(Equal(1))

				Expect(replicaSets.Items[0].Status.Replicas).Should(Equal(instance.Status.ReplicantNodesStatus.Replicas))
			})

			By("Checking the EMQX Custom Resource's Pod and EndpointSlice", func() {
				checkPodAndEndpointsAndEndpointSlices(instance, 3)
			})

			By("Checking the EMQX Custom Resource's Service", func() {
				checkService(instance)
			})
		})
	})

	Context("Blue Green Update", Label("blue"), func() {
		instance := &appsv2alpha2.EMQX{}
		currentImage := "emqx/emqx:5.0.23"
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
				list := &appsv1.StatefulSetList{}
				Eventually(func() []appsv1.StatefulSet {
					_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(instance), instance)
					_ = k8sClient.List(context.TODO(), list,
						client.InNamespace(instance.Namespace),
						client.MatchingLabels(appsv2alpha2.CloneAndAddLabel(
							instance.Spec.CoreTemplate.Labels,
							appsv1.DefaultDeploymentUniqueLabelKey,
							instance.Status.CoreNodesStatus.CurrentVersion,
						)),
					)
					return list.Items
				}, timeout, interval).Should(ConsistOf(
					WithTransform(func(s appsv1.StatefulSet) string { return s.Spec.Template.Spec.Containers[0].Image }, Equal(currentImage)),
				))
			})

			By("Checking replicaSet list", func() {
				var dList []*appsv1.ReplicaSet
				Eventually(func() int {
					replicaSets := &appsv1.ReplicaSetList{}
					_ = k8sClient.List(context.TODO(), replicaSets,
						client.InNamespace(instance.Namespace),
						client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
					)
					dList = []*appsv1.ReplicaSet{}
					for _, d := range replicaSets.Items {
						dList = append(dList, d.DeepCopy())
					}
					return len(dList)
				}, timeout, interval).Should(Equal(2))

				sort.Sort(appscontrollersv2alpha2.ReplicaSetsByCreationTimestamp(dList))

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
				}, timeout, interval).Should(Equal(instance.Status.ReplicantNodesStatus.Replicas))
			})

			By("Checking endpointScales list", func() {
				checkPodAndEndpointsAndEndpointSlices(instance, 2)
			})

			By("Checking the EMQX Custom Resource's Service", func() {
				checkService(instance)
			})

			By("Checking the EMQX Custom Resource status", func() {
				Eventually(func() string {
					_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(instance), instance)
					return instance.Status.CurrentImage
				}).Should(Equal(currentImage))
				checkRunning(instance.DeepCopy())
			})
		})
	})
})

func createResource(instance *appsv2alpha2.EMQX) {
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

func deleteResource(instance *appsv2alpha2.EMQX) {
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

func checkPodAndEndpointsAndEndpointSlices(instance *appsv2alpha2.EMQX, count int) {
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
						HaveField("Type", appsv2alpha2.PodOnServing),
						HaveField("Type", corev1.PodReady),
					)))),
			),
		),
	)

	endPointsMatcher := []gomegaTypes.GomegaMatcher{}
	endpointSliceMatcher := []gomegaTypes.GomegaMatcher{}
	for _, p := range podList.Items {
		pod := p.DeepCopy()
		ep := And(
			HaveField("IP", pod.Status.PodIP),
			HaveField("NodeName", HaveValue(Equal(pod.Spec.NodeName))),
			HaveField("TargetRef", And(
				HaveField("Kind", "Pod"),
				HaveField("UID", pod.GetUID()),
				HaveField("Name", pod.GetName()),
				HaveField("Namespace", pod.GetNamespace()),
			)),
		)
		endPointsMatcher = append(endPointsMatcher, ep)

		eps := And(
			HaveField("Addresses", ConsistOf([]string{pod.Status.PodIP})),
			HaveField("NodeName", HaveValue(Equal(pod.Spec.NodeName))),
			HaveField("Conditions", And(
				HaveField("Ready", HaveValue(BeTrue())),
				HaveField("Serving", BeNil()),
				HaveField("Terminating", BeNil()),
			)),
			HaveField("TargetRef", And(
				HaveField("Kind", "Pod"),
				HaveField("UID", pod.GetUID()),
				HaveField("Name", pod.GetName()),
				HaveField("Namespace", pod.GetNamespace()),
			)),
		)
		endpointSliceMatcher = append(endpointSliceMatcher, eps)
	}

	Eventually(func() *corev1.Endpoints {
		ep := &corev1.Endpoints{}
		_ = k8sClient.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.ListenersServiceTemplate.Name, Namespace: instance.Namespace}, ep)
		return ep
	}, timeout, interval).Should(HaveField("Subsets",
		And(
			HaveLen(1),
			ContainElement(
				HaveField("Addresses", ConsistOf(endPointsMatcher)),
			),
			ContainElement(
				HaveField("Ports", ConsistOf([]corev1.EndpointPort{
					{
						Name:     "tcp-default",
						Port:     1883,
						Protocol: corev1.ProtocolTCP,
					},
					{
						Name:     "lwm2m-udp-default",
						Port:     5783,
						Protocol: corev1.ProtocolUDP,
					},
				})),
			),
		),
	))

	Eventually(func() []discoveryv1.EndpointSlice {
		list := &discoveryv1.EndpointSliceList{}
		_ = k8sClient.List(
			context.TODO(), list,
			client.InNamespace(instance.Namespace),
			client.MatchingLabels(
				map[string]string{
					"kubernetes.io/service-name": instance.Spec.ListenersServiceTemplate.Name,
				},
			),
		)
		return list.Items
	}, timeout, interval).Should(
		And(
			HaveLen(1),
			ContainElement(
				HaveField("Endpoints", ConsistOf(endpointSliceMatcher)),
			),
			ContainElement(
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
		),
	)
}

func checkService(instance *appsv2alpha2.EMQX) {
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

func checkRunning(instance *appsv2alpha2.EMQX) {
	Eventually(func() appsv2alpha2.EMQXStatus {
		_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(instance), instance)
		return instance.Status
	}, timeout, interval).Should(
		And(
			HaveField("Conditions", ConsistOf(
				HaveField("Type", appsv2alpha2.Ready),
				HaveField("Type", appsv2alpha2.CodeNodesReady),
				HaveField("Type", appsv2alpha2.CoreNodesProgressing),
				HaveField("Type", appsv2alpha2.Initialized),
			)),
			HaveField("CoreNodesStatus", And(
				HaveField("Nodes", HaveLen(2)),
				HaveField("Replicas", Equal(int32(2))),
				HaveField("ReadyReplicas", Equal(int32(2))),
			)),
			HaveField("ReplicantNodesStatus", And(
				HaveField("Nodes", HaveLen(2)),
				HaveField("Replicas", Equal(int32(2))),
				HaveField("ReadyReplicas", Equal(int32(2))),
			)),
		),
	)
}
