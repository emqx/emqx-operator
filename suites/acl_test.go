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
	Context("Check acl", func() {
		It("Check acl", func() {
			for _, emqx := range emqxList() {
				cm := &corev1.ConfigMap{}

				Eventually(func() bool {
					err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      emqx.GetACL()["name"],
							Namespace: emqx.GetNamespace(),
						},
						cm,
					)
					return err == nil
				}, tuneout, interval).Should(BeTrue())

				Expect(cm.Data).Should(Equal(map[string]string{
					"acl.conf": emqx.GetACL()["conf"],
				}))

			}
		})

		It("Update acl", func() {
			for _, emqx := range emqxList() {
				patch := []byte(`{"spec": {"acl": [{"permission": "deny"}]}}`)
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
							Name:      emqx.GetACL()["name"],
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
							"acl.conf": "{deny, all, pubsub, [\"#\"]}.\n",
						},
					)
				}, tuneout, interval).Should(BeTrue())
			}
			// TODO: check acl status by emqx api
			// TODO: test acl by mqtt pubsub
		})
	})
})
