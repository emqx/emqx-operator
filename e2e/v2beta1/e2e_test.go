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
	"slices"

	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	. "github.com/emqx/emqx-operator/internal/test"
	dedent "github.com/lithammer/dedent"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegaTypes "github.com/onsi/gomega/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("E2E Test", Label("base"), Ordered, func() {

	var instance *appsv2beta1.EMQX
	var instanceKey client.ObjectKey

	createNamespace := func() error {
		err := k8sClient.Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   emqx.coresOnly.GetNamespace(),
				Labels: map[string]string{"apps.emqx.io/test": "e2e"},
			},
		})
		if err == nil {
			return nil
		}
		if k8sErrors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}

	createInstance := func(spec appsv2beta1.EMQXSpec) error {
		err := k8sClient.Get(ctx, instanceKey, instance)
		if k8sErrors.IsNotFound(err) {
			instance.Spec = *spec.DeepCopy()
			return k8sClient.Create(ctx, instance)
		}
		instance.Spec = *spec.DeepCopy()
		return k8sClient.Update(ctx, instance)
	}

	refreshInstance := func() *appsv2beta1.EMQX {
		err := k8sClient.Get(ctx, instanceKey, instance)
		if err != nil {
			return nil
		}
		return instance
	}

	BeforeAll(func() {
		instance = emqx.coresOnly.DeepCopy()
		instanceKey = client.ObjectKeyFromObject(instance)
	})

	Context("cluster with core nodes only", func() {

		It("should create namespace successfully", func() {
			Expect(createNamespace()).Should(Succeed())
		})

		It("should create EMQX CR successfully", func() {
			Expect(createInstance(emqx.coresOnly.Spec)).Should(Succeed())
			Eventually(refreshInstance).WithTimeout(timeout).WithPolling(interval).Should(
				And(
					HaveCondition(appsv2beta1.Ready, HaveField("Status", Equal(metav1.ConditionTrue))),
					HaveField("Status.CoreNodes", HaveLen(int(*instance.Spec.CoreTemplate.Spec.Replicas))),
					HaveField("Status.CoreNodesStatus", And(
						HaveField("Replicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("ReadyReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("CurrentRevision", Not(Equal(""))),
						HaveField("CurrentReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("UpdateRevision", Not(Equal(""))),
						HaveField("UpdateReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
					)),
					HaveField("Status.ReplicantNodes", BeNil()),
					HaveField("Status.ReplicantNodesStatus", And(
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
			Expect(k8sClient.Get(ctx, instanceKey, instance)).Should(Succeed())
			storage := instance.DeepCopy()

			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, instanceKey, instance); err != nil {
					return err
				}
				instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(3))
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())

			Eventually(refreshInstance).WithTimeout(timeout).WithPolling(interval).Should(
				And(
					HaveCondition(appsv2beta1.Ready, HaveField("Status", Equal(metav1.ConditionTrue))),
					HaveField("Status.CoreNodes", HaveLen(int(*instance.Spec.CoreTemplate.Spec.Replicas))),
					HaveField("Status.CoreNodesStatus", And(
						HaveField("Replicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("ReadyReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("CurrentRevision", Equal(storage.Status.CoreNodesStatus.CurrentRevision)),
						HaveField("CurrentReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("UpdateRevision", Not(Equal(""))),
						HaveField("UpdateReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
					)),
					HaveField("Status.ReplicantNodes", BeNil()),
					HaveField("Status.ReplicantNodesStatus", And(
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
			Expect(k8sClient.Get(ctx, instanceKey, instance)).Should(Succeed())
			storage := instance.DeepCopy()

			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, instanceKey, instance); err != nil {
					return err
				}
				instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(1))
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())

			Eventually(refreshInstance).WithTimeout(timeout).WithPolling(interval).Should(
				And(
					HaveCondition(appsv2beta1.Ready, HaveField("Status", Equal(metav1.ConditionTrue))),
					HaveField("Status.CoreNodes", HaveLen(int(*instance.Spec.CoreTemplate.Spec.Replicas))),
					HaveField("Status.CoreNodesStatus", And(
						HaveField("Replicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("ReadyReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("CurrentRevision", Equal(storage.Status.CoreNodesStatus.CurrentRevision)),
						HaveField("CurrentReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("UpdateRevision", Equal(storage.Status.CoreNodesStatus.CurrentRevision)),
						HaveField("UpdateReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
					)),
					HaveField("Status.ReplicantNodes", BeNil()),
					HaveField("Status.ReplicantNodesStatus", And(
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
			Expect(k8sClient.Get(ctx, instanceKey, instance)).Should(Succeed())
			storage := instance.DeepCopy()

			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, instanceKey, instance); err != nil {
					return err
				}
				instance.Spec.Image = "emqx:5"
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())

			Eventually(refreshInstance).WithTimeout(timeout).WithPolling(interval).Should(
				And(
					HaveCondition(appsv2beta1.Ready, HaveField("Status", Equal(metav1.ConditionTrue))),
					HaveField("Status.CoreNodesStatus", And(
						HaveField("Replicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("ReadyReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("CurrentRevision", Not(Equal(storage.Status.CoreNodesStatus.CurrentRevision))),
						HaveField("CurrentReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("UpdateRevision", Not(Equal(storage.Status.CoreNodesStatus.CurrentRevision))),
						HaveField("UpdateReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
					)),
					HaveField("Status.ReplicantNodes", BeNil()),
					HaveField("Status.ReplicantNodesStatus", And(
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
			Expect(k8sClient.Get(ctx, instanceKey, instance)).Should(Succeed())
			Eventually(func() []appsv1.StatefulSet {
				list := &appsv1.StatefulSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(appsv2beta1.DefaultLabels(instance)),
				)
				return slices.DeleteFunc(
					list.Items,
					func(sts appsv1.StatefulSet) bool {
						return sts.Labels[appsv2beta1.LabelsPodTemplateHashKey] == instance.Status.CoreNodesStatus.CurrentRevision
					},
				)
			}).WithTimeout(timeout).WithPolling(interval).Should(HaveEach(
				And(
					HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(0))),
					HaveField("Status.Replicas", HaveValue(BeEquivalentTo(0))),
				),
			))
		})

		It("change EMQX listener port", func() {
			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, instanceKey, instance); err != nil {
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
				HaveField("Port", Equal(int32(11883))),
			))

			Eventually(func() []corev1.EndpointPort {
				ep := &corev1.Endpoints{}
				_ = k8sClient.Get(ctx, instance.ListenersServiceNamespacedName(), ep)
				return ep.Subsets[0].Ports
			}).WithTimeout(timeout).WithPolling(interval).Should(ContainElement(
				HaveField("Port", Equal(int32(11883))),
			))

			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, instanceKey, instance); err != nil {
					return err
				}
				instance.Spec.Config.Data = `listeners.tcp.default.bind = "1883"`
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())
		})
	})

	Context("cluster with core + replicant nodes", func() {

		It("should create namespace successfully", func() {
			Expect(createNamespace()).Should(Succeed())
		})

		It("should update EMQX CR successfully", func() {
			Expect(createInstance(emqx.coresReplicants.Spec)).Should(Succeed())
			Eventually(refreshInstance).WithTimeout(timeout).WithPolling(interval).Should(
				And(
					HaveCondition(appsv2beta1.Ready, HaveField("Status", Equal(metav1.ConditionTrue))),
					HaveField("Status.CoreNodesStatus", And(
						HaveField("Replicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("ReadyReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("CurrentRevision", Not(Equal(""))),
						HaveField("CurrentReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("UpdateRevision", Not(Equal(""))),
						HaveField("UpdateReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
					)),
					HaveField("Status.ReplicantNodesStatus", And(
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
			Expect(k8sClient.Get(ctx, instanceKey, instance)).Should(Succeed())
			storage := instance.DeepCopy()

			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, instanceKey, instance); err != nil {
					return err
				}
				instance.Spec.ReplicantTemplate.Spec.Replicas = ptr.To(int32(3))
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())

			Eventually(refreshInstance).WithTimeout(timeout).WithPolling(interval).Should(
				And(
					HaveCondition(appsv2beta1.Ready, HaveField("Status", Equal(metav1.ConditionTrue))),
					HaveField("Status.CoreNodes", HaveLen(int(*instance.Spec.CoreTemplate.Spec.Replicas))),
					HaveField("Status.CoreNodesStatus", And(
						HaveField("Replicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("ReadyReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("CurrentRevision", Equal(storage.Status.CoreNodesStatus.CurrentRevision)),
						HaveField("CurrentReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("UpdateRevision", Equal(storage.Status.CoreNodesStatus.CurrentRevision)),
						HaveField("UpdateReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
					)),
					HaveField("Status.ReplicantNodes", HaveLen(int(*instance.Spec.ReplicantTemplate.Spec.Replicas))),
					HaveField("Status.ReplicantNodesStatus", And(
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
			Expect(k8sClient.Get(ctx, instanceKey, instance)).Should(Succeed())
			storage := instance.DeepCopy()

			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, instanceKey, instance); err != nil {
					return err
				}
				instance.Spec.ReplicantTemplate.Spec.Replicas = ptr.To(int32(0))
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())

			Eventually(refreshInstance).WithTimeout(timeout).WithPolling(interval).Should(
				And(
					HaveCondition(appsv2beta1.Ready, HaveField("Status", Equal(metav1.ConditionTrue))),
					HaveField("Status.CoreNodes", HaveLen(int(*instance.Spec.CoreTemplate.Spec.Replicas))),
					HaveField("Status.CoreNodesStatus", And(
						HaveField("Replicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("ReadyReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("CurrentRevision", Equal(storage.Status.CoreNodesStatus.CurrentRevision)),
						HaveField("CurrentReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("UpdateRevision", Equal(storage.Status.CoreNodesStatus.CurrentRevision)),
						HaveField("UpdateReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
					)),
					HaveField("Status.ReplicantNodes", HaveLen(int(*instance.Spec.ReplicantTemplate.Spec.Replicas))),
					HaveField("Status.ReplicantNodesStatus", And(
						HaveField("Replicas", Equal(int32(0))),
						HaveField("ReadyReplicas", Equal(int32(0))),
						HaveField("CurrentRevision", Equal(storage.Status.ReplicantNodesStatus.CurrentRevision)),
						HaveField("CurrentReplicas", Equal(int32(0))),
						HaveField("UpdateRevision", Equal(storage.Status.ReplicantNodesStatus.CurrentRevision)),
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
			Expect(k8sClient.Get(ctx, instanceKey, instance)).Should(Succeed())
			storage := instance.DeepCopy()

			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, instanceKey, instance); err != nil {
					return err
				}
				instance.Spec.Image = "emqx/emqx-enterprise:latest-elixir" // EMQX Community Edition is not supported core + replicant cluster after 5.7
				instance.Spec.ReplicantTemplate.Spec.Replicas = ptr.To(int32(2))
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())

			Eventually(refreshInstance).WithTimeout(timeout).WithPolling(interval).Should(
				And(
					HaveCondition(appsv2beta1.Ready, HaveField("Status", Equal(metav1.ConditionTrue))),
					HaveField("Status.CoreNodesStatus", And(
						HaveField("Replicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("ReadyReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("CurrentRevision", Not(Equal(storage.Status.CoreNodesStatus.CurrentRevision))),
						HaveField("CurrentReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("UpdateRevision", Not(Equal(storage.Status.CoreNodesStatus.CurrentRevision))),
						HaveField("UpdateReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
					)),
					HaveField("Status.ReplicantNodesStatus", And(
						HaveField("Replicas", Equal(int32(*instance.Spec.ReplicantTemplate.Spec.Replicas))),
						HaveField("ReadyReplicas", Equal(int32(*instance.Spec.ReplicantTemplate.Spec.Replicas))),
						HaveField("CurrentRevision", Not(Equal(storage.Status.ReplicantNodesStatus.CurrentRevision))),
						HaveField("CurrentReplicas", Equal(int32(*instance.Spec.ReplicantTemplate.Spec.Replicas))),
						HaveField("UpdateRevision", Not(Equal(storage.Status.ReplicantNodesStatus.CurrentRevision))),
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

		It("change EMQX image and scale down EMQX replicant nodes to 0", func() {
			Expect(k8sClient.Get(ctx, instanceKey, instance)).Should(Succeed())
			storage := instance.DeepCopy()

			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, instanceKey, instance); err != nil {
					return err
				}
				instance.Spec.Image = "emqx:5"
				instance.Spec.ReplicantTemplate.Spec.Replicas = ptr.To(int32(0))
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())

			Eventually(refreshInstance).WithTimeout(timeout).WithPolling(interval).Should(
				And(
					HaveCondition(appsv2beta1.Ready, HaveField("Status", Equal(metav1.ConditionTrue))),
					HaveField("Status.CoreNodesStatus", And(
						HaveField("Replicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("ReadyReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("CurrentRevision", Not(Equal(storage.Status.CoreNodesStatus.CurrentRevision))),
						HaveField("CurrentReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
						HaveField("UpdateRevision", Not(Equal(storage.Status.CoreNodesStatus.CurrentRevision))),
						HaveField("UpdateReplicas", Equal(int32(*instance.Spec.CoreTemplate.Spec.Replicas))),
					)),
					HaveField("Status.ReplicantNodesStatus", And(
						HaveField("Replicas", Equal(int32(0))),
						HaveField("ReadyReplicas", Equal(int32(0))),
						HaveField("CurrentRevision", Not(Equal(storage.Status.ReplicantNodesStatus.CurrentRevision))),
						HaveField("CurrentReplicas", Equal(int32(0))),
						HaveField("UpdateRevision", Not(Equal(storage.Status.ReplicantNodesStatus.CurrentRevision))),
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

		It("old rs should scale down to 0", func() {
			Expect(k8sClient.Get(ctx, instanceKey, instance)).Should(Succeed())
			Eventually(func() []appsv1.ReplicaSet {
				list := &appsv1.ReplicaSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(appsv2beta1.DefaultLabels(instance)),
				)
				return slices.DeleteFunc(
					list.Items,
					func(rs appsv1.ReplicaSet) bool {
						return rs.Labels[appsv2beta1.LabelsPodTemplateHashKey] == instance.Status.ReplicantNodesStatus.CurrentRevision
					},
				)
			}).WithTimeout(timeout).WithPolling(interval).Should(HaveEach(
				And(
					HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(0))),
					HaveField("Status.Replicas", HaveValue(BeEquivalentTo(0))),
				),
			))
		})

		It("old sts should scale down to 0", func() {
			Expect(k8sClient.Get(ctx, instanceKey, instance)).Should(Succeed())
			Eventually(func() []appsv1.StatefulSet {
				list := &appsv1.StatefulSetList{}
				_ = k8sClient.List(ctx, list,
					client.InNamespace(instance.Namespace),
					client.MatchingLabels(appsv2beta1.DefaultLabels(instance)),
				)
				return slices.DeleteFunc(
					list.Items,
					func(sts appsv1.StatefulSet) bool {
						return sts.Labels[appsv2beta1.LabelsPodTemplateHashKey] == instance.Status.CoreNodesStatus.CurrentRevision
					},
				)
			}).WithTimeout(timeout).WithPolling(interval).Should(HaveEach(
				And(
					HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(0))),
					HaveField("Status.Replicas", HaveValue(BeEquivalentTo(0))),
				),
			))
		})

		It("change EMQX listener port", func() {
			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, instanceKey, instance); err != nil {
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
				HaveField("Port", Equal(int32(11883))),
			))

			Eventually(func() []corev1.EndpointPort {
				ep := &corev1.Endpoints{}
				_ = k8sClient.Get(ctx, instance.ListenersServiceNamespacedName(), ep)
				return ep.Subsets[0].Ports
			}).WithTimeout(timeout).WithPolling(interval).Should(ContainElement(
				HaveField("Port", Equal(int32(11883))),
			))

			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, instanceKey, instance); err != nil {
					return err
				}
				instance.Spec.Config.Data = `listeners.tcp.default.bind = "1883"`
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())
		})

	})

	Context("cluster with DS enabled", func() {

		It("should create namespace successfully", func() {
			Expect(createNamespace()).Should(Succeed())
		})

		It("should update EMQX CR successfully", func() {
			spec := *emqx.coresReplicants.Spec.DeepCopy()
			spec.Config.Data += dedent.Dedent(`
			durable_sessions.enable = true
			durable_storage.messages {
			  backend = "builtin_raft"
			  n_shards = 8
			}
			`)
			Expect(createInstance(spec)).Should(Succeed())
			Eventually(refreshInstance).WithTimeout(timeout).WithPolling(interval).Should(
				And(
					HaveCondition(appsv2beta1.Ready, HaveField("Status", Equal(metav1.ConditionTrue))),
					HaveField("Status.CoreNodesStatus", And(
						HaveField("Replicas", BeEquivalentTo(2)),
						HaveField("ReadyReplicas", BeEquivalentTo(2)),
						HaveField("CurrentRevision", Not(Equal(""))),
						HaveField("CurrentReplicas", BeEquivalentTo(2)),
						HaveField("UpdateRevision", Not(Equal(""))),
						HaveField("UpdateReplicas", BeEquivalentTo(2)),
					)),
					HaveField("Status.ReplicantNodesStatus", And(
						HaveField("Replicas", BeEquivalentTo(2)),
						HaveField("ReadyReplicas", BeEquivalentTo(2)),
						HaveField("CurrentRevision", Not(Equal(""))),
						HaveField("CurrentReplicas", BeEquivalentTo(2)),
						HaveField("UpdateRevision", Not(Equal(""))),
						HaveField("UpdateReplicas", BeEquivalentTo(2)),
					)),
					HaveField("Status.DSReplication.DBs",
						ContainElement(
							And(
								HaveField("Name", Equal("messages")),
								HaveField("NumShards", BeEquivalentTo(8)),
								HaveField("NumTransitions", BeEquivalentTo(0)),
								HaveField("MinReplicas", BeEquivalentTo(2)),
								HaveField("MaxReplicas", BeEquivalentTo(2)),
							),
						),
					),
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

		It("scale up EMQX core nodes", func() {
			Expect(k8sClient.Get(ctx, instanceKey, instance)).Should(Succeed())
			storage := instance.DeepCopy()

			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, instanceKey, instance); err != nil {
					return err
				}
				instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(3))
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())

			Eventually(refreshInstance).WithTimeout(timeout).WithPolling(interval).Should(
				And(
					HaveCondition(appsv2beta1.Ready, HaveField("Status", Equal(metav1.ConditionTrue))),
					HaveField("Status.CoreNodes", HaveLen(3)),
					HaveField("Status.CoreNodesStatus", And(
						HaveField("Replicas", BeEquivalentTo(3)),
						HaveField("ReadyReplicas", BeEquivalentTo(3)),
						HaveField("CurrentRevision", Equal(storage.Status.CoreNodesStatus.CurrentRevision)),
						HaveField("CurrentReplicas", BeEquivalentTo(3)),
						HaveField("UpdateReplicas", BeEquivalentTo(3)),
					)),
					HaveField("Status.ReplicantNodes", HaveLen(2)),
					HaveField("Status.ReplicantNodesStatus", And(
						HaveField("Replicas", BeEquivalentTo(2)),
						HaveField("ReadyReplicas", BeEquivalentTo(2)),
						HaveField("CurrentRevision", Equal(storage.Status.ReplicantNodesStatus.CurrentRevision)),
						HaveField("CurrentReplicas", BeEquivalentTo(2)),
						HaveField("UpdateReplicas", BeEquivalentTo(2)),
					)),
					HaveField("Status.DSReplication.DBs",
						ContainElement(
							And(
								HaveField("Name", Equal("messages")),
								HaveField("NumShards", BeEquivalentTo(8)),
								HaveField("NumTransitions", BeEquivalentTo(0)),
								HaveField("MinReplicas", BeEquivalentTo(3)),
								HaveField("MaxReplicas", BeEquivalentTo(3)),
							),
						),
					),
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

		It("scale down EMQX core nodes", func() {
			Expect(k8sClient.Get(ctx, instanceKey, instance)).Should(Succeed())
			storage := instance.DeepCopy()

			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, instanceKey, instance); err != nil {
					return err
				}
				instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(1))
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())

			Eventually(refreshInstance).WithTimeout(timeout).WithPolling(interval).Should(
				And(
					HaveCondition(appsv2beta1.Ready, HaveField("Status", Equal(metav1.ConditionTrue))),
					HaveField("Status.CoreNodes", HaveLen(1)),
					HaveField("Status.CoreNodesStatus", And(
						HaveField("Replicas", BeEquivalentTo(1)),
						HaveField("ReadyReplicas", BeEquivalentTo(1)),
						HaveField("CurrentRevision", Equal(storage.Status.CoreNodesStatus.CurrentRevision)),
						HaveField("CurrentReplicas", BeEquivalentTo(1)),
						HaveField("UpdateReplicas", BeEquivalentTo(1)),
					)),
					HaveField("Status.ReplicantNodes", HaveLen(2)),
					HaveField("Status.ReplicantNodesStatus", And(
						HaveField("Replicas", BeEquivalentTo(2)),
						HaveField("ReadyReplicas", BeEquivalentTo(2)),
						HaveField("CurrentRevision", Equal(storage.Status.ReplicantNodesStatus.CurrentRevision)),
						HaveField("CurrentReplicas", BeEquivalentTo(2)),
						HaveField("UpdateReplicas", BeEquivalentTo(2)),
					)),
					HaveField("Status.DSReplication.DBs",
						ContainElement(
							And(
								HaveField("Name", Equal("messages")),
								HaveField("NumShards", BeEquivalentTo(8)),
								HaveField("MinReplicas", BeEquivalentTo(1)),
								HaveField("MaxReplicas", BeEquivalentTo(1)),
							),
						),
					),
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

		It("should change configuration smoothly", func() {
			Expect(k8sClient.Get(ctx, instanceKey, instance)).Should(Succeed())
			storage := instance.DeepCopy()

			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, instanceKey, instance); err != nil {
					return err
				}
				instance.Spec.CoreTemplate.Spec.Env = []corev1.EnvVar{
					{
						Name:  "EMQX_LOG__CONSOLE__LEVEL",
						Value: "info",
					},
				}
				instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(3))
				instance.Spec.ReplicantTemplate.Spec.Replicas = ptr.To(int32(0))
				instance.Spec.Config.Data += dedent.Dedent(`
				listeners.tcp.default.bind = "11883"
				`)
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())

			Eventually(refreshInstance).WithTimeout(timeout).WithPolling(interval).Should(
				And(
					HaveCondition(appsv2beta1.Ready, HaveField("Status", Equal(metav1.ConditionTrue))),
					HaveField("Status.CoreNodes", HaveLen(3)),
					HaveField("Status.CoreNodesStatus", And(
						HaveField("Replicas", BeEquivalentTo(3)),
						HaveField("ReadyReplicas", BeEquivalentTo(3)),
						HaveField("CurrentRevision", Not(Equal(storage.Status.CoreNodesStatus.CurrentRevision))),
						HaveField("CurrentReplicas", BeEquivalentTo(3)),
						HaveField("UpdateReplicas", BeEquivalentTo(3)),
					)),
					HaveField("Status.ReplicantNodes", BeNil()),
					HaveField("Status.ReplicantNodesStatus", And(
						HaveField("Replicas", BeEquivalentTo(0)),
						HaveField("ReadyReplicas", BeEquivalentTo(0)),
						HaveField("CurrentReplicas", BeEquivalentTo(0)),
						HaveField("UpdateReplicas", BeEquivalentTo(0)),
					)),
					HaveField("Status.DSReplication.DBs",
						ContainElement(
							And(
								HaveField("Name", Equal("messages")),
								HaveField("NumShards", BeEquivalentTo(8)),
								HaveField("MinReplicas", BeEquivalentTo(3)),
								HaveField("MaxReplicas", BeEquivalentTo(3)),
							),
						),
					),
				),
			)

			checkPods(instance)

			Eventually(func() []corev1.ServicePort {
				svc := &corev1.Service{}
				_ = k8sClient.Get(ctx, instance.ListenersServiceNamespacedName(), svc)
				return svc.Spec.Ports
			}).WithTimeout(timeout).WithPolling(interval).Should(
				ConsistOf([]corev1.ServicePort{
					{
						Name:       "tcp-default",
						Port:       11883,
						Protocol:   corev1.ProtocolTCP,
						TargetPort: intstr.FromInt(11883),
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

			Eventually(func() *corev1.Endpoints {
				ep := &corev1.Endpoints{}
				_ = k8sClient.Get(ctx, instance.ListenersServiceNamespacedName(), ep)
				return ep
			}, timeout, interval).Should(HaveField("Subsets",
				And(
					HaveLen(1),
					ContainElement(
						HaveField("Ports", ConsistOf([]corev1.EndpointPort{
							{
								Name:     "tcp-default",
								Port:     11883,
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
		})

		It("disabling durable sessions should succeed", func() {
			Expect(k8sClient.Get(ctx, instanceKey, instance)).Should(Succeed())
			Expect(retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, instanceKey, instance); err != nil {
					return err
				}
				instance.Spec.Image = "emqx:5"
				instance.Spec.Config.Data = ""
				return k8sClient.Update(ctx, instance)
			})).Should(Succeed())

			Eventually(refreshInstance).WithTimeout(timeout).WithPolling(interval).Should(
				HaveField("Status.DSReplication.DBs", BeEmpty()),
			)
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
		HaveEach(
			HaveField("Status", And(
				HaveField("Phase", corev1.PodRunning),
				HaveField("Conditions", ContainElements(
					HaveField("Type", appsv2beta1.PodOnServing),
					HaveField("Type", corev1.PodReady),
				)))),
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
