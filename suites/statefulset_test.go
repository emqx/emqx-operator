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

package suites_test

import (
	"context"
	"fmt"

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
				podList := &corev1.PodList{}

				sts := &appsv1.StatefulSet{}

				Eventually(func() bool {
					err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      emqx.GetName(),
							Namespace: emqx.GetNamespace(),
						},
						sts,
					)
					return err == nil
				}, tuneout, interval).Should(BeTrue())

				fmt.Printf("===================%+v\n", sts)

				Eventually(func() bool {
					err := k8sClient.List(
						context.Background(),
						podList,
						client.InNamespace(emqx.GetNamespace()),
					)
					return err == nil
				}, tuneout, interval).Should(BeTrue())

				fmt.Printf("===================%+v\n", podList)

				// Expect(sts.Status.ReadyReplicas).Should(Equal(*emqx.GetReplicas()))
			}
		})
	})
})
