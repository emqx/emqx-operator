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
	"fmt"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var _ = Describe("Check broker modules", func() {
	loadedModulesString := "{emqx_mod_acl_internal, true}.\n"
	Context("Check default modules", func() {
		It("should create a configMap with loaded modules", func() {
			check_modules_config(broker, loadedModulesString)
		})
		It("should create a annotation for sts with loaded modules", func() {
			check_modules_annotation(broker, loadedModulesString)
		})

	})
	Context("Check update modules", func() {
		JustBeforeEach(func() {
			loadedModulesString = "{emqx_mod_presence, false}.\n"
			modules := []v1beta3.EmqxBrokerModule{
				{
					Name:   "emqx_mod_presence",
					Enable: false,
				},
			}
			broker.Spec.EmqxTemplate.Modules = modules
			updateEmqx(broker)
		})

		It("should create a configMap with loaded modules", func() {
			check_modules_config(broker, loadedModulesString)
		})
		It("should create a annotation for sts with loaded modules", func() {
			check_modules_annotation(broker, loadedModulesString)
		})
	})
})

var _ = Describe("Check enterprise modules", func() {
	loadedModulesString := `[{"name":"internal_acl","enable":true,"configs":{"acl_rule_file":"/mounted/acl/acl.conf"}},{"name":"retainer","enable":true,"configs":{"expiry_interval":0,"max_payload_size":"1MB","max_retained_messages":0,"storage_type":"ram"}}]`
	Context("Check default modules", func() {
		It("should create a configMap with loaded modules", func() {
			check_modules_config(enterprise, loadedModulesString)
		})
		It("should create a annotation for sts with loaded modules", func() {
			check_modules_annotation(enterprise, loadedModulesString)
		})

	})
	Context("Check update modules", func() {
		JustBeforeEach(func() {
			loadedModulesString = `[{"name":"internal_acl","enable":true,"configs":{"acl_rule_file":"/mounted/acl/acl.conf"}}]`
			modules := []v1beta3.EmqxEnterpriseModule{
				{
					Name:    "internal_acl",
					Enable:  true,
					Configs: runtime.RawExtension{Raw: []byte(`{"acl_rule_file": "/mounted/acl/acl.conf"}`)},
				},
			}
			enterprise.Spec.EmqxTemplate.Modules = modules
			updateEmqx(enterprise)
		})

		It("should create a configMap with loaded modules", func() {
			check_modules_config(enterprise, loadedModulesString)
		})
		It("should create a annotation for sts with loaded modules", func() {
			check_modules_annotation(enterprise, loadedModulesString)
		})
	})
})

func check_modules_config(emqx v1beta3.Emqx, loadedModulesString string) {
	Eventually(func() map[string]string {
		cm := &corev1.ConfigMap{}
		_ = k8sClient.Get(
			context.Background(),
			types.NamespacedName{
				Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "loaded-modules"),
				Namespace: emqx.GetNamespace(),
			}, cm,
		)
		return cm.Data
	}, timeout, interval).Should(Equal(
		map[string]string{"loaded_modules": loadedModulesString},
	))
}

func check_modules_annotation(emqx v1beta3.Emqx, loadedModulesString string) {
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
			"LoadedModules/Base64EncodeConfig",
			base64.StdEncoding.EncodeToString([]byte(loadedModulesString)),
		),
	)
}
