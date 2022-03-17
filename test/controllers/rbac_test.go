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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var _ = Describe("", func() {
	Context("Check RBAC", func() {
		It("Check exist", func() {
			for _, emqx := range emqxList() {
				sa := &corev1.ServiceAccount{}
				role := &rbacv1.Role{}
				roleBinding := &rbacv1.RoleBinding{}
				Eventually(func() error {
					if err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      emqx.GetName(),
							Namespace: emqx.GetNamespace(),
						},
						sa,
					); err != nil {
						return err
					}
					if err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      emqx.GetName(),
							Namespace: emqx.GetNamespace(),
						},
						role,
					); err != nil {
						return err
					}
					if err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      emqx.GetName(),
							Namespace: emqx.GetNamespace(),
						},
						roleBinding,
					); err != nil {
						return err
					}
					return nil
				}, timeout, interval).ShouldNot(HaveOccurred())
			}
		})
	})
})
