package webhook_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("EMQX Broker", func() {
	Context("Check EMQX Broker", func() {
		AfterEach(func() {
			Expect(k8sClient.Delete(
				context.Background(),
				&v1beta3.EmqxBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "broker",
						Namespace: "default",
					},
				},
			)).Should(Succeed())
		})
		It("Check defaulting", func() {
			v1beta3EmqxBroker := &v1beta3.EmqxBroker{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "broker",
					Namespace: "default",
				},
			}
			Expect(k8sClient.Create(context.Background(), v1beta3EmqxBroker)).Should(Succeed())
			checkDefaulting(v1beta3EmqxBroker)
		})
		It("Check defaulting with telegraf", func() {
			v1beta3EmqxBroker := &v1beta3.EmqxBroker{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "broker",
					Namespace: "default",
				},
			}

			telegrafConf := `[global_tags]
			instanceID = "test"
			[[outputs.discard]]`

			v1beta3EmqxBroker.Spec.TelegrafTemplate = &v1beta3.TelegrafTemplate{
				Image: "telegraf:1.19.3",
				Conf:  &telegrafConf,
			}
			Expect(k8sClient.Create(context.Background(), v1beta3EmqxBroker)).Should(Succeed())
			checkDefaultingWithTelegraf(v1beta3EmqxBroker)
		})
	})
})

var _ = Describe("EMQX Enterprise", func() {
	Context("Check EMQX Enterprise", func() {
		AfterEach(func() {
			Expect(k8sClient.Delete(
				context.Background(),
				&v1beta3.EmqxEnterprise{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "enterprise",
						Namespace: "default",
					},
				},
			)).Should(Succeed())
		})
		It("Check defaulting", func() {
			v1beta3EmqxEnterprise := &v1beta3.EmqxEnterprise{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "enterprise",
					Namespace: "default",
				},
			}
			Expect(k8sClient.Create(context.Background(), v1beta3EmqxEnterprise)).Should(Succeed())
			checkDefaulting(v1beta3EmqxEnterprise)
		})
		It("Check defaulting with telegraf", func() {
			v1beta3EmqxEnterprise := &v1beta3.EmqxEnterprise{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "enterprise",
					Namespace: "default",
				},
			}

			telegrafConf := `[global_tags]
			instanceID = "test"
			[[outputs.discard]]`

			v1beta3EmqxEnterprise.Spec.TelegrafTemplate = &v1beta3.TelegrafTemplate{
				Image: "telegraf:1.19.3",
				Conf:  &telegrafConf,
			}
			Expect(k8sClient.Create(context.Background(), v1beta3EmqxEnterprise)).Should(Succeed())
			checkDefaultingWithTelegraf(v1beta3EmqxEnterprise)
		})
	})
})

func checkDefaulting(emqx v1beta3.Emqx) {
	Eventually(func() error {
		err := k8sClient.Get(
			context.TODO(),
			types.NamespacedName{
				Name:      emqx.GetName(),
				Namespace: emqx.GetNamespace(),
			},
			emqx,
		)
		return err
	}, timeout, interval).Should(Succeed())

	Expect(emqx.GetLabels()).Should(HaveKeyWithValue("apps.emqx.io/managed-by", "emqx-operator"))
	Expect(emqx.GetLabels()).Should(HaveKeyWithValue("apps.emqx.io/instance", emqx.GetName()))

	replicas := int32(3)
	Expect(emqx.GetReplicas()).Should(Equal(&replicas))

	Expect(emqx.GetEnv()).Should(
		ContainElements(
			corev1.EnvVar{
				Name:  "EMQX_CLUSTER__DISCOVERY",
				Value: "k8s",
			},
		),
	)
}

func checkDefaultingWithTelegraf(emqx v1beta3.Emqx) {
	Eventually(func() error {
		err := k8sClient.Get(
			context.TODO(),
			types.NamespacedName{
				Name:      emqx.GetName(),
				Namespace: emqx.GetNamespace(),
			},
			emqx,
		)
		return err
	}, timeout, interval).Should(Succeed())

	Expect(emqx.GetPlugins()).Should(
		ContainElements(
			v1beta3.Plugin{
				Name:   "emqx_prometheus",
				Enable: true,
			},
		),
	)
	Expect(emqx.GetEnv()).Should(
		ContainElements(
			corev1.EnvVar{
				Name:  "EMQX_PROMETHEUS__PUSH__GATEWAY__SERVER",
				Value: "",
			},
		),
	)
}
