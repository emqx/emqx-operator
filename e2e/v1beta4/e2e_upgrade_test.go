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

			By("check currentStatefulSetVersion in CR status")
			Eventually(func() string {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				return ee.Status.CurrentStatefulSetVersion
			}, timeout, interval).Should(Equal(sts.Status.CurrentRevision))

			By("check emqx nodes in CR status")
			Eventually(func() string {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				if len(ee.GetStatus().GetEmqxNodes()) > 0 {
					return ee.GetStatus().GetEmqxNodes()[0].Node
				}
				return ""
			}, timeout, interval).Should(Equal(fmt.Sprintf("emqx-ee@%s-0.emqx-ee-headless.%s.svc.cluster.local", sts.Name, emqx.GetNamespace())))

			By("check running condition in CR status")
			Eventually(func() corev1.ConditionStatus {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				if ee.GetStatus().GetConditions()[0].Type == appsv1beta4.ConditionRunning {
					return ee.GetStatus().GetConditions()[0].Status
				}
				return corev1.ConditionUnknown
			}, timeout, interval).Should(Equal(corev1.ConditionTrue))

			By("checking the EMQX Custom Resource's EndpointSlice", func() {
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
			Expect(k8sClient.Update(context.Background(), emqx)).Should(Succeed())

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

			By("check emqx nodes in CR status")
			Eventually(func() []appsv1beta4.EmqxNode {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				return ee.GetStatus().GetEmqxNodes()
			}, timeout, interval).Should(HaveLen(2))

			By("check readyReplicas in CR status")
			Eventually(func() int {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				return int(ee.Status.ReadyReplicas)
			}, timeout, interval).Should(Equal(2))

			By("check blue-green status in CR status")
			blueGreenStatus := &appsv1beta4.EmqxBlueGreenUpdateStatus{}
			Eventually(func() bool {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				if ee.Status.EmqxBlueGreenUpdateStatus != nil &&
					ee.Status.EmqxBlueGreenUpdateStatus.EvacuationsStatus != nil &&
					ee.Status.EmqxBlueGreenUpdateStatus.StartedAt != nil {
					blueGreenStatus = ee.Status.EmqxBlueGreenUpdateStatus.DeepCopy()
					return true
				}
				return false
			}, timeout, interval).Should(BeTrue())
			Expect(blueGreenStatus.OriginStatefulSet).Should(Equal(sts.Name))
			Expect(blueGreenStatus.CurrentStatefulSet).Should(Equal(newSts.Name))

			By("check blue-green condition in CR status")
			Eventually(func() corev1.ConditionStatus {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				if ee.GetStatus().GetConditions()[0].Type == appsv1beta4.ConditionBlueGreenUpdating {
					return ee.GetStatus().GetConditions()[0].Status
				}
				return corev1.ConditionUnknown
			}, timeout, interval).Should(Equal(corev1.ConditionTrue))

			By("checking the EMQX Custom Resource's EndpointSlice when blue-green", func() {
				checkPodAndEndpointsAndEndpointSlices(emqx, ports, []corev1.ServicePort{}, headlessPort, 1)
			})

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

			By("check currentStatefulSetVersion in CR status")
			Eventually(func() string {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				return ee.Status.CurrentStatefulSetVersion
			}, timeout, interval).Should(Equal(newSts.Status.CurrentRevision))

			By("check emqx nodes in CR status")
			Eventually(func() string {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				if len(ee.GetStatus().GetEmqxNodes()) > 0 {
					return ee.GetStatus().GetEmqxNodes()[0].Node
				}
				return ""
			}, timeout, interval).Should(Equal(fmt.Sprintf("emqx-ee@%s-0.emqx-ee-headless.%s.svc.cluster.local", newSts.Name, emqx.GetNamespace())))

			By("check running condition in CR status")
			Eventually(func() corev1.ConditionStatus {
				ee := &appsv1beta4.EmqxEnterprise{}
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), ee)
				if ee.GetStatus().GetConditions()[0].Type == appsv1beta4.ConditionRunning {
					return ee.GetStatus().GetConditions()[0].Status
				}
				return corev1.ConditionUnknown
			}, timeout, interval).Should(Equal(corev1.ConditionTrue))
		})
	})
})
