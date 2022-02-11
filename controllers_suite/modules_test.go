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
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/emqx/emqx-operator/apis/apps/v1beta1"
	"github.com/emqx/emqx-operator/pkg/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var _ = Describe("", func() {
	Context("Check modules", func() {
		It("Check emqx broker loaded modules", func() {
			broker := generateEmqxBroker(brokerName, brokerNameSpace)
			loadedModulesString := util.StringEmqxBrokerLoadedModules(broker.Spec.Modules)

			cm := &corev1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Name:      fmt.Sprintf("%s-%s", broker.GetName(), "loaded-modules"),
						Namespace: broker.GetNamespace(),
					}, cm,
				)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(cm.Data).Should(Equal(
				map[string]string{"loaded_modules": loadedModulesString},
			))

			Eventually(func() map[string]string {
				sts := &appsv1.StatefulSet{}
				_ = k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Name:      broker.GetName(),
						Namespace: broker.GetNamespace(),
					},
					sts,
				)
				return sts.Annotations
			}, timeout, interval).Should(
				HaveKeyWithValue(
					"LoadedModules/Base64EncodeConfig",
					base64.StdEncoding.EncodeToString([]byte(loadedModulesString)),
				),
			)
		})

		It("Update emqx broker loaded modules", func() {
			broker := generateEmqxBroker(brokerName, brokerNameSpace)

			modules := []v1beta1.EmqxBrokerModules{
				{
					Name:   "emqx_mod_presence",
					Enable: false,
				},
			}
			broker.Spec.Modules = modules

			loadedModulesString := util.StringEmqxBrokerLoadedModules(broker.Spec.Modules)

			Expect(k8sClient.Update(
				context.Background(),
				broker,
			)).Should(Succeed())

			cm := &corev1.ConfigMap{}
			Eventually(func() map[string]string {
				_ = k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Name:      fmt.Sprintf("%s-%s", broker.GetName(), "loaded-modules"),
						Namespace: broker.GetNamespace(),
					},
					cm,
				)
				return cm.Data
			}, timeout, interval).Should(Equal(
				map[string]string{"loaded_modules": loadedModulesString},
			))

			Eventually(func() map[string]string {
				sts := &appsv1.StatefulSet{}
				_ = k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Name:      broker.GetName(),
						Namespace: broker.GetNamespace(),
					},
					sts,
				)
				return sts.Annotations
			}, timeout, interval).Should(
				HaveKeyWithValue(
					"LoadedModules/Base64EncodeConfig",
					base64.StdEncoding.EncodeToString([]byte(loadedModulesString)),
				),
			)
			// TODO: check modules status by emqx api
		})

		It("Check emqx enterprise loaded modules", func() {
			enterprise := generateEmqxEnterprise(enterpriseName, enterpriseNameSpace)
			data, _ := json.Marshal(enterprise.Spec.Modules)
			loadedModulesString := string(data)

			cm := &corev1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Name:      fmt.Sprintf("%s-%s", enterprise.GetName(), "loaded-modules"),
						Namespace: enterprise.GetNamespace(),
					}, cm,
				)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(cm.Data).Should(Equal(
				map[string]string{"loaded_modules": loadedModulesString},
			))

			Eventually(func() map[string]string {
				sts := &appsv1.StatefulSet{}
				_ = k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Name:      enterprise.GetName(),
						Namespace: enterprise.GetNamespace(),
					},
					sts,
				)
				return sts.Annotations
			}, timeout, interval).Should(
				HaveKeyWithValue(
					"LoadedModules/Base64EncodeConfig",
					base64.StdEncoding.EncodeToString([]byte(loadedModulesString)),
				),
			)
		})

		It("Update emqx enterprise loaded modules", func() {
			enterprise := generateEmqxEnterprise(enterpriseName, enterpriseNameSpace)
			modules := []v1beta1.EmqxEnterpriseModules{
				{
					Name:    "internal_cal",
					Enable:  true,
					Configs: runtime.RawExtension{Raw: []byte(`{"acl_rule_file": "etc/acl.conf"}`)},
				},
			}
			enterprise.Spec.Modules = modules

			data, _ := json.Marshal(enterprise.Spec.Modules)
			loadedModulesString := string(data)

			patch := []byte(`{"spec":{"modules":[{"name": "internal_acl", "enable": false, "configs": {"acl_rule_file": "etc/acl.conf"}}]}}`)
			Expect(k8sClient.Patch(
				context.Background(),
				enterprise,
				client.RawPatch(types.MergePatchType, patch),
			)).Should(Succeed())

			cm := &corev1.ConfigMap{}
			Eventually(func() map[string]string {
				_ = k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Name:      fmt.Sprintf("%s-%s", enterprise.GetName(), "loaded-modules"),
						Namespace: enterprise.GetNamespace(),
					},
					cm,
				)
				return cm.Data
			}, timeout, interval).Should(Equal(
				map[string]string{"loaded_modules": loadedModulesString},
			))

			Eventually(func() map[string]string {
				sts := &appsv1.StatefulSet{}
				_ = k8sClient.Get(
					context.Background(),
					types.NamespacedName{
						Name:      enterprise.GetName(),
						Namespace: enterprise.GetNamespace(),
					},
					sts,
				)
				return sts.Annotations
			}, timeout, interval).Should(
				HaveKeyWithValue(
					"LoadedModules/Base64EncodeConfig",
					base64.StdEncoding.EncodeToString([]byte(loadedModulesString)),
				),
			)
			// TODO: check modules status by emqx api
		})
	})
})
