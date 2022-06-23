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
	"os"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var _ = Describe("", func() {
	Context("Check emqx custom resource", func() {
		It("Check status", func() {
			if os.Getenv("USE_EXISTING_CLUSTER") == "true" {
				for _, emqx := range emqxList() {
					var instance v1beta3.Emqx
					switch emqx.(type) {
					case *v1beta3.EmqxBroker:
						instance = &v1beta3.EmqxBroker{}
					case *v1beta3.EmqxEnterprise:
						instance = &v1beta3.EmqxEnterprise{}
					}

					Eventually(func() v1beta3.ConditionType {
						_ = k8sClient.Get(
							context.TODO(),
							types.NamespacedName{
								Name:      emqx.GetName(),
								Namespace: emqx.GetNamespace(),
							},
							instance,
						)
						return instance.GetStatus().Conditions[0].Type
					}, timeout, instance).Should(Equal(v1beta3.ClusterConditionRunning))
				}
			}
		})
	})
})
