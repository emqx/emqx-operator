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
var _ = Describe("Check statefulSet", func() {
	It("should create a statefulSet", func() {
		check_statefulset(broker)
		check_statefulset(enterprise)
	})
})

func check_statefulset(emqx v1beta3.Emqx) {
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
	Expect(sts.Spec.Template.Spec.Containers).Should(HaveLen(2))
	Expect(sts.Spec.Template.Spec.Containers[0].ImagePullPolicy).Should(Equal(corev1.PullIfNotPresent))
	Expect(sts.Spec.Template.Spec.Containers[0].Resources).Should(Equal(emqx.GetResource()))
	Expect(sts.Spec.Template.Spec.Containers[0].Args).Should(Equal(emqx.GetArgs()))
	Expect(sts.Spec.Template.Spec.Containers[1].Args).Should(Equal([]string{"-u", "admin", "-p", "password", "-P", "8081"}))
	Expect(sts.Spec.Template.Spec.SecurityContext.FSGroup).Should(Equal(emqx.GetSecurityContext().FSGroup))
	Expect(sts.Spec.Template.Spec.SecurityContext.RunAsUser).Should(Equal(emqx.GetSecurityContext().RunAsUser))
	Expect(sts.Spec.Template.Spec.SecurityContext.SupplementalGroups).Should(Equal(emqx.GetSecurityContext().SupplementalGroups))
	// if emqx.GetInitContainers() != nil {
	// 	Expect(sts.Spec.Template.Spec.InitContainers[0].Name).Should(Equal(emqx.GetInitContainers()[0].Name))
	// 	Expect(sts.Spec.Template.Spec.InitContainers[0].Image).Should(Equal(emqx.GetInitContainers()[0].Image))
	// 	Expect(sts.Spec.Template.Spec.InitContainers[0].Args).Should(Equal(emqx.GetInitContainers()[0].Args))
	// }
}
