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

	appsv1 "k8s.io/api/apps/v1"
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
				}, timeout, interval).Should(BeTrue())

				Expect(cm.Data).Should(Equal(map[string]string{
					"acl.conf": emqx.GetACL()["conf"],
				}))

				Eventually(func() map[string]string {
					sts := &appsv1.StatefulSet{}
					_ = k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      emqx.GetName(),
							Namespace: emqx.GetNamespace(),
						},
						sts,
					)
					return sts.Spec.Template.Annotations
				}, timeout, interval).Should(
					HaveKeyWithValue("ACL/ResourceVersion", cm.ResourceVersion),
				)
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

				cm := &corev1.ConfigMap{}
				Eventually(func() map[string]string {
					_ = k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      emqx.GetACL()["name"],
							Namespace: emqx.GetNamespace(),
						},
						cm,
					)
					return cm.Data
				}, timeout, interval).Should(Equal(
					map[string]string{
						"acl.conf": "{deny, all, pubsub, [\"#\"]}.\n",
					},
				))

				Eventually(func() map[string]string {
					sts := &appsv1.StatefulSet{}
					_ = k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      emqx.GetName(),
							Namespace: emqx.GetNamespace(),
						},
						sts,
					)
					return sts.Spec.Template.Annotations
				}, timeout, interval).Should(
					HaveKeyWithValue("ACL/ResourceVersion", cm.ResourceVersion),
				)
			}
			// TODO: check acl status by emqx api
			// TODO: test acl by mqtt pubsub
		})
	})
})
