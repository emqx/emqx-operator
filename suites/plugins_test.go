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
	"k8s.io/apimachinery/pkg/types"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var _ = Describe("", func() {
	Context("Check plugins", func() {
		// Define utility constants for object names and testing timeouts/durations and intervals.
		BeforeEach(func() {
			for _, emqx := range emqxList() {
				Expect(k8sClient.Create(context.Background(), emqx)).Should(Succeed())
			}
		})

		It("Check loaded plugins", func() {
			for _, emqx := range emqxList() {
				cm := &corev1.ConfigMap{}
				Eventually(func() bool {
					err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      emqx.GetLoadedPlugins()["name"],
							Namespace: emqx.GetNamespace(),
						},
						cm,
					)
					return err == nil
				}, Timeout, Interval).Should(BeTrue())

				Expect(cm.Data).Should(Equal(map[string]string{
					"loaded_plugins": emqx.GetLoadedPlugins()["conf"],
				}))
			}
		})

		It("Check update plugins", func() {
			for _, emqx := range emqxList() {
				patch := []byte(`{"spec":{"plugins":[{"enable": true, "name": "emqx_management"},{"enable": true, "name": "emqx_rule_engine"}]}}`)
				Expect(k8sClient.Patch(
					context.Background(),
					emqx,
					client.RawPatch(types.MergePatchType, patch),
				)).Should(Succeed())

				Eventually(func() bool {
					cm := &corev1.ConfigMap{}
					err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      emqx.GetLoadedPlugins()["name"],
							Namespace: emqx.GetNamespace(),
						},
						cm,
					)
					if err != nil {
						return false
					}
					return reflect.DeepEqual(
						cm.Data,
						map[string]string{
							"loaded_plugins": "{emqx_management, true}.\n{emqx_rule_engine, true}.\n",
						},
					)
					//return true
				}, Timeout, Interval).Should(BeTrue())
			}
			// TODO: check plugins status by emqx api
		})

		AfterEach(func() {
			for _, emqx := range emqxList() {
				Expect(DeleteAll(emqx)).ToNot(HaveOccurred())
				Eventually(func() bool {
					return EnsureDeleteAll(emqx)
				}, Timeout, Interval).Should(BeTrue())
			}
		})
	})
})
