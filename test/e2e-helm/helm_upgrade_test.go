/*
Copyright 2026.

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

package helm

import (
	. "github.com/emqx/emqx-operator/test/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

const (
	// v22xChartVersion is the 2.2.x chart version from the emqx Helm repo.
	v22xChartVersion = "2.2.29"
)

// helmInstall22x installs the 2.2.x operator chart into the given namespace.
// Does not use --wait because the deployment won't become ready without cert-manager.
func helmInstall22x(namespace string) error {
	return Run("helm", "install",
		helmReleaseName,
		"emqx/emqx-operator",
		"--version", v22xChartVersion,
		"--namespace", namespace,
		"--set", "cert-manager.enable=false",
	)
}

// cleanup22x removes all resources that 2.2.x chart may have left behind.
func cleanup22x() {
	_ = Kubectl("delete", "mutatingwebhookconfiguration",
		"emqx-operator-mutating-webhook-configuration", "--ignore-not-found")
	_ = Kubectl("delete", "validatingwebhookconfiguration",
		"emqx-operator-validating-webhook-configuration", "--ignore-not-found")
	_ = Kubectl("delete", "crd", "emqxbrokers.apps.emqx.io", "--ignore-not-found")
	_ = Kubectl("delete", "crd", "emqxenterprises.apps.emqx.io", "--ignore-not-found")
	_ = Kubectl("delete", "crd", "emqxplugins.apps.emqx.io", "--ignore-not-found")
	_ = Kubectl("delete", "ns", "emqx-operator-system", "--ignore-not-found")
}

//nolint:errcheck
var _ = Describe("Helm Upgrade / 2.2.x", Ordered, func() {

	const namespace = "emqx-operator-helm"

	BeforeAll(func() {
		By("clean up any leftover resources from previous test runs")
		helmCleanup(namespace)
		cleanup22x()

		By("create test namespace")
		Expect(Kubectl("create", "ns", namespace)).To(Succeed())
	})

	AfterAll(func() {
		helmCleanup(namespace)
		cleanup22x()
	})

	AfterEach(func() {
		if CurrentSpecReport().Failed() {
			dumpHelmDiagnostics(namespace)
		}
	})

	It("should install 2.2.x and upgrade to 2.3.x", func() {
		By("install emqx-operator 2.2.x from Helm repo")
		Expect(helmInstall22x(namespace)).To(Succeed())

		By("verify 2.2.x deployment and all 5 CRDs exist")
		Expect(Kubectl("get", "deployment", "emqx-operator-controller-manager",
			"--namespace", namespace,
		)).To(Succeed())
		for _, crd := range []string{
			"emqxes.apps.emqx.io",
			"rebalances.apps.emqx.io",
			"emqxbrokers.apps.emqx.io",
			"emqxenterprises.apps.emqx.io",
			"emqxplugins.apps.emqx.io",
		} {
			Expect(crdExists(crd)).To(
				BeTrue(),
				"%s CRD should exist after 2.2.x install", crd,
			)
		}

		By("upgrade to 2.3.x chart with default values (cleanup enabled)")
		Expect(helmUpgrade(namespace)).To(Succeed())

		By("verify pre-upgrade completed successfully")
		var jobList batchv1.JobList
		Eventually(KubectlOut).
			WithArguments("get", "jobs", "--namespace", namespace, "-o", "json").
			Should(
				// Hook jobs may be cleaned up already (hook-succeeded policy),
				// which is fine: it means they completed successfully.
				BeUnmarshalledAs(&jobList, HaveField("Items", Or(
					BeEmpty(),
					ContainElement(And(
						HaveField("Name", Equal("pre-upgrade")),
						HaveField("Status.Succeeded", BeNumerically(">=", 1)),
					)),
				))),
				"pre-upgrade job should have completed successfully",
			)

		By("verify CRDs no longer have conversion webhooks")
		for _, crdName := range []string{"emqxes.apps.emqx.io", "rebalances.apps.emqx.io"} {
			var crd apiextv1.CustomResourceDefinition
			Expect(KubectlOut("get", "crd", crdName, "-o", "json")).To(
				BeUnmarshalledAs(&crd, Or(
					HaveField("Spec.Conversion", BeNil()),
					HaveField("Spec.Conversion.Strategy", Equal(apiextv1.NoneConverter)),
				)),
				"%s should have None or no conversion strategy after upgrade", crdName,
			)
		}

		By("verify webhook configurations no longer exist")
		Expect(resourceExists("mutatingwebhookconfiguration",
			"emqx-operator-mutating-webhook-configuration")).To(BeFalse())
		Expect(resourceExists("validatingwebhookconfiguration",
			"emqx-operator-validating-webhook-configuration")).To(BeFalse())

		By("verify 2.3.x operator deployment is running")
		var deployment appsv1.Deployment
		Eventually(KubectlOut).
			WithArguments("get", "deployment",
				"emqx-operator-controller-manager",
				"--namespace", namespace,
				"-o", "json",
			).
			Should(BeUnmarshalledAs(&deployment,
				HaveField("Status.AvailableReplicas", BeNumerically(">=", 1)),
			))

		By("verify legacy CRDs were removed by Helm")
		Expect(crdExists("emqxbrokers.apps.emqx.io")).To(BeFalse(),
			"emqxbrokers CRD should be removed after upgrade")
		Expect(crdExists("emqxenterprises.apps.emqx.io")).To(BeFalse(),
			"emqxenterprises CRD should be removed after upgrade")
		Expect(crdExists("emqxplugins.apps.emqx.io")).To(BeFalse(),
			"emqxplugins CRD should be removed after upgrade")
	})
})

//nolint:errcheck
var _ = Describe("Helm Upgrade / 2.2.x + legacy CRs", Ordered, func() {

	const namespace = "emqx-operator-legacy"

	BeforeAll(func() {
		By("clean up any leftover resources")
		helmCleanup(namespace)
		cleanup22x()

		By("create test namespace")
		Expect(Kubectl("create", "ns", namespace)).To(Succeed())
	})

	AfterAll(func() {
		helmCleanup(namespace)
		cleanup22x()
	})

	AfterEach(func() {
		if CurrentSpecReport().Failed() {
			dumpHelmDiagnostics(namespace)
			out, _ := KubectlOut("logs", "--namespace", namespace,
				"-l", "app.kubernetes.io/name=emqx-operator",
				"--tail", "-1")
			GinkgoWriter.Print("Pre-upgrade job logs:\n", out)
		}
	})

	It("should block upgrade when legacy CRs exist", func() {
		By("install 2.2.x operator")
		Expect(helmInstall22x(namespace)).To(Succeed())

		By("verify legacy CRD exists")
		Expect(crdExists("emqxbrokers.apps.emqx.io")).To(BeTrue())

		By("delete webhook configurations so we can create a CR without admission")
		Expect(Kubectl("delete", "mutatingwebhookconfiguration",
			"emqx-operator-mutating-webhook-configuration", "--ignore-not-found")).To(Succeed())
		Expect(Kubectl("delete", "validatingwebhookconfiguration",
			"emqx-operator-validating-webhook-configuration", "--ignore-not-found")).To(Succeed())

		By("patch emqxbrokers CRD to remove conversion webhook")
		Expect(Kubectl("patch", "crd", "emqxbrokers.apps.emqx.io",
			"--type", "json",
			"--patch", `[{"op":"replace","path":"/spec/conversion","value":{"strategy":"None"}}]`,
		)).To(Succeed())

		By("create a minimal EmqxBroker CR")
		brokerCR := []byte(`{
			"apiVersion": "apps.emqx.io/v1beta4",
			"kind": "EmqxBroker",
			"metadata": {"name": "test-broker", "namespace": "` + namespace + `"},
			"spec": {}
		}`)
		Expect(KubectlStdin(brokerCR, "apply", "-f", "-", "--namespace", namespace)).
			To(Succeed())

		By("attempt upgrade to 2.3.x")
		Expect(helmUpgrade(namespace)).To(HaveOccurred(),
			"upgrade should fail when legacy CRs exist")

		By("verify the Helm release is in failed state")
		out, err := Output("helm", "list", "--namespace", namespace)
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring("failed"),
			"release should be in failed state after blocked upgrade")

		By("delete the legacy CR so cleanup can proceed")
		Expect(Kubectl("delete", "emqxbrokers", "test-broker",
			"--namespace", namespace, "--ignore-not-found")).To(Succeed())

		By("retry upgrade")
		Expect(helmUpgrade(namespace)).To(Succeed())

		By("verify 2.3.x operator is running")
		Expect(Kubectl("wait", "deployment",
			"emqx-operator-controller-manager",
			"--for", "condition=Available",
			"--namespace", namespace,
			"--timeout", "1m",
		)).To(Succeed())
	})
})
