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
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/emqx/emqx-operator/test/util"
)

const (
	// helmReleaseName is the Helm release name used across all Helm upgrade tests.
	helmReleaseName = "emqx-operator"

	// localChartPath is the path to the local 3.x chart being tested.
	localChartPath = "deploy/charts/emqx-operator"

	// operatorImage is the operator image which should already be built and available.
	operatorImageRepo = "emqx/emqx-operator"
	operatorImageTag  = "0.0.1-helm"
	operatorImage     = operatorImageRepo + ":" + operatorImageTag
)

func TestHelmE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	_, _ = fmt.Fprintf(GinkgoWriter, "Starting Helm e2e test suite\n")
	RunSpecs(t, "Helm e2e suite")
}

var _ = BeforeSuite(func() {
	SetDefaultEventuallyTimeout(3 * time.Minute)
	SetDefaultEventuallyPollingInterval(3 * time.Second)

	By("add emqx Helm repository")
	Expect(Run("helm", "repo", "add", "emqx", "https://repos.emqx.io/charts")).To(Succeed())
	Expect(Run("helm", "repo", "update")).To(Succeed())

	By("generate Helm chart files")
	Expect(Run("make", "helm")).To(Succeed())

	By("build emqx-operator docker image")
	Expect(Run("make", "docker-build",
		fmt.Sprintf("OPERATOR_IMAGE=%s", operatorImage),
	)).To(Succeed())

	By("load operator image into Kind cluster")
	Expect(LoadImageToKindClusterWithName(operatorImage)).To(Succeed())
})

// crdExists checks whether a CRD exists in the cluster.
func crdExists(name string) bool {
	return Kubectl("get", "crd", name) == nil
}

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
