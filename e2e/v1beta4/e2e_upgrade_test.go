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

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Upgrade Test", Label("upgrade"), func() {
	DescribeTable("upgrade emqx",
		func(emqx appsv1beta4.Emqx) {
			By("create EMQX", func() {
				createEmqx(emqx)
			})

			By("wait sts ready", func() {
				Eventually(func() []appsv1.StatefulSet {
					list := &appsv1.StatefulSetList{}
					_ = k8sClient.List(context.TODO(), list,
						client.InNamespace(emqx.GetNamespace()),
						client.MatchingLabels(emqx.GetLabels()),
					)
					return list.Items
				}, timeout, interval).Should(And(
					HaveLen(1),
					HaveEach(
						HaveField("Status", And(
							HaveField("ObservedGeneration", Equal(int64(1))),
							HaveField("ReadyReplicas", Equal(int32(1))),
						)),
					),
				))
			})

			By("checking the EMQX Custom Resource's EndpointSlice", func() {
				checkPodAndEndpointsAndEndpointSlices(emqx, ports, nil, headlessPort, 1)
			})

			By("update EMQX", func() {
				switch emqx.(type) {
				case *appsv1beta4.EmqxBroker:
					obj := &appsv1beta4.EmqxBroker{}
					Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), obj)).Should(Succeed())
					obj.Spec.Template.Spec.EmqxContainer.Image.Version = "4.4.17"
					Expect(k8sClient.Update(context.TODO(), obj)).Should(Succeed())
				case *appsv1beta4.EmqxEnterprise:
					obj := &appsv1beta4.EmqxEnterprise{}
					Expect(k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(emqx), obj)).Should(Succeed())
					obj.Spec.Template.Spec.EmqxContainer.Image.Version = "4.4.17"
					Expect(k8sClient.Update(context.TODO(), obj)).Should(Succeed())
				}
			})

			By("wait sts ready", func() {
				Eventually(func() []appsv1.StatefulSet {
					list := &appsv1.StatefulSetList{}
					_ = k8sClient.List(context.TODO(), list,
						client.InNamespace(emqx.GetNamespace()),
						client.MatchingLabels(emqx.GetLabels()),
					)
					return list.Items
				}, timeout, interval).Should(And(
					HaveLen(1),
					HaveEach(
						HaveField("Status", And(
							HaveField("ObservedGeneration", Equal(int64(2))),
							HaveField("ReadyReplicas", Equal(int32(1))),
						)),
					),
				))
			})

			By("checking the EMQX Custom Resource's EndpointSlice", func() {
				checkPodAndEndpointsAndEndpointSlices(emqx, ports, nil, headlessPort, 1)
			})

			By("delete EMQX", func() {
				deleteEmqx(emqx)
			})
		},
		Entry(nil, emqxBroker.DeepCopy()),
		Entry(nil, emqxEnterprise.DeepCopy()),
	)
})
