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

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var _ = Describe("", func() {
	Context("Check license", func() {
		It("Check license", func() {
			emqx := generateEmqxEnterprise(enterpriseName, enterpriseNameSpace)
			check_license(emqx)
		})
	})
})

func check_license(emqx v1beta3.Emqx) {
	names := v1beta3.Names{Object: emqx}
	emqxEneterprise, _ := emqx.(*v1beta3.EmqxEnterprise)
	Eventually(func() map[string][]byte {
		secret := &corev1.Secret{}
		_ = k8sClient.Get(
			context.Background(),
			types.NamespacedName{
				Namespace: emqx.GetNamespace(),
				Name:      names.License(),
			},
			secret,
		)
		return secret.Data
	}, timeout, interval).Should(Equal(
		map[string][]byte{"emqx.lic": []byte(emqxEneterprise.GetLicense().Data)}),
	)
}
