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

package controller_suite_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var _ = Describe("", func() {
	Context("Check modules", func() {
		It("Check emqx broker loaded modules", func() {
			broker := generateEmqxBroker(brokerName, brokerNameSpace)

			cm := &corev1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Name:      broker.GetLoadedModules()["name"],
						Namespace: broker.GetNamespace(),
					}, cm,
				)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(cm.Data).Should(Equal(map[string]string{
				"loaded_modules": broker.GetLoadedModules()["conf"],
			}))
		})

		It("Update emqx broker loaded modules", func() {
			broker := generateEmqxBroker(brokerName, brokerNameSpace)

			patch := []byte(`{"spec":{"modules":[{"name": "emqx_mod_presence", "enable": false}]}}`)
			Expect(k8sClient.Patch(
				context.Background(),
				broker,
				client.RawPatch(types.MergePatchType, patch),
			)).Should(Succeed())

			Eventually(func() map[string]string {
				cm := &corev1.ConfigMap{}
				_ = k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Name:      broker.GetLoadedModules()["name"],
						Namespace: broker.GetNamespace(),
					},
					cm,
				)
				return cm.Data
			}, timeout, interval).Should(Equal(
				map[string]string{"loaded_modules": "{emqx_mod_presence, false}.\n"},
			))
			// TODO: check modules status by emqx api
		})

		It("Check emqx enterprise loaded modules", func() {
			enterprise := generateEmqxEnterprise(enterpriseName, enterpriseNameSpace)

			cm := &corev1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Name:      enterprise.GetLoadedModules()["name"],
						Namespace: enterprise.GetNamespace(),
					}, cm,
				)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(cm.Data).Should(Equal(map[string]string{
				"loaded_modules": enterprise.GetLoadedModules()["conf"],
			}))
		})

		It("Update emqx enterprise loaded modules", func() {
			enterprise := generateEmqxEnterprise(enterpriseName, enterpriseNameSpace)

			patch := []byte(`{"spec":{"modules":[{"name": "internal_acl", "enable": false, "configs": {"acl_rule_file": "etc/acl.conf"}}]}}`)
			Expect(k8sClient.Patch(
				context.Background(),
				enterprise,
				client.RawPatch(types.MergePatchType, patch),
			)).Should(Succeed())

			Eventually(func() map[string]string {
				cm := &corev1.ConfigMap{}
				_ = k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Name:      enterprise.GetLoadedModules()["name"],
						Namespace: enterprise.GetNamespace(),
					},
					cm,
				)
				return cm.Data
			}, timeout, interval).Should(Equal(
				map[string]string{
					"loaded_modules": "[{\"name\":\"internal_acl\",\"configs\":{\"acl_rule_file\":\"etc/acl.conf\"}}]",
				},
			))
			// TODO: check modules status by emqx api
		})
	})
})
