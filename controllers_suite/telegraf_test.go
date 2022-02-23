package controller_suite_test

import (
	"context"
	"fmt"

	"github.com/emqx/emqx-operator/apis/apps/v1beta1"
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
	Context("Check telegraf Conf", func() {
		It("Check loaded plugins", func() {
			for _, emqx := range emqxList() {
				check_telegraf(emqx)
			}
		})

		It("Check update plugins", func() {
			for _, emqx := range emqxList() {
				plugins := []v1beta1.Plugin{
					{
						Name:   "emqx_management",
						Enable: true,
					},
					{
						Name:   "emqx_rule_engine",
						Enable: true,
					},
				}
				emqx.SetPlugins(plugins)
				Expect(updateEmqx(emqx)).Should(Succeed())

				check_plugins(emqx)
			}
		})
	})
})

func check_telegraf(emqx v1beta1.Emqx) {

	Eventually(func() map[string]string {
		cm := &corev1.ConfigMap{}
		_ = k8sClient.Get(
			context.Background(),
			types.NamespacedName{
				Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "telegraf-config"),
				Namespace: emqx.GetNamespace(),
			},
			cm,
		)
		return cm.Data
	}, timeout, interval).Should(Equal(map[string]string{
		"telegraf.conf": *emqx.GetTelegrafTemplate().Conf,
	}))

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
		return sts.Spec.Template.Spec.Containers[1].Name
	}, timeout, interval).Should(Equal("telegraf"))
}
