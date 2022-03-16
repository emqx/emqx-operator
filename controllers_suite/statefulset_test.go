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
	Context("Check statefulset", func() {
		It("Check statefulset", func() {
			for _, emqx := range emqxList() {
				sts := &appsv1.StatefulSet{}
				Eventually(func() error {
					err := k8sClient.Get(
						context.TODO(),
						types.NamespacedName{
							Name:      emqx.GetName(),
							Namespace: emqx.GetNamespace(),
						},
						sts,
					)
					return err
				}, timeout, interval).Should(Succeed())

				Expect(sts.Spec.Replicas).Should(Equal(emqx.GetReplicas()))
				Expect(sts.Spec.Template.Labels).Should(Equal(emqx.GetLabels()))
				Expect(sts.Spec.Template.Spec.Affinity).Should(Equal(emqx.GetAffinity()))
				Expect(sts.Spec.Template.Spec.Containers[0].ImagePullPolicy).Should(Equal(corev1.PullIfNotPresent))
				Expect(sts.Spec.Template.Spec.Containers[0].Resources).Should(Equal(emqx.GetResource()))
			}
		})
	})
})
