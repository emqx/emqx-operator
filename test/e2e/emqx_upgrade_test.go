package e2e

import (
	"flag"
	"fmt"

	. "github.com/emqx/emqx-operator/test/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Number of core and replicant replicas:
var coreReplicas int = 2
var replicantReplicas int = 2

// Initial EMQX image:
var emqxImageInitial string

// EMQX image to upgrade to:
var emqxImageUpgrade string

func init() {
	flag.StringVar(&emqxImageInitial, "emqx-image-initial", "", "Initial EMQX image to deploy")
	flag.StringVar(&emqxImageUpgrade, "emqx-image-upgrade", "", "EMQX image to upgrade to")
}

//nolint:errcheck
var _ = Describe("EMQX Upgrade Test", Ordered, func() {

	const emqxCRBasic = "test/e2e/files/resources/emqx.yaml"

	BeforeAll(func() {
		if emqxImageInitial == "" || emqxImageUpgrade == "" {
			Skip("Both `-emqx-image-initial` and `-emqx-image-upgrade` should be set")
		}

		By("create manager namespace")
		Expect(Kubectl("create", "ns", namespace)).To(Succeed())

		By("install CRDs")
		Expect(Run("make", "install")).To(Succeed())

		By("deploy emqx-operator")
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
		By("undeploy emqx-operator")
		_ = Run("make", "undeploy")

		By("uninstall CRDs")
		_ = Run("make", "uninstall")

		By("delete manager namespace")
		_ = Kubectl("delete", "ns", namespace)
	})

	It("deploy cluster", func() {
		By("create EMQX cluster")
		emqxCR := PatchDocument(
			FromYAMLFile(emqxCRBasic),
			withImage(emqxImageInitial),
			withCores(coreReplicas),
			withReplicants(replicantReplicas),
			withConfig(configDS()),
		)
		Expect(KubectlStdin(emqxCR, "apply", "-f", "-")).To(Succeed())
		By("wait for EMQX cluster to be ready")
		Eventually(checkEMQXReady).Should(Succeed())
		Eventually(checkEMQXStatus).WithArguments(coreReplicas).Should(Succeed())
		Eventually(checkReplicantStatus).WithArguments(replicantReplicas).Should(Succeed())
		Eventually(checkDSReplicationStatus).WithArguments(coreReplicas).Should(Succeed())
		Eventually(checkDSReplicationHealthy).Should(Succeed())
	})

	It("upgrade EMQX version", func() {
		By("create client workload")
		Expect(Kubectl("apply", "-f", "test/e2e/files/resources/mqttx.yaml")).To(Succeed())
		defer Kubectl("delete", "-f", "test/e2e/files/resources/mqttx.yaml")
		Expect(Kubectl("wait", "pod",
			"--selector=app=mqttx",
			"--for=condition=Ready",
			"--timeout=1m",
		)).To(Succeed(), "Timed out waiting for MQTTX to be ready")

		By("change EMQX image")
		changingTime := metav1.Now()
		Expect(Kubectl("patch", "emqx", "emqx",
			"--type", "json",
			"--patch", `[{"op": "replace", "path": "/spec/image", "value": "`+emqxImageUpgrade+`"}]`)).
			To(Succeed())

		By("wait for EMQX cluster to be ready again")
		Eventually(checkEMQXReady).WithArguments(changingTime).Should(Succeed())
		Eventually(checkEMQXStatus).WithArguments(coreReplicas).Should(Succeed())
		Eventually(checkReplicantStatus).WithArguments(replicantReplicas).Should(Succeed())
		Eventually(checkDSReplicationStatus).WithArguments(coreReplicas).Should(Succeed())
		Eventually(checkDSReplicationHealthy).Should(Succeed())
	})

	AfterEach(func() {
		if CurrentSpecReport().Failed() {
			PrintDiagnosticReport(namespace)
		}
	})

})
