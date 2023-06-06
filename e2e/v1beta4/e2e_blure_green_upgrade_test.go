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

package v1beta4

import (
	"context"
	"fmt"
	"sort"

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	appscontrollersv1beta4 "github.com/emqx/emqx-operator/controllers/apps/v1beta4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Blue Green Update Test", Label("blue"), func() {
	Describe("Just check enterprise", func() {
		emqx := emqxEnterprise.DeepCopy()
		emqx.Spec.Template.Spec.EmqxContainer.Image.Version = "4.4.15"
		emqx.Spec.EmqxBlueGreenUpdate = &appsv1beta4.EmqxBlueGreenUpdate{
			InitialDelaySeconds: 5,
			EvacuationStrategy: appsv1beta4.EvacuationStrategy{
				WaitTakeover:  int32(0),
				ConnEvictRate: int32(1),
				SessEvictRate: int32(1),
			},
		}

		BeforeEach(func() {
			createEmqx(emqx)
		})

		AfterEach(func() {
			deleteEmqx(emqx)
		})

		It("blue green update", func() {
			var existedStsList *appsv1.StatefulSetList
			existedStsList = &appsv1.StatefulSetList{}
			Eventually(func() []appsv1.StatefulSet {
				_ = k8sClient.List(
					context.TODO(),
					existedStsList,
					client.InNamespace(emqx.GetNamespace()),
					client.MatchingLabels(emqx.GetLabels()),
				)
				return existedStsList.Items
			}, timeout, interval).Should(HaveLen(1))

			sts := existedStsList.Items[0].DeepCopy()
			Eventually(func() string {
				// Wait sts ready
				_ = k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Name:      sts.GetName(),
						Namespace: sts.GetNamespace(),
					},
					sts,
				)
				return sts.Status.CurrentRevision
			}, timeout, interval).ShouldNot(BeEmpty())

			By("check CR status before update")
			Eventually(func() appsv1beta4.EmqxEnterpriseStatus {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				return ee.Status
			}, timeout, interval).Should(And(
				HaveField("Conditions", HaveExactElements(
					And(
						HaveField("Type", Equal(appsv1beta4.ConditionRunning)),
						HaveField("Status", Equal(corev1.ConditionTrue)),
					),
				)),
				HaveField("EmqxBlueGreenUpdateStatus", BeNil()),
				HaveField("ReadyReplicas", Equal(int32(1))),
				HaveField("EmqxNodes", ConsistOf(
					HaveField("Node", Equal(fmt.Sprintf("emqx-ee@%s-0.emqx-ee-headless.%s.svc.cluster.local", sts.Name, emqx.GetNamespace()))),
				)),
				HaveField("CurrentStatefulSetVersion", Equal(sts.Status.CurrentRevision)),
			))

			By("checking the EMQX Custom Resource's EndpointSlice before update", func() {
				checkPodAndEndpointsAndEndpointSlices(emqx, ports, []corev1.ServicePort{}, headlessPort, 1)
			})

			By("update EMQX CR")
			Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), emqx)).Should(Succeed())
			emqx.Spec.Template.Spec.Volumes = append(emqx.Spec.Template.Spec.Volumes, corev1.Volume{
				Name: "test-blue-green-update",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			})
			Expect(k8sClient.Patch(context.Background(), emqx.DeepCopy(), client.MergeFrom(emqx))).Should(Succeed())

			By("wait create new sts")
			existedStsList = &appsv1.StatefulSetList{}
			Eventually(func() []appsv1.StatefulSet {
				_ = k8sClient.List(
					context.TODO(),
					existedStsList,
					client.InNamespace(emqx.GetNamespace()),
					client.MatchingLabels(emqx.GetLabels()),
				)
				return existedStsList.Items
			}, timeout, interval).Should(HaveLen(2))

			allSts := []*appsv1.StatefulSet{}
			for _, es := range existedStsList.Items {
				allSts = append(allSts, es.DeepCopy())
			}
			sort.Sort(appscontrollersv1beta4.StatefulSetsBySizeNewer(allSts))

			newSts := allSts[0].DeepCopy()
			Expect(newSts.UID).ShouldNot(Equal(sts.UID))
			Eventually(func() string {
				// Wait sts ready
				_ = k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Name:      newSts.GetName(),
						Namespace: newSts.GetNamespace(),
					},
					newSts,
				)
				return newSts.Status.CurrentRevision
			}, timeout, interval).ShouldNot(BeEmpty())

			By("check CR status in blue-green updating")
			Eventually(func() appsv1beta4.EmqxEnterpriseStatus {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				return ee.Status
			}, timeout, interval).Should(And(
				HaveField("Conditions", HaveExactElements(
					And(
						HaveField("Type", Equal(appsv1beta4.ConditionBlueGreenUpdating)),
						HaveField("Status", Equal(corev1.ConditionTrue)),
					),
					And(
						HaveField("Type", Equal(appsv1beta4.ConditionRunning)),
						HaveField("Status", Equal(corev1.ConditionTrue)),
					),
				)),
				HaveField("EmqxBlueGreenUpdateStatus",
					And(
						HaveField("StartedAt", Not(BeNil())),
						HaveField("OriginStatefulSet", Equal(sts.Name)),
						HaveField("CurrentStatefulSet", Equal(newSts.Name)),
					)),
				HaveField("ReadyReplicas", Equal(int32(2))),
				HaveField("EmqxNodes", ConsistOf(
					HaveField("Node", Equal(fmt.Sprintf("emqx-ee@%s-0.emqx-ee-headless.%s.svc.cluster.local", sts.Name, emqx.GetNamespace()))),
					HaveField("Node", Equal(fmt.Sprintf("emqx-ee@%s-0.emqx-ee-headless.%s.svc.cluster.local", newSts.Name, emqx.GetNamespace()))),
				)),
			))

			By("check CR status after update")
			Eventually(func() appsv1beta4.EmqxEnterpriseStatus {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				return ee.Status
			}, timeout, interval).Should(And(
				HaveField("Conditions", HaveExactElements(
					And(
						HaveField("Type", Equal(appsv1beta4.ConditionRunning)),
						HaveField("Status", Equal(corev1.ConditionTrue)),
					),
					And(
						HaveField("Type", Equal(appsv1beta4.ConditionBlueGreenUpdating)),
						HaveField("Status", Equal(corev1.ConditionTrue)),
					),
				)),
				HaveField("EmqxBlueGreenUpdateStatus", BeNil()),
				HaveField("ReadyReplicas", Equal(int32(1))),
				HaveField("EmqxNodes", ConsistOf(
					HaveField("Node", Equal(fmt.Sprintf("emqx-ee@%s-0.emqx-ee-headless.%s.svc.cluster.local", newSts.Name, emqx.GetNamespace()))),
				)),
				HaveField("CurrentStatefulSetVersion", Equal(newSts.Status.CurrentRevision)),
			))

			By("check old sts's pod is deleted after update")
			Eventually(func() []corev1.Pod {
				podList := &corev1.PodList{}
				_ = k8sClient.List(
					context.TODO(),
					podList,
					client.InNamespace(sts.GetNamespace()),
					client.MatchingLabels(map[string]string{
						"controller-revision-hash": sts.Status.CurrentRevision,
					}),
				)
				return podList.Items
			}, timeout, interval).Should(HaveLen(0))

			By("checking the EMQX Custom Resource's EndpointSlice after update", func() {
				checkPodAndEndpointsAndEndpointSlices(emqx, ports, []corev1.ServicePort{}, headlessPort, 1)
			})
		})
	})
})
