package webhook_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

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
		It("Check validate", func() {
			checkValidation(
				&v1beta3.EmqxBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "broker",
						Namespace: "default",
					},
				},
			)
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
		It("Check validation", func() {
			checkValidation(
				&v1beta3.EmqxEnterprise{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "enterprise",
						Namespace: "default",
					},
				},
			)
		})
	})
})

func checkValidation(emqx v1beta3.Emqx) {
	emqx.SetImage("emqx/emqx:fake")
	Expect(k8sClient.Create(context.Background(), emqx)).ShouldNot(Succeed())

	emqx.SetImage("emqx/emqx:latest")
	Expect(k8sClient.Create(context.Background(), emqx)).Should(Succeed())

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

	var obj v1beta3.Emqx

	switch emqx.(type) {
	case *v1beta3.EmqxBroker:
		obj = &v1beta3.EmqxBroker{}
	case *v1beta3.EmqxEnterprise:
		obj = &v1beta3.EmqxEnterprise{}
	}

	obj.SetName(emqx.GetName())
	obj.SetNamespace(emqx.GetNamespace())
	obj.SetResourceVersion(emqx.GetResourceVersion())
	obj.SetCreationTimestamp(emqx.GetCreationTimestamp())
	obj.SetManagedFields(emqx.GetManagedFields())

	obj.SetImage("emqx/emqx:fake")
	Expect(k8sClient.Update(context.Background(), obj)).ShouldNot(Succeed())
	obj.SetImage("emqx/emqx:latest")
	Expect(k8sClient.Update(context.Background(), obj)).Should(Succeed())
}
