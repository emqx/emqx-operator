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
	"reflect"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var _ = Describe("", func() {
	Context("Check service", func() {
		It("Check mqtts cert", func() {
			for _, emqx := range emqxList() {
				names := v1beta3.Names{Object: emqx}
				check_cert(
					emqx.GetListener().MQTTS.Cert,
					types.NamespacedName{
						Name:      names.MQTTSCertificate(),
						Namespace: emqx.GetNamespace(),
					},
				)
			}
		})
		It("Check wss cert", func() {
			for _, emqx := range emqxList() {
				names := v1beta3.Names{Object: emqx}
				check_cert(
					emqx.GetListener().WSS.Cert,
					types.NamespacedName{
						Name:      names.MQTTSCertificate(),
						Namespace: emqx.GetNamespace(),
					},
				)
			}
		})
	})
})

func check_cert(cert v1beta3.CertConf, namespacedName types.NamespacedName) {
	if !reflect.ValueOf(cert).IsZero() {
		secret := &corev1.Secret{}
		Eventually(func() bool {
			err := k8sClient.Get(
				context.Background(),
				namespacedName,
				secret,
			)
			return err == nil
		}, timeout, interval).Should(BeTrue())
		if !reflect.ValueOf(cert.Data).IsZero() {
			Expect(secret.Data).Should(HaveKeyWithValue("ca.crt", cert.Data.CaCert))
			Expect(secret.Data).Should(HaveKeyWithValue("tls.crt", cert.Data.TLSCert))
			Expect(secret.Data).Should(HaveKeyWithValue("tls.key", cert.Data.TLSKey))
		}
		// Any []byte slices will be converted to a base64-encoded string when encoding them to JSON.
		if !reflect.ValueOf(cert.StringData).IsZero() {
			Expect(secret.Data).Should(
				HaveKeyWithValue(
					"ca.crt",
					[]byte(cert.StringData.CaCert)),
			)
			Expect(secret.Data).Should(
				HaveKeyWithValue(
					"tls.crt",
					[]byte(cert.StringData.TLSCert)),
			)
			Expect(secret.Data).Should(
				HaveKeyWithValue(
					"tls.key",
					[]byte(cert.StringData.TLSKey)),
			)
		}
	}
}
