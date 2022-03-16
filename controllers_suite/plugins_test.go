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

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var _ = Describe("", func() {
	Context("Check plugins", func() {
		It("Check loaded plugins", func() {
			for _, emqx := range emqxList() {
				check_plugins(emqx)
			}
		})

		It("Check update plugins", func() {
			for _, emqx := range emqxList() {
				plugins := []v1beta3.Plugin{
					{
						Name:   "emqx_management",
						Enable: true,
					},
					{
						Name:   "emqx_rule_engine",
						Enable: true,
					},
					{
						Name:   "emqx_prometheus",
						Enable: true,
					},
				}
				emqx.SetPlugins(plugins)
				Expect(updateEmqx(emqx)).Should(Succeed())

				check_plugins(emqx)
			}
		})
	})
})

func check_plugins(emqx v1beta3.Emqx) {
	names := v1beta3.Names{Object: emqx}
	plugins := &v1beta3.PluginList{
		Items: emqx.GetPlugins(),
	}
	loadedPluginsString := plugins.String()

	Eventually(func() map[string]string {
		cm := &corev1.ConfigMap{}
		_ = k8sClient.Get(
			context.Background(),
			types.NamespacedName{
				Namespace: emqx.GetNamespace(),
				Name:      names.Plugins(),
			},
			cm,
		)
		return cm.Data
	}, timeout, interval).Should(Equal(map[string]string{
		"loaded_plugins": loadedPluginsString,
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
		return sts.Annotations
	}, timeout, interval).Should(
		HaveKeyWithValue(
			"LoadedPlugins/Base64EncodeConfig",
			base64.StdEncoding.EncodeToString([]byte(loadedPluginsString)),
		),
	)
	// TODO: check plugins status by emqx api
}
