package v1beta3

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("EMQX Broker", func() {
	Context("Check EMQX Broker", func() {
		AfterEach(func() {
			Expect(k8sClient.Delete(
				context.Background(),
				&EmqxBroker{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "broker",
						Namespace: "default",
					},
				},
			)).Should(Succeed())
		})
		It("Check validate", func() {
			checkValidation(
				&EmqxBroker{
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
				&EmqxEnterprise{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "enterprise",
						Namespace: "default",
					},
				},
			)).Should(Succeed())
		})
		It("Check validation", func() {
			checkValidation(
				&EmqxEnterprise{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "enterprise",
						Namespace: "default",
					},
				},
			)
		})
	})
})

var _ = Describe("EMQX Enterprise License", func() {
	Context("Check EMQX Enterprise License", func() {
		emqxEnterprise := &EmqxEnterprise{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "enterprise",
				Namespace: "default",
			},
			Spec: EmqxEnterpriseSpec{
				EmqxTemplate: EmqxEnterpriseTemplate{
					License: License{
						SecretName: "license",
					},
				},
			},
		}

		AfterEach(func() {
			Expect(k8sClient.Delete(context.Background(), emqxEnterprise)).Should(Succeed())
		})
		It("Check validation", func() {
			checkLicenseValidation(emqxEnterprise)
		})
	})
})

var _ = Describe("EMQX Plugin", func() {
	Context("Check EMQX Plugin", func() {
		namespace := "default"
		management := &EmqxPlugin{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "emqx-management",
				Namespace: namespace,
			},
			Spec: EmqxPluginSpec{
				PluginName: "emqx_management",
			},
		}
		retainer := &EmqxPlugin{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "emqx-retainer",
				Namespace: namespace,
			},
			Spec: EmqxPluginSpec{
				PluginName: "emqx_retainer",
			},
		}
		dashboard := &EmqxPlugin{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "emqx-dashboard",
				Namespace: namespace,
			},
			Spec: EmqxPluginSpec{
				PluginName: "emqx_dashboard",
			},
		}
		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), management)).ShouldNot(Succeed())
			Expect(k8sClient.Create(context.Background(), retainer)).Should(Succeed())
			Expect(k8sClient.Create(context.Background(), dashboard)).ShouldNot(Succeed())
		})
		AfterEach(func() {
			Expect(k8sClient.Delete(context.Background(), retainer)).Should(Succeed())
		})
		It("Check Plugin validateUpdate ", func() {
			checkValidationUpdate(retainer)
		})
	})
})

func checkValidationUpdate(plugin *EmqxPlugin) {
	Eventually(func() error {
		err := k8sClient.Get(
			context.TODO(),
			types.NamespacedName{
				Name:      plugin.Name,
				Namespace: plugin.Namespace,
			},
			plugin,
		)
		return err
	}, timeout, interval).Should(Succeed())

	if plugin.Spec.Config == nil {
		plugin.Spec.Config = map[string]string{}
	}
	plugin.Spec.Config["test"] = "test"
	Expect(k8sClient.Update(context.Background(), plugin)).Should(Succeed())
	plugin.Spec.PluginName = "test"
	Expect(k8sClient.Update(context.Background(), plugin)).ShouldNot(Succeed())
}

func checkValidation(emqx Emqx) {
	emqx.SetImage("emqx/emqx:4.3.3")
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

	var obj Emqx

	switch emqx.(type) {
	case *EmqxBroker:
		obj = &EmqxBroker{}
	case *EmqxEnterprise:
		obj = &EmqxEnterprise{}
	}

	obj.SetName(emqx.GetName())
	obj.SetNamespace(emqx.GetNamespace())
	obj.SetResourceVersion(emqx.GetResourceVersion())
	obj.SetCreationTimestamp(emqx.GetCreationTimestamp())
	obj.SetManagedFields(emqx.GetManagedFields())

	obj.SetImage("emqx:4.3")
	Expect(k8sClient.Update(context.Background(), obj)).ShouldNot(Succeed())
	obj.SetImage("emqx:4")
	Expect(k8sClient.Update(context.Background(), obj)).Should(Succeed())

	obj.SetImage("127.0.0.1:8443/emqx/emqx:4.3.3")
	Expect(k8sClient.Update(context.Background(), obj)).ShouldNot(Succeed())
	obj.SetImage("127.0.0.1:8443/emqx/emqx:4.4.11")
	Expect(k8sClient.Update(context.Background(), obj)).Should(Succeed())

	obj.SetPassword("test")
	Expect(k8sClient.Update(context.Background(), obj)).ShouldNot(Succeed())

	obj.SetUsername("test")
	Expect(k8sClient.Update(context.Background(), obj)).ShouldNot(Succeed())
}

func checkLicenseValidation(emqx *EmqxEnterprise) {
	license := emqx.GetLicense()

	license.Data = []byte("test")
	emqx.SetLicense(license)
	Expect(k8sClient.Create(context.Background(), emqx)).ShouldNot(Succeed())

	license.StringData = "test"
	emqx.SetLicense(license)
	Expect(k8sClient.Create(context.Background(), emqx)).ShouldNot(Succeed())

	license.StringData = "test"
	license.Data = []byte("test")
	emqx.SetLicense(license)
	Expect(k8sClient.Create(context.Background(), emqx)).ShouldNot(Succeed())

	license.StringData = ""
	license.Data = []byte("")
	emqx.SetLicense(license)
	Expect(k8sClient.Create(context.Background(), emqx)).Should(Succeed())

	license.StringData = "test"
	emqx.SetLicense(license)
	Expect(k8sClient.Update(context.Background(), emqx)).ShouldNot(Succeed())

	license.Data = []byte("test")
	emqx.SetLicense(license)
	Expect(k8sClient.Update(context.Background(), emqx)).ShouldNot(Succeed())

	license.StringData = "test"
	license.Data = []byte("test")
	emqx.SetLicense(license)
	Expect(k8sClient.Update(context.Background(), emqx)).ShouldNot(Succeed())
}
