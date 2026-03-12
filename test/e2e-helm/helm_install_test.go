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
)

// helmInstall installs the local 2.3.x chart into the given namespace with --wait.
func helmInstall(namespace string, extraArgs ...string) error {
	args := []string{
		"install",
		helmReleaseName,
		localChartPath,
		"--namespace", namespace,
		"--set", "image.repository=" + operatorImageRepo,
		"--set", "image.tag=" + operatorImageTag,
		"--set", "image.pullPolicy=Never",
		"--wait",
		"--timeout", "2m",
	}
	args = append(args, extraArgs...)
	return Run("helm", args...)
}

// helmUpgrade upgrades to the local 2.3.x chart in the given namespace with --wait.
func helmUpgrade(namespace string) error {
	return Run("helm", "upgrade",
		helmReleaseName,
		localChartPath,
		"--namespace", namespace,
		"--set", "image.repository="+operatorImageRepo,
		"--set", "image.tag="+operatorImageTag,
		"--set", "image.pullPolicy=Never",
		"--wait",
		"--timeout", "2m",
	)
}

// helmCleanup removes all resources that 2.3.x chart may have left behind in
// the given namespace, including the Helm release itself.
func helmCleanup(namespace string) {
	_ = Run("helm", "uninstall", helmReleaseName, "--namespace", namespace)
	_ = Kubectl("delete", "clusterrole", "emqx-operator-manager-role", "--ignore-not-found")
	_ = Kubectl("delete", "clusterrolebinding", "emqx-operator-manager-rolebinding", "--ignore-not-found")
	_ = Kubectl("delete", "clusterrole", "emqx-operator-pre-upgrade", "--ignore-not-found")
	_ = Kubectl("delete", "clusterrolebinding", "emqx-operator-pre-upgrade", "--ignore-not-found")
	_ = Kubectl("delete", "crd", "emqxes.apps.emqx.io", "--ignore-not-found")
	_ = Kubectl("delete", "crd", "rebalances.apps.emqx.io", "--ignore-not-found")
	_ = Kubectl("delete", "ns", namespace, "--ignore-not-found")
}

//nolint:errcheck
var _ = Describe("Helm Install", Ordered, func() {

	const namespace = "emqx-operator-fresh"

	BeforeEach(func() {
		By("ensure clean state")
		helmCleanup(namespace)

		By("create namespace")
		Expect(Kubectl("create", "ns", namespace)).To(Succeed())
	})

	AfterEach(func() {
		helmCleanup(namespace)
	})

	AfterEach(func() {
		if CurrentSpecReport().Failed() {
			dumpHelmDiagnostics(namespace)
		}
	})

	It("should install cleanly / cleanup no-op", func() {
		By("install 2.3.x chart")
		Expect(helmInstall(namespace)).To(Succeed())

		By("verify CRDs are installed")
		Expect(crdExists("emqxes.apps.emqx.io")).To(BeTrue())
		Expect(crdExists("rebalances.apps.emqx.io")).To(BeTrue())

		By("verify operator deployment is available")
		Expect(Kubectl("wait", "deployment",
			"emqx-operator-controller-manager",
			"--for", "condition=Available",
			"--namespace", namespace,
			"--timeout", "1m",
		)).To(Succeed())
	})

	It("should install cleanly / pre-upgrade check disabled", func() {
		By("install 2.3.x with pre-upgrade check disabled")
		Expect(helmInstall(namespace, "--set", "upgrade.preUpgradeCheck=false")).
			To(Succeed())

		By("verify operator is running")
		Expect(Kubectl("wait", "deployment",
			"emqx-operator-controller-manager",
			"--for", "condition=Available",
			"--namespace", namespace,
			"--timeout", "1m",
		)).To(Succeed())

		By("verify CRDs are installed")
		Expect(crdExists("emqxes.apps.emqx.io")).To(BeTrue())
		Expect(crdExists("rebalances.apps.emqx.io")).To(BeTrue())
	})
})
