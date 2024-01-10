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

package v2beta1

import (
	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegaTypes "github.com/onsi/gomega/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("E2E Test", Label("base"), Ordered, func() {
	var instance *appsv2beta1.EMQX = new(appsv2beta1.EMQX)
	BeforeEach(func() {
		instance = emqx.DeepCopy()
	})

	Context("replicant template is nil", func() {
		JustBeforeEach(func() {
			instance.Spec.ReplicantTemplate = nil
			instance.Spec.CoreTemplate.Spec.Replicas = pointer.Int32Ptr(2)
		})

		It("should create namespace and EMQX CR", func() {
			Expect(k8sClient.Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: instance.GetNamespace(),
					Labels: map[string]string{
						"test": "e2e",
					},
				},
			})).Should(Succeed())
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())
		})

		It("should create EMQX CR successfully", func() {
			Eventually(func() *appsv2beta1.EMQX {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
				return instance
			}).WithTimeout(timeout).WithPolling(interval).Should(
				And(
					WithTransform(func(instance *appsv2beta1.EMQX) bool {
						return instance.Status.IsConditionTrue(appsv2beta1.Ready)
					}, BeTrue()),
					WithTransform(func(instance *appsv2beta1.EMQX) []appsv2beta1.EMQXNode {
						return instance.Status.CoreNodes
					}, HaveLen(int(*instance.Spec.CoreTemplate.Spec.Replicas))),
					WithTransform(func(instance *appsv2beta1.EMQX) appsv2beta1.EMQXNodesStatus {
						return instance.Status.CoreNodesStatus
					}, And(
						HaveField("Replicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("ReadyReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("CurrentRevision", Not(Equal(""))),
						HaveField("CurrentReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("UpdateRevision", Not(Equal(""))),
						HaveField("UpdateReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
					)),
					WithTransform(func(instance *appsv2beta1.EMQX) []appsv2beta1.EMQXNode {
						return instance.Status.ReplicantNodes
					}, BeNil()),
					WithTransform(func(instance *appsv2beta1.EMQX) appsv2beta1.EMQXNodesStatus {
						return instance.Status.ReplicantNodesStatus
					}, And(
						HaveField("Replicas", Equal(int32(0))),
						HaveField("ReadyReplicas", Equal(int32(0))),
						HaveField("CurrentRevision", Equal("")),
						HaveField("CurrentReplicas", Equal(int32(0))),
						HaveField("UpdateRevision", Equal("")),
						HaveField("UpdateReplicas", Equal(int32(0))),
					)),
				),
			)

			checkServices(instance)
			checkPods(instance)
			checkEndpoints(instance, appsv2beta1.CloneAndAddLabel(
				appsv2beta1.DefaultCoreLabels(instance),
				appsv2beta1.LabelsPodTemplateHashKey,
				instance.Status.CoreNodesStatus.CurrentRevision,
			))
		})

		It("scale up EMQX core nodes", func() {
			var storage *appsv2beta1.EMQX
			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance); err != nil {
					return err
				}
				storage = instance.DeepCopy()
				instance.Spec.CoreTemplate.Spec.Replicas = pointer.Int32Ptr(3)
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())

			Eventually(func() *appsv2beta1.EMQX {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
				return instance
			}).WithTimeout(timeout).WithPolling(interval).Should(
				And(
					WithTransform(func(instance *appsv2beta1.EMQX) bool {
						return instance.Status.IsConditionTrue(appsv2beta1.Ready)
					}, BeTrue()),
					WithTransform(func(instance *appsv2beta1.EMQX) []appsv2beta1.EMQXNode {
						return instance.Status.CoreNodes
					}, HaveLen(int(*instance.Spec.CoreTemplate.Spec.Replicas))),
					WithTransform(func(instance *appsv2beta1.EMQX) appsv2beta1.EMQXNodesStatus {
						return instance.Status.CoreNodesStatus
					}, And(
						HaveField("Replicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("ReadyReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("CurrentRevision", Equal(storage.Status.CoreNodesStatus.CurrentRevision)),
						HaveField("CurrentReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("UpdateRevision", Not(Equal(""))),
						HaveField("UpdateReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
					)),
					WithTransform(func(instance *appsv2beta1.EMQX) []appsv2beta1.EMQXNode {
						return instance.Status.ReplicantNodes
					}, BeNil()),
					WithTransform(func(instance *appsv2beta1.EMQX) appsv2beta1.EMQXNodesStatus {
						return instance.Status.ReplicantNodesStatus
					}, And(
						HaveField("Replicas", Equal(int32(0))),
						HaveField("ReadyReplicas", Equal(int32(0))),
						HaveField("CurrentRevision", Equal("")),
						HaveField("CurrentReplicas", Equal(int32(0))),
						HaveField("UpdateRevision", Equal("")),
						HaveField("UpdateReplicas", Equal(int32(0))),
					)),
				),
			)

			checkServices(instance)
			checkPods(instance)
			checkEndpoints(instance, appsv2beta1.CloneAndAddLabel(
				appsv2beta1.DefaultCoreLabels(instance),
				appsv2beta1.LabelsPodTemplateHashKey,
				instance.Status.CoreNodesStatus.CurrentRevision,
			))
		})

		It("scale down EMQX core nodes", func() {
			var storage *appsv2beta1.EMQX
			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance); err != nil {
					return err
				}
				storage = instance.DeepCopy()
				instance.Spec.CoreTemplate.Spec.Replicas = pointer.Int32Ptr(1)
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())

			Eventually(func() *appsv2beta1.EMQX {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
				return instance
			}).WithTimeout(timeout).WithPolling(interval).Should(
				And(
					WithTransform(func(instance *appsv2beta1.EMQX) bool {
						return instance.Status.IsConditionTrue(appsv2beta1.Ready)
					}, BeTrue()),
					WithTransform(func(instance *appsv2beta1.EMQX) []appsv2beta1.EMQXNode {
						return instance.Status.CoreNodes
					}, HaveLen(int(*instance.Spec.CoreTemplate.Spec.Replicas))),
					WithTransform(func(instance *appsv2beta1.EMQX) appsv2beta1.EMQXNodesStatus {
						return instance.Status.CoreNodesStatus
					}, And(
						HaveField("Replicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("ReadyReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("CurrentRevision", Equal(storage.Status.CoreNodesStatus.CurrentRevision)),
						HaveField("CurrentReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("UpdateRevision", Equal(storage.Status.CoreNodesStatus.CurrentRevision)),
						HaveField("UpdateReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
					)),
					WithTransform(func(instance *appsv2beta1.EMQX) []appsv2beta1.EMQXNode {
						return instance.Status.ReplicantNodes
					}, BeNil()),
					WithTransform(func(instance *appsv2beta1.EMQX) appsv2beta1.EMQXNodesStatus {
						return instance.Status.ReplicantNodesStatus
					}, And(
						HaveField("Replicas", Equal(int32(0))),
						HaveField("ReadyReplicas", Equal(int32(0))),
						HaveField("CurrentRevision", Equal("")),
						HaveField("CurrentReplicas", Equal(int32(0))),
						HaveField("UpdateRevision", Equal("")),
						HaveField("UpdateReplicas", Equal(int32(0))),
					)),
				),
			)

			checkServices(instance)
			checkPods(instance)
			checkEndpoints(instance, appsv2beta1.CloneAndAddLabel(
				appsv2beta1.DefaultCoreLabels(instance),
				appsv2beta1.LabelsPodTemplateHashKey,
				instance.Status.CoreNodesStatus.CurrentRevision,
			))
		})

		It("change EMQX image", func() {
			var storage *appsv2beta1.EMQX
			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance); err != nil {
					return err
				}
				storage = instance.DeepCopy()
				instance.Spec.Image = "emqx:5"
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())

			Eventually(func() *appsv2beta1.EMQX {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
				return instance
			}).WithTimeout(timeout).WithPolling(interval).Should(
				And(
					WithTransform(func(instance *appsv2beta1.EMQX) bool {
						return instance.Status.IsConditionTrue(appsv2beta1.Ready)
					}, BeTrue()),
					WithTransform(func(instance *appsv2beta1.EMQX) appsv2beta1.EMQXNodesStatus {
						return instance.Status.CoreNodesStatus
					}, And(
						HaveField("CurrentRevision", Not(Equal(storage.Status.CoreNodesStatus.CurrentRevision))),
						HaveField("UpdateRevision", Not(Equal(storage.Status.CoreNodesStatus.CurrentRevision))),
					)),
					WithTransform(func(instance *appsv2beta1.EMQX) appsv2beta1.EMQXNodesStatus {
						return instance.Status.ReplicantNodesStatus
					}, And(
						HaveField("Replicas", Equal(int32(0))),
						HaveField("ReadyReplicas", Equal(int32(0))),
						HaveField("CurrentRevision", Equal("")),
						HaveField("CurrentReplicas", Equal(int32(0))),
						HaveField("UpdateRevision", Equal("")),
						HaveField("UpdateReplicas", Equal(int32(0))),
					)),
				),
			)

			checkServices(instance)
			checkPods(instance)
			checkEndpoints(instance, appsv2beta1.CloneAndAddLabel(
				appsv2beta1.DefaultCoreLabels(instance),
				appsv2beta1.LabelsPodTemplateHashKey,
				instance.Status.CoreNodesStatus.CurrentRevision,
			))
		})

		It("old sts should scale down to 0", func() {
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)).Should(Succeed())
			Eventually(func() []appsv1.StatefulSet {
				list := &appsv1.StatefulSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(appsv2beta1.DefaultLabels(instance)),
				)
				for i, sts := range list.Items {
					if podTemplateHash, ok := sts.Labels[appsv2beta1.LabelsPodTemplateHashKey]; ok {
						if podTemplateHash == instance.Status.CoreNodesStatus.CurrentRevision {
							list.Items = append(list.Items[:i], list.Items[i+1:]...)
						}
					}
				}
				return list.Items
			}).WithTimeout(timeout).WithPolling(interval).Should(HaveEach(
				WithTransform(func(sts appsv1.StatefulSet) int32 {
					return *sts.Spec.Replicas
				}, Equal(int32(0))),
			))
		})

		It("change EMQX listener port", func() {
			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance); err != nil {
					return err
				}
				instance.Spec.Config.Data = `listeners.tcp.default.bind = "11883"`
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())

			Eventually(func() []corev1.ServicePort {
				svc := &corev1.Service{}
				_ = k8sClient.Get(ctx, instance.ListenersServiceNamespacedName(), svc)
				return svc.Spec.Ports
			}).WithTimeout(timeout).WithPolling(interval).Should(ContainElement(
				WithTransform(func(port corev1.ServicePort) int32 {
					return port.Port
				}, Equal(int32(11883))),
			))

			Eventually(func() []corev1.EndpointPort {
				ep := &corev1.Endpoints{}
				_ = k8sClient.Get(ctx, instance.ListenersServiceNamespacedName(), ep)
				return ep.Subsets[0].Ports
			}).WithTimeout(timeout).WithPolling(interval).Should(ContainElement(
				WithTransform(func(port corev1.EndpointPort) int32 {
					return port.Port
				}, Equal(int32(11883))),
			))

			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance); err != nil {
					return err
				}
				instance.Spec.Config.Data = `listeners.tcp.default.bind = "1883"`
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())
		})
	})

	Context("replicant template is not nil", func() {
		JustBeforeEach(func() {
			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance); err != nil {
					return err
				}
				instance.Spec = *emqx.Spec.DeepCopy()
				instance.Spec.ReplicantTemplate = &appsv2beta1.EMQXReplicantTemplate{
					Spec: appsv2beta1.EMQXReplicantTemplateSpec{
						Replicas: pointer.Int32Ptr(2),
					},
				}
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())
		})

		It("should update EMQX CR successfully", func() {
			Eventually(func() *appsv2beta1.EMQX {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
				return instance
			}).WithTimeout(timeout).WithPolling(interval).Should(
				And(
					WithTransform(func(instance *appsv2beta1.EMQX) bool {
						return instance.Status.IsConditionTrue(appsv2beta1.Ready)
					}, BeTrue()),
					WithTransform(func(instance *appsv2beta1.EMQX) appsv2beta1.EMQXNodesStatus {
						return instance.Status.CoreNodesStatus
					}, And(
						HaveField("Replicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("ReadyReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("CurrentRevision", Not(Equal(""))),
						HaveField("CurrentReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("UpdateRevision", Not(Equal(""))),
						HaveField("UpdateReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
					)),
					WithTransform(func(instance *appsv2beta1.EMQX) appsv2beta1.EMQXNodesStatus {
						return instance.Status.ReplicantNodesStatus
					}, And(
						HaveField("Replicas", Equal(int32(*instance.Spec.ReplicantTemplate.Spec.Replicas))),
						HaveField("ReadyReplicas", Equal(int32(*instance.Spec.ReplicantTemplate.Spec.Replicas))),
						HaveField("CurrentRevision", Not(Equal(""))),
						HaveField("CurrentReplicas", Equal(int32(*instance.Spec.ReplicantTemplate.Spec.Replicas))),
						HaveField("UpdateRevision", Not(Equal(""))),
						HaveField("UpdateReplicas", Equal(int32(*instance.Spec.ReplicantTemplate.Spec.Replicas))),
					)),
				),
			)

			checkServices(instance)
			checkPods(instance)
			checkEndpoints(instance, appsv2beta1.CloneAndAddLabel(
				appsv2beta1.DefaultReplicantLabels(instance),
				appsv2beta1.LabelsPodTemplateHashKey,
				instance.Status.ReplicantNodesStatus.CurrentRevision,
			))
		})

		It("scale up EMQX replicant nodes", func() {
			var storage *appsv2beta1.EMQX
			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance); err != nil {
					return err
				}
				storage = instance.DeepCopy()
				instance.Spec.ReplicantTemplate.Spec.Replicas = pointer.Int32Ptr(3)
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())

			Eventually(func() *appsv2beta1.EMQX {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
				return instance
			}).WithTimeout(timeout).WithPolling(interval).Should(
				And(
					WithTransform(func(instance *appsv2beta1.EMQX) bool {
						return instance.Status.IsConditionTrue(appsv2beta1.Ready)
					}, BeTrue()),
					WithTransform(func(instance *appsv2beta1.EMQX) []appsv2beta1.EMQXNode {
						return instance.Status.CoreNodes
					}, HaveLen(int(*instance.Spec.CoreTemplate.Spec.Replicas))),
					WithTransform(func(instance *appsv2beta1.EMQX) appsv2beta1.EMQXNodesStatus {
						return instance.Status.CoreNodesStatus
					}, And(
						HaveField("Replicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("ReadyReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("CurrentRevision", Equal(storage.Status.CoreNodesStatus.CurrentRevision)),
						HaveField("CurrentReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("UpdateRevision", Equal(storage.Status.CoreNodesStatus.CurrentRevision)),
						HaveField("UpdateReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
					)),
					WithTransform(func(instance *appsv2beta1.EMQX) []appsv2beta1.EMQXNode {
						return instance.Status.ReplicantNodes
					}, HaveLen(int(*instance.Spec.ReplicantTemplate.Spec.Replicas))),
					WithTransform(func(instance *appsv2beta1.EMQX) appsv2beta1.EMQXNodesStatus {
						return instance.Status.ReplicantNodesStatus
					}, And(
						HaveField("Replicas", Equal(int32(*instance.Spec.ReplicantTemplate.Spec.Replicas))),
						HaveField("ReadyReplicas", Equal(int32(*instance.Spec.ReplicantTemplate.Spec.Replicas))),
						HaveField("CurrentRevision", Equal(storage.Status.ReplicantNodesStatus.CurrentRevision)),
						HaveField("CurrentReplicas", Equal(int32(*instance.Spec.ReplicantTemplate.Spec.Replicas))),
						HaveField("UpdateRevision", Equal(storage.Status.ReplicantNodesStatus.CurrentRevision)),
						HaveField("UpdateReplicas", Equal(int32(*instance.Spec.ReplicantTemplate.Spec.Replicas))),
					)),
				),
			)

			checkServices(instance)
			checkPods(instance)
			checkEndpoints(instance, appsv2beta1.CloneAndAddLabel(
				appsv2beta1.DefaultReplicantLabels(instance),
				appsv2beta1.LabelsPodTemplateHashKey,
				instance.Status.ReplicantNodesStatus.CurrentRevision,
			))
		})

		It("scale down EMQX replicant nodes to 0", func() {
			var storage *appsv2beta1.EMQX
			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance); err != nil {
					return err
				}
				storage = instance.DeepCopy()
				instance.Spec.ReplicantTemplate.Spec.Replicas = pointer.Int32Ptr(0)
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())

			Eventually(func() *appsv2beta1.EMQX {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
				return instance
			}).WithTimeout(timeout).WithPolling(interval).Should(
				And(
					WithTransform(func(instance *appsv2beta1.EMQX) bool {
						return instance.Status.IsConditionTrue(appsv2beta1.Ready)
					}, BeTrue()),
					WithTransform(func(instance *appsv2beta1.EMQX) []appsv2beta1.EMQXNode {
						return instance.Status.CoreNodes
					}, HaveLen(int(*instance.Spec.CoreTemplate.Spec.Replicas))),
					WithTransform(func(instance *appsv2beta1.EMQX) appsv2beta1.EMQXNodesStatus {
						return instance.Status.CoreNodesStatus
					}, And(
						HaveField("Replicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("ReadyReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("CurrentRevision", Equal(storage.Status.CoreNodesStatus.CurrentRevision)),
						HaveField("CurrentReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("UpdateRevision", Equal(storage.Status.CoreNodesStatus.CurrentRevision)),
						HaveField("UpdateReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
					)),
					WithTransform(func(instance *appsv2beta1.EMQX) []appsv2beta1.EMQXNode {
						return instance.Status.ReplicantNodes
					}, HaveLen(int(*instance.Spec.ReplicantTemplate.Spec.Replicas))),
					WithTransform(func(instance *appsv2beta1.EMQX) appsv2beta1.EMQXNodesStatus {
						return instance.Status.ReplicantNodesStatus
					}, And(
						HaveField("Replicas", Equal(int32(*instance.Spec.ReplicantTemplate.Spec.Replicas))),
						HaveField("ReadyReplicas", Equal(int32(*instance.Spec.ReplicantTemplate.Spec.Replicas))),
						HaveField("CurrentRevision", Equal(storage.Status.ReplicantNodesStatus.CurrentRevision)),
						HaveField("CurrentReplicas", Equal(int32(*instance.Spec.ReplicantTemplate.Spec.Replicas))),
						HaveField("UpdateRevision", Equal(storage.Status.ReplicantNodesStatus.CurrentRevision)),
						HaveField("UpdateReplicas", Equal(int32(*instance.Spec.ReplicantTemplate.Spec.Replicas))),
					)),
				),
			)

			checkServices(instance)
			checkPods(instance)
			checkEndpoints(instance, appsv2beta1.CloneAndAddLabel(
				appsv2beta1.DefaultCoreLabels(instance),
				appsv2beta1.LabelsPodTemplateHashKey,
				instance.Status.CoreNodesStatus.CurrentRevision,
			))
		})

		It("change EMQX image", func() {
			var storage *appsv2beta1.EMQX
			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance); err != nil {
					return err
				}
				storage = instance.DeepCopy()
				instance.Spec.Image = "emqx/emqx:latest"
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())

			Eventually(func() *appsv2beta1.EMQX {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
				return instance
			}).WithTimeout(timeout).WithPolling(interval).Should(
				And(
					WithTransform(func(instance *appsv2beta1.EMQX) bool {
						return instance.Status.IsConditionTrue(appsv2beta1.Ready)
					}, BeTrue()),
					WithTransform(func(instance *appsv2beta1.EMQX) appsv2beta1.EMQXNodesStatus {
						return instance.Status.CoreNodesStatus
					}, And(
						HaveField("CurrentRevision", Not(Equal(storage.Status.CoreNodesStatus.CurrentRevision))),
						HaveField("UpdateRevision", Not(Equal(storage.Status.CoreNodesStatus.CurrentRevision))),
					)),
					WithTransform(func(instance *appsv2beta1.EMQX) appsv2beta1.EMQXNodesStatus {
						return instance.Status.ReplicantNodesStatus
					}, And(
						HaveField("CurrentRevision", Not(Equal(storage.Status.ReplicantNodesStatus.CurrentRevision))),
						HaveField("UpdateRevision", Not(Equal(storage.Status.ReplicantNodesStatus.CurrentRevision))),
					)),
				),
			)

			checkServices(instance)
			checkPods(instance)
			checkEndpoints(instance, appsv2beta1.CloneAndAddLabel(
				appsv2beta1.DefaultReplicantLabels(instance),
				appsv2beta1.LabelsPodTemplateHashKey,
				instance.Status.ReplicantNodesStatus.CurrentRevision,
			))
		})

		It("old rs should scale down to 0", func() {
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)).Should(Succeed())
			Eventually(func() []appsv1.ReplicaSet {
				list := &appsv1.ReplicaSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(appsv2beta1.DefaultLabels(instance)),
				)
				for i, sts := range list.Items {
					if podTemplateHash, ok := sts.Labels[appsv2beta1.LabelsPodTemplateHashKey]; ok {
						if podTemplateHash == instance.Status.ReplicantNodesStatus.CurrentRevision {
							list.Items = append(list.Items[:i], list.Items[i+1:]...)
						}
					}
				}
				return list.Items
			}).WithTimeout(timeout).WithPolling(interval).Should(HaveEach(
				WithTransform(func(rs appsv1.ReplicaSet) int32 {
					return *rs.Spec.Replicas
				}, Equal(int32(0))),
			))
		})

		It("old sts should scale down to 0", func() {
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)).Should(Succeed())
			Eventually(func() []appsv1.StatefulSet {
				list := &appsv1.StatefulSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(appsv2beta1.DefaultLabels(instance)),
				)
				for i, sts := range list.Items {
					if podTemplateHash, ok := sts.Labels[appsv2beta1.LabelsPodTemplateHashKey]; ok {
						if podTemplateHash == instance.Status.CoreNodesStatus.CurrentRevision {
							list.Items = append(list.Items[:i], list.Items[i+1:]...)
						}
					}
				}
				return list.Items
			}).WithTimeout(timeout).WithPolling(interval).Should(HaveEach(
				WithTransform(func(sts appsv1.StatefulSet) int32 {
					return *sts.Spec.Replicas
				}, Equal(int32(0))),
			))
		})

		It("change EMQX listener port", func() {
			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance); err != nil {
					return err
				}
				instance.Spec.Config.Data = `listeners.tcp.default.bind = "11883"`
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())

			Eventually(func() []corev1.ServicePort {
				svc := &corev1.Service{}
				_ = k8sClient.Get(ctx, instance.ListenersServiceNamespacedName(), svc)
				return svc.Spec.Ports
			}).WithTimeout(timeout).WithPolling(interval).Should(ContainElement(
				WithTransform(func(port corev1.ServicePort) int32 {
					return port.Port
				}, Equal(int32(11883))),
			))

			Eventually(func() []corev1.EndpointPort {
				ep := &corev1.Endpoints{}
				_ = k8sClient.Get(ctx, instance.ListenersServiceNamespacedName(), ep)
				return ep.Subsets[0].Ports
			}).WithTimeout(timeout).WithPolling(interval).Should(ContainElement(
				WithTransform(func(port corev1.EndpointPort) int32 {
					return port.Port
				}, Equal(int32(11883))),
			))

			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance); err != nil {
					return err
				}
				instance.Spec.Config.Data = `listeners.tcp.default.bind = "1883"`
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())
		})
	})

	It("should delete namespace", func() {
		Expect(k8sClient.Delete(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: instance.Namespace,
			},
		})).Should(Succeed())
	})
})

