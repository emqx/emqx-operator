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

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var rebalance = appsv1beta4.Rebalance{
	ObjectMeta: metav1.ObjectMeta{
		Name: "rebalance",
	},
	Spec: appsv1beta4.RebalanceSpec{
		RebalanceStrategy: appsv1beta4.RebalanceStrategy{
			WaitTakeover:     10,
			ConnEvictRate:    10,
			SessEvictRate:    10,
			WaitHealthCheck:  10,
			AbsSessThreshold: 100,
			RelConnThreshold: "1.2",
			AbsConnThreshold: 100,
			RelSessThreshold: "1.2",
		},
	},
}

var _ = Describe("Emqx Rebalance Test", Label("rebalance"), func() {
	Describe("Enterprise is not found", func() {
		var r *appsv1beta4.Rebalance
		BeforeEach(func() {
			r = rebalance.DeepCopy()
			r.Namespace = emqxEnterprise.GetNamespace() + "-" + rand.String(5)
			r.Spec.InstanceName = "fake"
			Expect(k8sClient.Create(context.TODO(), &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: r.Namespace,
					Labels: map[string]string{
						"test": "e2e",
					},
				},
			})).Should(Succeed())
			Expect(k8sClient.Create(context.TODO(), r)).Should(Succeed())
		})

		AfterEach(func() {
			Eventually(func() error {
				return k8sClient.Delete(context.TODO(), r)
			}, timeout, interval).Should(Succeed())
			Eventually(func() bool {
				return k8sErrors.IsNotFound(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(r), r))
			}).Should(BeTrue())
		})

		It("Rebalance will failed, because the EMQX Enterprise is not found", func() {
			Eventually(func() appsv1beta4.RebalanceStatus {
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(r), r)
				return r.Status
			}, timeout, interval).Should(And(
				HaveField("Phase", appsv1beta4.RebalancePhaseFailed),
				HaveField("RebalanceStates", BeNil()),
				HaveField("Conditions", ContainElements(
					HaveField("Type", appsv1beta4.RebalanceConditionFailed),
				)),
				HaveField("Conditions", ContainElements(
					HaveField("Status", corev1.ConditionTrue),
				)),
				HaveField("Conditions", ContainElements(
					HaveField("Message", fmt.Sprintf("EMQX Enterprise %s is not found", r.Spec.InstanceName)),
				)),
			))
		})
	})

	Describe("Enterprise is exist", func() {
		var emqx *appsv1beta4.EmqxEnterprise
		var r *appsv1beta4.Rebalance
		BeforeEach(func() {
			emqx = emqxEnterprise.DeepCopy()
			r = rebalance.DeepCopy()
			createEmqx(emqx)
			Eventually(func() appsv1beta4.EmqxStatus {
				_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), emqx)
				return &emqx.Status
			}, timeout, interval).Should(HaveField("Conditions", ContainElements(HaveField("Type", appsv1beta4.ConditionRunning))))
		})

		AfterEach(func() {
			By("Rebalance can be deleted", func() {
				Eventually(func() error {
					return k8sClient.Delete(context.TODO(), r)
				}, timeout, interval).Should(Succeed())
				Eventually(func() bool {
					return k8sErrors.IsNotFound(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(r), r))
				}).Should(BeTrue())
			})

			deleteEmqx(emqx)
		})

		It("Check rebalance status", func() {
			By("Create rebalance", func() {
				r.Namespace = emqx.GetNamespace()
				r.Spec.InstanceName = emqx.GetName()
				Expect(k8sClient.Create(context.TODO(), r)).Should(Succeed())
			})

			By("Rebalance should have finalizer", func() {
				Eventually(func() []string {
					_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(r), r)
					return r.GetFinalizers()
				}, timeout, interval).Should(ContainElements("apps.emqx.io/finalizer"))
			})

			By("Rebalance will failed, because the EMQX Enterprise is nothing to balance", func() {
				Eventually(func() appsv1beta4.RebalanceStatus {
					_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(r), r)
					return r.Status
				}, timeout, interval).Should(And(
					HaveField("Phase", appsv1beta4.RebalancePhaseFailed),
					HaveField("RebalanceStates", BeNil()),
					HaveField("Conditions", ContainElements(
						And(
							HaveField("Type", appsv1beta4.RebalanceConditionFailed),
							HaveField("Status", corev1.ConditionTrue),
							HaveField("Message", "Failed to start rebalance: nothing_to_balance"),
						),
					)),
				))
			})

			By("Mock rebalance is inProgress", func() {
				// mock rebalance processing
				r.Status.Phase = appsv1beta4.RebalancePhaseProcessing
				r.Status.Conditions = []appsv1beta4.RebalanceCondition{}
				Expect(k8sClient.Status().Patch(context.TODO(), r.DeepCopy(), client.MergeFrom(r))).Should(Succeed())

				// update annotations for target reconciler
				Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(r), r)).Should(Succeed())
				r.Annotations = map[string]string{"test": "e2e"}
				Expect(k8sClient.Patch(context.TODO(), r.DeepCopy(), client.MergeFrom(r))).Should(Succeed())
			})

			By("Rebalance should completed", func() {
				Eventually(func() appsv1beta4.RebalanceStatus {
					_ = k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(r), r)
					return r.Status
				}, timeout, interval).Should(And(
					HaveField("Phase", appsv1beta4.RebalancePhaseCompleted),
					HaveField("RebalanceStates", BeNil()),
					HaveField("Conditions", ContainElements(
						HaveField("Type", appsv1beta4.RebalanceConditionCompleted),
					)),
					HaveField("Conditions", ContainElements(
						HaveField("Status", corev1.ConditionTrue),
					)),
				))
			})
		})
	})
})
