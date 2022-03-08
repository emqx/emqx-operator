package controller_suite_test

import (
	"context"

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
	Context("Check telegraf conf", func() {
		It("Check telegraf conf", func() {
			for _, emqx := range emqxList() {
				check_telegraf(emqx)
			}
		})

	})
})

func check_telegraf(emqx v1beta2.Emqx) {

	Eventually(func() map[string]string {
		cm := &corev1.ConfigMap{}
		_ = k8sClient.Get(
			context.Background(),
			types.NamespacedName{
				Namespace: emqx.GetNamespace(),
				Name:      util.NameForTelegraf(emqx),
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
