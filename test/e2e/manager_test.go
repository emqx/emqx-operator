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

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/emqx/emqx-operator/test/util"
	. "github.com/emqx/emqx-operator/test/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var (
	// Optional Environment Variables:
	// - TEST_E2E_SKIP_PROMETHEUS_INSTALL=true:
	//   Skips Prometheus Operator installation during test setup.
	//   Useful if applications are already installed, avoiding re-installation and conflicts.
	skipPrometheusInstall = os.Getenv("TEST_E2E_SKIP_PROMETHEUS_INSTALL") == "true"
	isPrometheusInstalled = false
)

var _ = Describe("Manager", Ordered, func() {
	const (
		// Kustomize manifest defining Service Monitor
		serviceMonitorManifest = "test/e2e/files/prometheus"

		// serviceAccountName created for the project
		serviceAccountName = "emqx-operator-controller-manager"

		// metricsServiceName is the name of the metrics service of the project
		metricsServiceName = "emqx-operator-controller-manager-metrics-service"

		// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
		metricsRoleBindingName = "emqx-operator-metrics-binding"
	)

	// Before running the tests, set up the environment by creating the namespace,
	// installing CRDs, and deploying the controller.
	BeforeAll(func() {
		if !skipPrometheusInstall {
			By("install Prometheus Operator")
			Expect(util.InstallPrometheusOperator()).To(Succeed())
			isPrometheusInstalled = true
		}

		By("deploy EMQX Operator")
		Expect(Run("make", "deploy",
			fmt.Sprintf("OPERATOR_IMAGE=%s", projectImage),
			fmt.Sprintf("KUSTOMIZATION_FILE_PATH=%s", "test/e2e/files/manager"),
		)).To(Succeed())
		Expect(Kubectl("wait", "deployment", "emqx-operator-controller-manager",
			"--for", "condition=Available",
			"--namespace", namespace,
			"--timeout", "1m",
		)).To(Succeed(), "Timed out waiting for emqx-operator deployment")
	})

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		By("undeploy EMQX Operator")
		_ = Run("make", "undeploy")
		if !skipPrometheusInstall && isPrometheusInstalled {
			By("uninstall Prometheus Operator")
			util.UninstallPrometheusOperator()
		}
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		if CurrentSpecReport().Failed() {
			PrintDiagnosticReport(namespace)
		}
	})

	It("emqx-operator pod should run successfully", func() {
		Eventually(KubectlOut).WithArguments("get", "pods",
			"--namespace", namespace,
			"--selector", "control-plane=controller-manager",
			"-o", "json",
		).Should(
			BeUnmarshalledAs(&corev1.PodList{}, HaveField("Items", ConsistOf(
				And(
					HaveField("Name", ContainSubstring("controller-manager")),
					HaveField("Status.Phase", Equal(corev1.PodRunning)),
				)),
			)))
	})

	It("metrics endpoint should be serving metrics", func() {
		By("create ClusterRoleBinding for service account to allow access to metrics")
		Expect(Kubectl("create", "clusterrolebinding", metricsRoleBindingName,
			"--clusterrole=emqx-operator-metrics-reader",
			fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
		)).To(Succeed())
		defer Kubectl("delete", "clusterrolebinding", metricsRoleBindingName) // nolint:errcheck

		By("verify metrics service is available")
		Expect(Kubectl("get", "service", metricsServiceName, "--namespace", namespace)).To(Succeed())

		By("deploy Prometheus ServiceMonitor")
		Expect(Kubectl("apply", "-k", serviceMonitorManifest)).To(Succeed())
		Expect(Kubectl("get", "ServiceMonitor", "--namespace", namespace)).To(Succeed())
		defer Kubectl("delete", "-k", serviceMonitorManifest) // nolint:errcheck

		By("fetch service account token")
		token, err := serviceAccountToken(serviceAccountName)
		Expect(err).NotTo(HaveOccurred())
		Expect(token).NotTo(BeEmpty())

		By("wait for metrics endpoint to be ready")
		Eventually(KubectlOut).
			WithArguments("get", "endpoints", metricsServiceName, "--namespace", namespace).
			Should(ContainSubstring("8443"))

		By("create curl-metrics pod to access the metrics endpoint")
		Expect(Kubectl("run", "curl-metrics", "--restart=Never",
			"--namespace", namespace,
			"--image=curlimages/curl:7.78.0",
			"--", "/bin/sh", "-c", fmt.Sprintf(
				"curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics",
				token, metricsServiceName, namespace),
		)).To(Succeed())
		Eventually(KubectlOut).
			WithArguments("get", "pods", "curl-metrics", "--namespace", namespace,
				"-o", "jsonpath={.status.phase}",
			).
			Should(Equal("Succeeded"), "curl-metrics pod in wrong status")

		By("lookup expected metrics in curl-metrics logs")
		Expect(KubectlOut("logs", "curl-metrics", "--namespace", namespace)).
			To(ContainSubstring("controller_runtime_reconcile_total"))
	})

	// +kubebuilder:scaffold:e2e-webhooks-checks
})

// serviceAccountToken returns a token for the specified service account in the given namespace.
// It uses the Kubernetes TokenRequest API to generate a token by directly sending a request
// and parsing the resulting token from the API response.
func serviceAccountToken(serviceAccountName string) (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	// Temporary file to store the token request
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		// Execute kubectl command to create the token
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		// Parse the JSON output to extract the token
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}
