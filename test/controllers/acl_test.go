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
	"k8s.io/apimachinery/pkg/types"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var _ = Describe("Check ACL", func() {
	aclString := "{allow, {user, \"dashboard\"}, subscribe, [\"$SYS/#\"]}.\n{allow, {ipaddr, \"127.0.0.1\"}, pubsub, [\"$SYS/#\", \"#\"]}.\n{deny, all, subscribe, [\"$SYS/#\", {eq, \"#\"}]}.\n{allow, all}.\n"

	Context("Check default ACL", func() {

		It("check acl config", func() {
			check_acl_config(broker, aclString)
			check_acl_config(enterprise, aclString)
		})

		It("check acl annotation", func() {
			check_acl_annotation(broker, aclString)
			check_acl_annotation(enterprise, aclString)
		})

		// TODO: check acl status by emqx api
		// TODO: test acl by mqtt pubsub
	})
	Context("Check update ACL", func() {
		JustBeforeEach(func() {
			aclString = "{deny, all}.\n"

			broker.SetACL([]string{`{deny, all}.`})
			updateEmqx(broker)
			enterprise.SetACL([]string{`{deny, all}.`})
			updateEmqx(enterprise)
		})

		It("check acl config", func() {
			check_acl_config(broker, aclString)
			check_acl_config(enterprise, aclString)
		})

		It("check acl annotation", func() {
			check_acl_annotation(broker, aclString)
			check_acl_annotation(enterprise, aclString)
		})
	})
})

func check_acl_config(emqx v1beta3.Emqx, aclString string) {
	Eventually(func() map[string]string {
		cm := &corev1.ConfigMap{}
		_ = k8sClient.Get(
			context.Background(),
			types.NamespacedName{
				Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "acl"),
				Namespace: emqx.GetNamespace(),
			},
			cm,
		)
		return cm.Data
	}, timeout, interval).Should(Equal(
		map[string]string{"acl.conf": aclString},
	))
}

func check_acl_annotation(emqx v1beta3.Emqx, aclString string) {
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
}
