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

	"github.com/emqx/emqx-operator/api/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
							Name:      emqx.GetServiceAccountName(),
							Namespace: emqx.GetNamespace(),
						},
						sa,
					); err != nil {
						return err
					}
					if err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      emqx.GetServiceAccountName(),
							Namespace: emqx.GetNamespace(),
						},
						role,
					); err != nil {
						return err
					}
					if err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      emqx.GetServiceAccountName(),
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
	Context("Check external RBAC", func() {
		BeforeEach(func() {
			for _, emqx := range emqxList() {
				meta := metav1.ObjectMeta{
					Name:      "external",
					Namespace: emqx.GetNamespace(),
				}

				sa := &corev1.ServiceAccount{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "ServiceAccount",
					},
					ObjectMeta: meta,
				}

				role := &rbacv1.Role{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "rbac.authorization.k8s.io/v1",
						Kind:       "Role",
					},
					ObjectMeta: meta,
					Rules: []rbacv1.PolicyRule{
						{
							Verbs:     []string{"get", "watch", "list"},
							APIGroups: []string{""},
							Resources: []string{"endpoints"},
						},
					},
				}

				roleBinding := &rbacv1.RoleBinding{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "rbac.authorization.k8s.io/v1",
						Kind:       "RoleBinding",
					},
					ObjectMeta: meta,
					Subjects: []rbacv1.Subject{
						{
							Kind:      sa.Kind,
							Name:      sa.Name,
							Namespace: sa.Namespace,
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     role.Kind,
						Name:     role.Name,
					},
				}

				Expect(k8sClient.Create(context.Background(), sa)).Should(Succeed())
				Expect(k8sClient.Create(context.Background(), role)).Should(Succeed())
				Expect(k8sClient.Create(context.Background(), roleBinding)).Should(Succeed())
			}
		})

		It("Use external RBAC", func() {
			for _, emqx := range emqxList() {
				patch := []byte(`{"spec": {"serviceAccountName": "external"}}`)
				Expect(k8sClient.Patch(
					context.Background(),
					emqx,
					client.RawPatch(types.MergePatchType, patch),
				)).Should(Succeed())

				Eventually(func() string {
					sts := &appsv1.StatefulSet{}
					_ = k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      emqx.GetName(),
							Namespace: emqx.GetNamespace(),
						},
						sts,
					)
					return sts.Spec.Template.Spec.ServiceAccountName
				}, timeout, interval).Should(Equal("external"))
			}
		})

		AfterEach(func() {
			for _, emqx := range emqxList() {
				if broker, ok := emqx.(*v1beta1.EmqxBroker); ok {
					old := &v1beta1.EmqxBroker{}
					_ = k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      emqx.GetName(),
							Namespace: emqx.GetNamespace(),
						},
						old,
					)
					broker.ResourceVersion = old.ResourceVersion
					Expect(k8sClient.Update(
						context.Background(),
						broker,
					)).Should(Succeed())
				}
				if enterprise, ok := emqx.(*v1beta1.EmqxEnterprise); ok {
					old := &v1beta1.EmqxEnterprise{}
					_ = k8sClient.Get(
						context.Background(),
						types.NamespacedName{
							Name:      emqx.GetName(),
							Namespace: emqx.GetNamespace(),
						},
						old,
					)
					enterprise.ResourceVersion = old.ResourceVersion
					Expect(k8sClient.Update(
						context.Background(),
						enterprise,
					)).Should(Succeed())
				}

				meta := metav1.ObjectMeta{
					Name:      "external",
					Namespace: emqx.GetNamespace(),
				}
				Expect(k8sClient.Delete(
					context.Background(),
					&rbacv1.RoleBinding{
						ObjectMeta: meta,
					},
				)).Should(Succeed())
				Expect(k8sClient.Delete(
					context.Background(),
					&rbacv1.Role{
						ObjectMeta: meta,
					},
				)).Should(Succeed())
				Expect(k8sClient.Delete(
					context.Background(),
					&corev1.ServiceAccount{
						ObjectMeta: meta,
					},
				)).Should(Succeed())

			}
		})
	})
})
