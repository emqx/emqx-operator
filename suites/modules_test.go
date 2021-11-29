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

package suites_test

import (
	"context"
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/emqx/emqx-operator/api/v1alpha2"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var _ = Describe("", func() {
	Context("Check modules", func() {
		// Define utility constants for object names and testing timeouts/durations and intervals.
		BeforeEach(func() {
			ctx := context.Background()

			sa, role, roleBinding := GenerateRBAC(BrokerName, BrokerNameSpace)
			Expect(k8sClient.Create(ctx, sa)).Should(Succeed())
			Expect(k8sClient.Create(ctx, role)).Should(Succeed())
			Expect(k8sClient.Create(ctx, roleBinding)).Should(Succeed())

			broker := GenerateEmqxBroker(BrokerName, BrokerNameSpace)
			Expect(k8sClient.Create(ctx, broker)).Should(Succeed())

			sa, role, roleBinding = GenerateRBAC(EnterpriseName, EnterpriseNameSpace)
			Expect(k8sClient.Create(ctx, sa)).Should(Succeed())
			Expect(k8sClient.Create(ctx, role)).Should(Succeed())
			Expect(k8sClient.Create(ctx, roleBinding)).Should(Succeed())

			enterprise := GenerateEmqxEnterprise(EnterpriseName, EnterpriseNameSpace)
			Expect(k8sClient.Create(ctx, enterprise)).Should(Succeed())
		})

		It("Check emqx broker loaded modules", func() {
			ctx := context.Background()
			broker := GenerateEmqxBroker(BrokerName, BrokerNameSpace)

			cm := &corev1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(
					ctx,
					types.NamespacedName{
						Name:      broker.GetLoadedModules()["name"],
						Namespace: broker.GetNamespace(),
					}, cm,
				)
				if err != nil {
					return false
				}
				return true
			}, Timeout, Interval).Should(BeTrue())

			Expect(cm.Data).Should(Equal(map[string]string{
				"loaded_modules": broker.GetLoadedModules()["conf"],
			}))
		})

		It("Update emqx broker loaded modules", func() {
			ctx := context.Background()
			broker := GenerateEmqxBroker(BrokerName, BrokerNameSpace)

			patch := []byte(`{"spec":{"modules":[{"name": "emqx_mod_presence", "enable": false}]}}`)
			Expect(k8sClient.Patch(
				ctx,
				broker,
				client.RawPatch(types.MergePatchType, patch),
			)).Should(Succeed())

			Eventually(func() bool {
				cm := &corev1.ConfigMap{}
				err := k8sClient.Get(
					ctx,
					types.NamespacedName{
						Name:      broker.GetLoadedModules()["name"],
						Namespace: broker.GetNamespace(),
					},
					cm,
				)
				if err != nil {
					return false
				}
				return reflect.DeepEqual(cm.Data, map[string]string{"loaded_modules": "{emqx_mod_presence, false}.\n"})
			}, Timeout, Interval).Should(BeTrue())

			// TODO: check modules status by emqx api
		})

		It("Check emqx enterprise loaded modules", func() {
			ctx := context.Background()
			enterprise := GenerateEmqxEnterprise(EnterpriseName, EnterpriseNameSpace)

			cm := &corev1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(
					ctx,
					types.NamespacedName{
						Name:      enterprise.GetLoadedModules()["name"],
						Namespace: enterprise.GetNamespace(),
					}, cm,
				)
				if err != nil {
					return false
				}
				return true
			}, Timeout, Interval).Should(BeTrue())

			Expect(cm.Data).Should(Equal(map[string]string{
				"loaded_modules": enterprise.GetLoadedModules()["conf"],
			}))
		})

		It("Update emqx enterprise loaded modules", func() {
			ctx := context.Background()
			enterprise := GenerateEmqxEnterprise(EnterpriseName, EnterpriseNameSpace)

			patch := []byte(`{"spec":{"modules":[{"name": "internal_acl", "enable": false, "configs": {"acl_rule_file": "etc/acl.conf"}}]}}`)
			Expect(k8sClient.Patch(
				ctx,
				enterprise,
				client.RawPatch(types.MergePatchType, patch),
			)).Should(Succeed())

			Eventually(func() bool {
				cm := &corev1.ConfigMap{}
				err := k8sClient.Get(
					ctx,
					types.NamespacedName{
						Name:      enterprise.GetLoadedModules()["name"],
						Namespace: enterprise.GetNamespace(),
					},
					cm,
				)
				if err != nil {
					return false
				}
				return reflect.DeepEqual(
					cm.Data,
					map[string]string{
						"loaded_modules": "[{\"name\":\"internal_acl\",\"configs\":{\"acl_rule_file\":\"etc/acl.conf\"}}]",
					},
				)
			}, Timeout, Interval).Should(BeTrue())

			// TODO: check modules status by emqx api
		})

		AfterEach(func() {
			emqx := &v1alpha2.EmqxBroker{
				ObjectMeta: metav1.ObjectMeta{
					Name:      BrokerName,
					Namespace: BrokerNameSpace,
				},
			}
			Expect(DeleteAll(emqx)).ToNot(HaveOccurred())
			Eventually(func() bool {
				return EnsureDeleteAll(emqx)
			}, Timeout, Interval).Should(BeTrue())

			emqxEnterprise := &v1alpha2.EmqxEnterprise{
				ObjectMeta: metav1.ObjectMeta{
					Name:      EnterpriseName,
					Namespace: EnterpriseNameSpace,
				},
			}
			Expect(DeleteAll(emqxEnterprise)).ToNot(HaveOccurred())
			Eventually(func() bool {
				return EnsureDeleteAll(emqxEnterprise)
			}, Timeout, Interval).Should(BeTrue())
		})
	})
})
