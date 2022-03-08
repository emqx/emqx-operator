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

	"github.com/emqx/emqx-operator/apis/apps/v1beta2"
	"github.com/emqx/emqx-operator/pkg/util"
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
	Context("Check acl", func() {
		It("Check acl", func() {
			for _, emqx := range emqxList() {
				check_acl(emqx)
			}
		})

		It("Update acl", func() {
			for _, emqx := range emqxList() {
				acl := []v1beta2.ACL{
					{
						Permission: "deny",
					},
				}
				emqx.SetACL(acl)
				Expect(updateEmqx(emqx)).Should(Succeed())

				check_acl(emqx)
			}

		})
	})
})

func check_acl(emqx v1beta2.Emqx) {
	aclString := util.StringACL(emqx.GetACL())

	Eventually(func() map[string]string {
		cm := &corev1.ConfigMap{}
		_ = k8sClient.Get(
			context.Background(),
			types.NamespacedName{
				Name:      util.NameForACL(emqx),
				Namespace: emqx.GetNamespace(),
			},
			cm,
		)
		return cm.Data
	}, timeout, interval).Should(Equal(
		map[string]string{"acl.conf": aclString},
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
		return sts.Annotations
	}, timeout, interval).Should(
		HaveKeyWithValue(
			"ACL/Base64EncodeConfig",
			base64.StdEncoding.EncodeToString([]byte(aclString)),
		),
	)
	// TODO: check acl status by emqx api
	// TODO: test acl by mqtt pubsub
}