func checkServices(instance *appsv2beta1.EMQX) {
	Eventually(func() []corev1.ServicePort {
		svc := &corev1.Service{}
		_ = k8sClient.Get(ctx, instance.ListenersServiceNamespacedName(), svc)
		return svc.Spec.Ports
	}).WithTimeout(timeout).WithPolling(interval).Should(
		ConsistOf([]corev1.ServicePort{
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
			{
				Name:       "lwm2m-udp-default",
				Port:       5783,
				Protocol:   corev1.ProtocolUDP,
				TargetPort: intstr.FromInt(5783),
			},
		}),
	)
}

func checkPods(instance *appsv2beta1.EMQX) {
	podList := &corev1.PodList{}
	Eventually(func() []corev1.Pod {
		_ = k8sClient.List(ctx, podList,
			client.InNamespace(instance.Namespace),
			client.MatchingLabels(appsv2beta1.DefaultLabels(instance)),
		)
		return podList.Items
	}).WithTimeout(timeout).WithPolling(interval).Should(
		And(
			HaveEach(
				HaveField("Status", And(
					HaveField("Phase", corev1.PodRunning),
					HaveField("Conditions", ContainElements(
						HaveField("Type", appsv2beta1.PodOnServing),
						HaveField("Type", corev1.PodReady),
					)))),
			),
		),
	)
}

func checkEndpoints(instance *appsv2beta1.EMQX, labels map[string]string) {
	podList := &corev1.PodList{}
	_ = k8sClient.List(ctx, podList,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(labels),
	)

	endPointsMatcher := []gomegaTypes.GomegaMatcher{}
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
	}

	Eventually(func() *corev1.Endpoints {
		ep := &corev1.Endpoints{}
		_ = k8sClient.Get(ctx, instance.ListenersServiceNamespacedName(), ep)
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
						Name:     "ssl-default",
						Port:     8883,
						Protocol: corev1.ProtocolTCP,
					},
					{
						Name:     "ws-default",
						Port:     8083,
						Protocol: corev1.ProtocolTCP,
					},
					{
						Name:     "wss-default",
						Port:     8084,
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
}
