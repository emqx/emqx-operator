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

// helmInstall installs the local 3.x chart into the given namespace with --wait.
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

// helmUpgrade upgrades to the local 3.x chart in the given namespace with --wait.
// func helmUpgrade(namespace string) error {
// 	return Run("helm", "upgrade",
// 		helmReleaseName,
// 		localChartPath,
// 		"--namespace", namespace,
// 		"--set", "image.repository="+operatorImageRepo,
// 		"--set", "image.tag="+operatorImageTag,
// 		"--set", "image.pullPolicy=Never",
// 		"--wait",
// 		"--timeout", "2m",
// 	)
// }

// helmCleanup removes all resources that 3.x chart may have left behind in
// the given namespace, including the Helm release itself.
func helmCleanup(namespace string) {
	_ = Run("helm", "uninstall", helmReleaseName, "--namespace", namespace)
	_ = Kubectl("delete", "clusterrole", "emqx-operator-manager-role")
	_ = Kubectl("delete", "clusterrolebinding", "emqx-operator-manager-rolebinding")
	_ = Kubectl("delete", "clusterrole", "emqx-operator-pre-upgrade")
	_ = Kubectl("delete", "clusterrolebinding", "emqx-operator-pre-upgrade")
	_ = Kubectl("delete", "crd", "emqxes.apps.emqx.io")
	_ = Kubectl("delete", "crd", "rebalances.apps.emqx.io")
	_ = Kubectl("delete", "ns", namespace)
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

	It("should install cleanly", func() {
		By("install 3.x chart")
		Expect(helmInstall(namespace)).To(Succeed())

		By("verify CRDs are installed")
		Expect(crdExists("emqxes.apps.emqx.io")).To(BeTrue())
		// NOTE: Rebalance controller is disabled in this release. See api/v2beta1/rebalance_types.go.
		// Expect(crdExists("rebalances.apps.emqx.io")).To(BeTrue())

		By("verify operator deployment is available")
		Expect(Kubectl("wait", "deployment",
			"emqx-operator-controller-manager",
			"--for", "condition=Available",
			"--namespace", namespace,
			"--timeout", "1m",
		)).To(Succeed())

		By("verify operator logs do not show errors")
		Consistently(KubectlOut, "20s", "5s").
			WithArguments("logs", "-n", namespace,
				"deployment/emqx-operator-controller-manager", "--tail=50",
			).ShouldNot(Or(
			ContainSubstring("ERROR"),
			ContainSubstring("Error"),
		))
	})

	It("should install cleanly / one watch namespace", func() {
		By("install 3.x chart with one watch namespace")
		Expect(helmInstall(namespace,
			"--set-json", `watchNamespaces=["emqx-operator-fresh"]`,
		)).To(Succeed())

		By("verify CRDs are installed")
		Expect(crdExists("emqxes.apps.emqx.io")).To(BeTrue())
		// NOTE: Rebalance controller is disabled in this release. See api/v2beta1/rebalance_types.go.
		// Expect(crdExists("rebalances.apps.emqx.io")).To(BeTrue())

		By("verify operator deployment is available")
		Expect(Kubectl("wait", "deployment",
			"emqx-operator-controller-manager",
			"--for", "condition=Available",
			"--namespace", namespace,
			"--timeout", "1m",
		)).To(Succeed())

		By("verify no ClusterRoles exists for the manager")
		Expect(Kubectl("get", "role", "emqx-operator-manager-role",
			"-n", namespace)).To(Succeed())
		Expect(Kubectl("get", "clusterrole", "emqx-operator-manager-role")).NotTo(Succeed())

		By("verify operator logs do not show errors")
		Consistently(KubectlOut, "30s", "5s").
			WithArguments("logs", "-n", namespace,
				"deployment/emqx-operator-controller-manager", "--tail=50",
			).ShouldNot(Or(
			ContainSubstring("ERROR"),
			ContainSubstring("Error"),
		))
	})

	It("should install cleanly / several watch namespaces", func() {
		watchNamespaces := []string{
			"emqx-operator-watch-a",
			"emqx-operator-watch-b",
		}
		DeferCleanup(func() {
			for _, ns := range watchNamespaces {
				_ = Kubectl("delete", "ns", ns, "--ignore-not-found")
			}
		})

		By("create watched namespaces")
		for _, ns := range watchNamespaces {
			_ = Kubectl("delete", "ns", ns, "--ignore-not-found")
			Expect(Kubectl("create", "ns", ns)).To(Succeed())
		}

		By("install 3.x chart with watchNamespaces")
		Expect(helmInstall(namespace,
			"--set-json", `watchNamespaces=["emqx-operator-fresh","emqx-operator-watch-a","emqx-operator-watch-b"]`,
		)).To(Succeed())

		By("verify operator deployment is available")
		Expect(Kubectl("wait", "deployment",
			"emqx-operator-controller-manager",
			"--for", "condition=Available",
			"--namespace", namespace,
			"--timeout", "1m",
		)).To(Succeed())

		By("verify CRDs are installed")
		Expect(crdExists("emqxes.apps.emqx.io")).To(BeTrue())
		// NOTE: Rebalance controller is disabled in this release. See api/v2beta1/rebalance_types.go.
		// Expect(crdExists("rebalances.apps.emqx.io")).To(BeTrue())

		By("verify no ClusterRoles exists for the manager")
		Expect(resourceExists("clusterrole", "emqx-operator-manager-role")).To(BeFalse())
		Expect(resourceExists("clusterrolebinding", "emqx-operator-manager-rolebinding")).To(BeFalse())

		By("verify effective watch namespaces have manager RBAC")
		Expect(Kubectl("get", "role", "emqx-operator-manager-role",
			"-n", namespace)).To(Succeed())
		Expect(Kubectl("get", "rolebinding", "emqx-operator-manager-rolebinding",
			"-n", namespace)).To(Succeed())
		for _, ns := range watchNamespaces {
			Expect(Kubectl("get", "role", "emqx-operator-manager-role",
				"-n", ns)).To(Succeed())
			Expect(Kubectl("get", "rolebinding", "emqx-operator-manager-rolebinding",
				"-n", ns)).To(Succeed())
		}

		By("verify operator logs do not show errors")
		Consistently(KubectlOut, "30s", "5s").
			WithArguments("logs", "-n", namespace,
				"deployment/emqx-operator-controller-manager", "--tail=50",
			).ShouldNot(Or(
			ContainSubstring("ERROR"),
			ContainSubstring("Error"),
		))
	})
})
