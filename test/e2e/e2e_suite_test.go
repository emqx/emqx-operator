/*
Copyright 2025-2026.

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
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"github.com/emqx/emqx-operator/test/util"
)

var (
	// Optional Environment Variables:
	// - PROMETHEUS_INSTALL_SKIP=true: Skips Prometheus Operator installation during test setup.
	// - CERT_MANAGER_INSTALL_SKIP=true: Skips CertManager installation during test setup.
	// These variables are useful if Prometheus or CertManager is already installed, avoiding
	// re-installation and conflicts.
	skipPrometheusInstall = os.Getenv("PROMETHEUS_INSTALL_SKIP") == "true"

	// projectImage is the name of the image which will be build and loaded
	// with the code source changes to be tested.
	projectImage = "emqx/emqx-operator:0.0.1"

	// diagnosticReportPath is the path to dump the diagnostic report on failure
	diagnosticReportPath = util.Env("TEST_E2E_DIAGNOSTIC_REPORT_PATH", "test/_reports")
)

var isPrometheusInstalled = false

// TestE2E runs the end-to-end (e2e) test suite for the project. These tests execute in an isolated,
// temporary environment to validate project changes with the the purposed to be used in CI jobs.
// The default setup requires Kind, builds/loads the Manager Docker image locally, and installs
// CertManager and Prometheus.
func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	_, _ = fmt.Fprintf(GinkgoWriter, "Starting emqx-operator integration test suite\n")
	RunSpecs(t, "e2e suite")
}

var _ = BeforeSuite(func() {
	// Set the default timeout and interval for async assertions
	SetDefaultEventuallyTimeout(time.Minute * 5)
	SetDefaultEventuallyPollingInterval(time.Second * 3)

	By("ensure Prometheus is enabled")
	_ = util.UncommentCode("config/default/kustomization.yaml", "#- ../prometheus", "#")

	By("generate files")
	Expect(util.Run("make", "generate")).To(Succeed())

	By("generate manifests")
	Expect(util.Run("make", "manifests")).To(Succeed())

	By("build emqx-operator docker image")
	Expect(util.Run("make", "docker-build-coverage",
		fmt.Sprintf("OPERATOR_IMAGE=%s", projectImage),
	)).To(Succeed())

	By("load emqx-operator docker image into kind cluster")
	Expect(util.LoadImageToKindClusterWithName(projectImage)).To(Succeed())

	By("install Metrics Server")
	Expect(util.InstallMetricsServer()).To(Succeed())

	// The tests-e2e are intended to run on a temporary cluster that is created and destroyed for testing.
	// To prevent errors when tests run in environments with Prometheus or CertManager already installed,
	// we check for their presence before execution.
	if !skipPrometheusInstall {
		By("install Prometheus Operator")
		Expect(util.InstallPrometheusOperator()).To(Succeed())
		isPrometheusInstalled = true
	}
})

var _ = AfterSuite(func() {
	util.UnnstallMetricsServer()
	if !skipPrometheusInstall && isPrometheusInstalled {
		By("uninstall Prometheus Operator")
		util.UninstallPrometheusOperator()
	}
})

func PrintDiagnosticReport(namespace string) {
	controllerLogs, err := util.KubectlOut("logs",
		"--selector", "control-plane=controller-manager",
		"--namespace", namespace,
		"--tail", "-1")
	if err == nil {
		GinkgoWriter.Print("Controller logs:\n", controllerLogs)
	} else {
		GinkgoWriter.Printf("Failed to get Controller logs: %s", err)
	}
	resources, err := util.KubectlOut("get", "all",
		"--selector", "apps.emqx.io/managed-by=emqx-operator")
	if err == nil {
		GinkgoWriter.Print("Managed EMQX resources:\n", resources)
	} else {
		GinkgoWriter.Printf("Failed to list managed resources: %s", err)
	}
	emqxCR, err := util.KubectlOut("get", "emqx", "emqx", "--output", "yaml")
	if err == nil {
		GinkgoWriter.Print("EMQX CR:\n", emqxCR)
	} else {
		GinkgoWriter.Printf("Failed to get EMQX CR: %s", err)
	}
	emqxLogs, err := util.KubectlOut("logs",
		"--selector", "apps.emqx.io/instance=emqx,apps.emqx.io/managed-by=emqx-operator")
	if err == nil {
		GinkgoWriter.Print("EMQX logs:\n", emqxLogs)
	} else {
		GinkgoWriter.Printf("Failed to get EMQX logs: %s", err)
	}
	eventsOutput, err := util.KubectlOut("get", "events", "--sort-by=.lastTimestamp")
	if err == nil {
		GinkgoWriter.Print("Kubernetes events:\n", eventsOutput)
	} else {
		GinkgoWriter.Printf("Failed to get Kubernetes events: %s", err)
	}
}

func DumpDiagnosticReport(namespace string, name string, time time.Time) {
	path := filepath.Join(diagnosticReportPath, name, time.Format("20060102150405"))
	err := os.MkdirAll(path, 0755)
	if err != nil {
		GinkgoWriter.Printf("Failed to create diagnostic report path: %s", err)
		return
	}

	GinkgoWriter.Println("Dumping diagnostic report: ", path)

	controllerLogs, err := util.KubectlOut("logs",
		"--selector", "control-plane=controller-manager",
		"--namespace", namespace,
		"--tail", "-1")
	if err == nil {
		_ = dumpString(path, "controller.log", controllerLogs)
	} else {
		GinkgoWriter.Println("Failed to get Controller logs:", err)
	}

	operatorManagedLabel := "apps.emqx.io/managed-by=emqx-operator"
	resources, err := util.KubectlOut("get", "all", "--selector", operatorManagedLabel)
	if err == nil {
		_ = dumpString(path, "resources", resources)
	} else {
		GinkgoWriter.Println("Failed to list managed resources:", err)
	}

	emqxCRs, err := util.KubectlOut("get", "emqx", "--output", "yaml")
	if err == nil {
		_ = dumpString(path, "emqx-crs.yaml", emqxCRs)
	} else {
		GinkgoWriter.Println("Failed to list EMQX CRs:", err)
	}

	emqxPods, err := util.KubectlOut("get", "pod", "--selector", operatorManagedLabel, "-o", "yaml")
	if err == nil {
		_ = dumpString(path, "emqx-pods.yaml", emqxPods)
	} else {
		GinkgoWriter.Println("Failed to list EMQX pods:", err)
	}

	emqxStatefulSets, err := util.KubectlOut("get", "statefulset", "--selector", operatorManagedLabel, "-o", "yaml")
	if err == nil {
		_ = dumpString(path, "emqx-statefulsets.yaml", emqxStatefulSets)
	} else {
		GinkgoWriter.Println("Failed to list EMQX StatefulSets:", err)
	}

	emqxReplicaSets, err := util.KubectlOut("get", "replicaset", "--selector", operatorManagedLabel, "-o", "yaml")
	if err == nil {
		_ = dumpString(path, "emqx-replicasets.yaml", emqxReplicaSets)
	} else {
		GinkgoWriter.Println("Failed to list EMQX ReplicaSets:", err)
	}

	var podList corev1.PodList
	err = json.Unmarshal(util.FromYAML([]byte(emqxPods)), &podList)
	if err != nil {
		GinkgoWriter.Println("Failed to unmarshal EMQX pods:", err)
	}
	for _, pod := range podList.Items {
		logs, err := util.KubectlOut("logs", pod.Name, "--tail", "-1")
		if err == nil {
			_ = dumpString(path, pod.Name+".log", logs)
		} else {
			GinkgoWriter.Println("Failed to get logs for pod: %s", pod.Name, err)
		}
	}

	dsInfo, _ := util.KubectlOut("exec", "service/emqx-listeners", "--", "emqx", "ctl", "ds", "info")
	if dsInfo != "" {
		_ = dumpString(path, "emqx.ctl.ds-info", dsInfo)
	}

	events, err := util.KubectlOut("get", "events", "--sort-by=.lastTimestamp")
	if err == nil {
		_ = dumpString(path, "events", events)
	} else {
		GinkgoWriter.Println("Failed to get Kubernetes events:", err)
	}
}

func dumpString(path string, name string, content string) error {
	return os.WriteFile(filepath.Join(path, name), []byte(content), 0644)
}
