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

package e2e

import (
	"fmt"

	. "github.com/emqx/emqx-operator/test/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//nolint:errcheck
var _ = Describe("EMQX 5.9.0 to 5.10.1 Upgrade", Label("emqx", "regression"), Ordered, func() {
	const (
		emqxImage510      = "emqx/emqx:5.10.0"
		emqxImage510Patch = "emqx/emqx:5.10.1"
		coreReplicas      = 2
		initialEMQXCR     = `
			apiVersion: apps.emqx.io/v3beta1
			kind: EMQX
			metadata:
			  name: emqx
			spec:
			  image: emqx/emqx:5.9.0
			  imagePullPolicy: IfNotPresent
			  config:
			    mode: Merge
			    data: |
			      license { key = "evaluation" }
			      mqtt.max_packet_size = 200MB
			  updateStrategy:
			    evacuationStrategy:
			      type: Disabled
			  coreTemplate:
			    spec:
			      replicas: 2
			  replicantTemplate:
			    spec:
			      replicas: 2
		`
	)

	var replicantReplicas = 2

	BeforeAll(func() {
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

	AfterAll(func() {
		By("undeploy EMQX Operator")
		_ = Run("make", "undeploy")
	})

	AfterEach(func() {
		DumpDiagnosticReport(namespace, "emqx-590-upgrade", CurrentSpecReport().StartTime)
		if CurrentSpecReport().Failed() {
			PrintDiagnosticReport(namespace)
		}
	})

	It("deploys EMQX 5.9.0 with two core and two replicant nodes", func() {
		By("create EMQX cluster with node evacuation disabled")
		emqxCR := SpecFromYAML(initialEMQXCR).ToJSONDocument()
		Expect(KubectlStdin(emqxCR, "apply", "-f", "-")).To(Succeed())

		By("wait for EMQX 5.9.0 cluster to be ready")
		Eventually(EMQXReady).Should(Succeed())
		Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
		Eventually(ReplicantsStable).WithArguments(replicantReplicas).Should(Succeed())
	})

	It("upgrades EMQX from 5.9.0 to 5.10.0", func() {
		changedAt := metav1.Now()
		Expect(Kubectl("patch", "emqx", "emqx",
			"--type", "json",
			"--patch", `[{"op": "replace", "path": "/spec/image", "value": "`+emqxImage510+`"}]`,
		)).To(Succeed())

		By("wait for EMQX 5.10.0 cluster to be ready")
		Eventually(EMQXReady).WithArguments(changedAt).Should(Succeed())
		Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
		Eventually(ReplicantsStable).WithArguments(replicantReplicas).Should(Succeed())
	})

	It("upgrades EMQX to 5.10.1 while scaling replicants from two to three", func() {
		replicantReplicas = 3
		changedAt := metav1.Now()
		Expect(Kubectl("patch", "emqx", "emqx",
			"--type", "json",
			"--patch", `[
				{"op": "replace", "path": "/spec/image", "value": "`+emqxImage510Patch+`"},
				{"op": "replace", "path": "/spec/replicantTemplate/spec/replicas", "value": 3}
			]`,
		)).To(Succeed())

		By("wait for the upgraded and scaled cluster to be ready")
		Eventually(EMQXReady).WithArguments(changedAt).Should(Succeed())
		Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
		Eventually(ReplicantsStable).WithArguments(replicantReplicas).Should(Succeed())
	})

	It("deletes the cluster", func() {
		Expect(Kubectl("delete", "emqx", "emqx")).To(Succeed())
		Expect(Kubectl("get", "emqx", "emqx")).To(HaveOccurred(), "EMQX cluster still exists")
	})
})
