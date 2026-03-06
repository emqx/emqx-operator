/*
Copyright 2025.

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

package e2e_helm

import (
	"encoding/json"

	. "github.com/emqx/emqx-operator/test/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

const (
	// helmReleaseName is the Helm release name used across all Helm upgrade tests.
	helmReleaseName = "emqx-operator"

	// old22xChartVersion is the 2.2.x chart version from the emqx Helm repo.
	old22xChartVersion = "2.2.29"

	// localChartPath is the path to the local 2.3.0 chart being tested.
	localChartPath = "deploy/charts/emqx-operator"
)

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

// cleanup23x removes all resources that 2.3.x chart may have left behind in
// the given namespace, including the Helm release itself.
func cleanup23x(namespace string) {
	_ = Run("helm", "uninstall", helmReleaseName, "--namespace", namespace)
	_ = Kubectl("delete", "clusterrole", "emqx-operator-manager-role", "--ignore-not-found")
	_ = Kubectl("delete", "clusterrolebinding", "emqx-operator-manager-rolebinding", "--ignore-not-found")
	_ = Kubectl("delete", "clusterrole", "emqx-operator-pre-upgrade", "--ignore-not-found")
	_ = Kubectl("delete", "clusterrolebinding", "emqx-operator-pre-upgrade", "--ignore-not-found")
	_ = Kubectl("delete", "crd", "emqxes.apps.emqx.io", "--ignore-not-found")
	_ = Kubectl("delete", "crd", "rebalances.apps.emqx.io", "--ignore-not-found")
	_ = Kubectl("delete", "ns", namespace, "--ignore-not-found")
}

// getCRD fetches a CRD by name as JSON.
func getCRD(name string) (*apiextv1.CustomResourceDefinition, error) {
	out, err := KubectlOut("get", "crd", name, "-o", "json")
	if err != nil {
		return nil, err
	}
	var crd apiextv1.CustomResourceDefinition
	if err := json.Unmarshal([]byte(out), &crd); err != nil {
		return nil, err
	}
	return &crd, nil
}

// crdExists checks whether a CRD exists in the cluster.
func crdExists(name string) bool {
	return Kubectl("get", "crd", name) == nil
}

// resourceExists checks whether a cluster-scoped resource exists.
func resourceExists(kind, name string) bool {
	return Kubectl("get", kind, name) == nil
}

// dumpHelmDiagnostics writes debug info for the given namespace to GinkgoWriter.
func dumpHelmDiagnostics(namespace string) {
	out, _ := Output("helm", "list", "--namespace", namespace)
	GinkgoWriter.Print("Helm releases:\n", out)
	out, _ = KubectlOut("get", "jobs", "--namespace", namespace, "-o", "yaml")
	GinkgoWriter.Print("Jobs in namespace:\n", out)
	out, _ = KubectlOut("get", "pods", "--namespace", namespace, "-o", "wide")
	GinkgoWriter.Print("Pods in namespace:\n", out)
	out, _ = KubectlOut("get", "crds", "-o", "custom-columns=NAME:.metadata.name,CONVERSION:.spec.conversion.strategy")
	GinkgoWriter.Print("CRDs:\n", out)
	out, _ = KubectlOut("get", "events", "--namespace", namespace, "--sort-by=.lastTimestamp")
	GinkgoWriter.Print("Events:\n", out)
}

// install22x installs the 2.2.x operator chart into the given namespace.
// Does not use --wait because the deployment won't become ready without cert-manager.
func install22x(namespace string) {
	Expect(Run("helm", "install",
		helmReleaseName,
		"emqx/emqx-operator",
		"--version", old22xChartVersion,
		"--namespace", namespace,
		"--set", "cert-manager.enable=false",
		"--timeout", "2m",
	)).To(Succeed())
}

// install23x installs the local 2.3.0 chart into the given namespace with --wait.
func install23x(namespace string, extraArgs ...string) {
	args := []string{
		"install",
		helmReleaseName,
		localChartPath,
		"--namespace", namespace,
		"--set", "image.repository=emqx/emqx-operator",
		"--set", "image.tag=0.0.1",
		"--set", "image.pullPolicy=Never",
		"--wait",
		"--timeout", "3m",
	}
	args = append(args, extraArgs...)
	Expect(Run("helm", args...)).To(Succeed())
}

// upgrade23x upgrades to the local 2.3.0 chart in the given namespace with --wait.
func upgrade23x(namespace string) error {
	return Run("helm", "upgrade",
		helmReleaseName,
		localChartPath,
		"--namespace", namespace,
		"--set", "image.repository=emqx/emqx-operator",
		"--set", "image.tag=0.0.1",
		"--set", "image.pullPolicy=Never",
		"--wait",
		"--timeout", "3m",
	)
}

//nolint:errcheck
var _ = Describe("Helm Fresh Install", Ordered, func() {

	const namespace = "emqx-operator-fresh"

	BeforeEach(func() {
		By("ensure clean state")
		cleanup23x(namespace)

		By("create namespace")
		Expect(Kubectl("create", "ns", namespace)).To(Succeed())
	})

	AfterEach(func() {
		cleanup23x(namespace)
	})

	AfterEach(func() {
		if CurrentSpecReport().Failed() {
			dumpHelmDiagnostics(namespace)
		}
	})

	It("should install cleanly / cleanup no-op", func() {
		By("install 2.3.0 chart")
		install23x(namespace)

		By("verify CRDs are installed")
		Expect(crdExists("emqxes.apps.emqx.io")).To(BeTrue())
		Expect(crdExists("rebalances.apps.emqx.io")).To(BeTrue())

		By("verify operator deployment is available")
		Expect(Kubectl("wait", "deployment",
			"emqx-operator-controller-manager",
			"--for", "condition=Available",
			"--namespace", namespace,
			"--timeout", "2m",
		)).To(Succeed())

		By("verify no legacy CRDs were created")
		Expect(crdExists("emqxbrokers.apps.emqx.io")).To(BeFalse())
		Expect(crdExists("emqxenterprises.apps.emqx.io")).To(BeFalse())
		Expect(crdExists("emqxplugins.apps.emqx.io")).To(BeFalse())
	})

	It("should install cleanly / pre-upgrade check disabled", func() {
		By("install 2.3.0 with pre-upgrade check disabled")
		install23x(namespace, "--set", "upgrade.preUpgradeCheck=false")

		By("verify operator is running")
		Expect(Kubectl("wait", "deployment",
			"emqx-operator-controller-manager",
			"--for", "condition=Available",
			"--namespace", namespace,
			"--timeout", "2m",
		)).To(Succeed())

		By("verify CRDs are installed")
		Expect(crdExists("emqxes.apps.emqx.io")).To(BeTrue())
		Expect(crdExists("rebalances.apps.emqx.io")).To(BeTrue())
	})
})

//nolint:errcheck
var _ = Describe("Helm Upgrade / 2.2.x", Ordered, func() {

	const namespace = "emqx-operator-helm"

	BeforeAll(func() {
		By("clean up any leftover resources from previous test runs")
		cleanup23x(namespace)
		cleanup22x()

		By("create test namespace")
		Expect(Kubectl("create", "ns", namespace)).To(Succeed())
	})

	AfterAll(func() {
		cleanup23x(namespace)
		cleanup22x()
	})

	AfterEach(func() {
		if CurrentSpecReport().Failed() {
			dumpHelmDiagnostics(namespace)
		}
	})

	It("should install 2.2.x and upgrade to 2.3.0", func() {
		By("install emqx-operator 2.2.x from Helm repo")
		install22x(namespace)

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

		By("upgrade to 2.3.0 chart with default values (cleanup enabled)")
		Expect(upgrade23x(namespace)).To(Succeed())

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
			crd, err := getCRD(crdName)
			Expect(err).NotTo(HaveOccurred(), "CRD %s should still exist", crdName)
			if crd.Spec.Conversion != nil {
				Expect(crd.Spec.Conversion.Strategy).To(
					Equal(apiextv1.NoneConverter),
					"%s should have None conversion strategy after upgrade", crdName,
				)
			}
		}

		By("verify webhook configurations no longer exist")
		Expect(resourceExists("mutatingwebhookconfiguration",
			"emqx-operator-mutating-webhook-configuration")).To(BeFalse())
		Expect(resourceExists("validatingwebhookconfiguration",
			"emqx-operator-validating-webhook-configuration")).To(BeFalse())

		By("verify 2.3.0 operator deployment is running")
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
		cleanup23x(namespace)
		cleanup22x()

		By("create test namespace")
		Expect(Kubectl("create", "ns", namespace)).To(Succeed())
	})

	AfterAll(func() {
		cleanup23x(namespace)
		cleanup22x()
	})

	AfterEach(func() {
		if CurrentSpecReport().Failed() {
			dumpHelmDiagnostics(namespace)
			out, _ := KubectlOut("logs", "--namespace", namespace,
				"-l", "app.kubernetes.io/name=emqx-operator", "--tail", "50")
			GinkgoWriter.Print("Pre-upgrade job logs:\n", out)
		}
	})

	It("should block upgrade when legacy CRs exist", func() {
		By("install 2.2.x operator")
		install22x(namespace)

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

		By("attempt upgrade to 2.3.0 — should fail because legacy CRs exist")
		Expect(Run("helm", "upgrade",
			helmReleaseName,
			localChartPath,
			"--namespace", namespace,
			"--set", "image.repository=emqx/emqx-operator",
			"--set", "image.tag=0.0.1",
			"--set", "image.pullPolicy=Never",
			"--wait",
			"--timeout", "2m",
		)).To(HaveOccurred(), "upgrade should fail when legacy CRs exist")

		By("verify the Helm release is in failed state")
		out, err := Output("helm", "list", "--namespace", namespace)
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring("failed"),
			"release should be in failed state after blocked upgrade")

		By("delete the legacy CR so cleanup can proceed")
		Expect(Kubectl("delete", "emqxbrokers", "test-broker",
			"--namespace", namespace, "--ignore-not-found")).To(Succeed())

		By("retry upgrade")
		Expect(upgrade23x(namespace)).To(Succeed())

		By("verify 2.3.0 operator is running")
		Expect(Kubectl("wait", "deployment",
			"emqx-operator-controller-manager",
			"--for", "condition=Available",
			"--namespace", namespace,
			"--timeout", "2m",
		)).To(Succeed())
	})
})
