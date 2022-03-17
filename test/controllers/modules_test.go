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

package controller_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

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
		It("Check loaded modles", func() {
			for _, emqx := range emqxList() {
				check_modules(emqx)
			}
		})

		It("Check update modules", func() {
			for _, emqx := range emqxList() {
				switch obj := emqx.(type) {
				case *v1beta3.EmqxBroker:
					modules := []v1beta3.EmqxBrokerModule{
						{
							Name:   "emqx_mod_presence",
							Enable: false,
						},
					}
					obj.Spec.EmqxTemplate.Modules = modules
					Expect(updateEmqx(obj)).Should(Succeed())
					check_modules(obj)
				case *v1beta3.EmqxEnterprise:
					modules := []v1beta3.EmqxEnterpriseModule{
						{
							Name:    "internal_cal",
							Enable:  true,
							Configs: runtime.RawExtension{Raw: []byte(`{"acl_rule_file": "/mounted/acl/acl.conf"}`)},
						},
					}
					obj.Spec.EmqxTemplate.Modules = modules
					Expect(updateEmqx(obj)).Should(Succeed())
					check_modules(obj)
				default:
					Fail("Type of emqx not found")
				}
			}
		})
	})
})

func check_modules(emqx v1beta3.Emqx) {
	switch obj := emqx.(type) {
	case *v1beta3.EmqxBroker:
		modules := &v1beta3.EmqxBrokerModuleList{
			Items: obj.Spec.EmqxTemplate.Modules,
		}
		loadedModulesString := modules.String()

		Eventually(func() map[string]string {
			cm := &corev1.ConfigMap{}
			_ = k8sClient.Get(
				context.Background(),
				types.NamespacedName{
					Name:      fmt.Sprintf("%s-%s", obj.GetName(), "loaded-modules"),
					Namespace: obj.GetNamespace(),
				}, cm,
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
					Name:      obj.GetName(),
					Namespace: obj.GetNamespace(),
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
	case *v1beta3.EmqxEnterprise:
		data, _ := json.Marshal(obj.Spec.EmqxTemplate.Modules)
		loadedModulesString := string(data)

		Eventually(func() map[string]string {
			cm := &corev1.ConfigMap{}
			_ = k8sClient.Get(
				context.Background(),
				types.NamespacedName{
					Name:      fmt.Sprintf("%s-%s", obj.GetName(), "loaded-modules"),
					Namespace: obj.GetNamespace(),
				}, cm,
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
					Name:      obj.GetName(),
					Namespace: obj.GetNamespace(),
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
	default:
		Fail("Type of emqx not found")
	}
}
