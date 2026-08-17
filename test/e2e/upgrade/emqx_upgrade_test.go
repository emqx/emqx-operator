package upgrade

import (
	"flag"
	"fmt"
	"testing"
	"time"

	. "github.com/emqx/emqx-operator/test/e2e"
	. "github.com/emqx/emqx-operator/test/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Initial EMQX image:
var emqxImageInitial string

// EMQX image to upgrade to:
var emqxImageUpgrade string

func init() {
	flag.StringVar(&emqxImageInitial, "emqx-image-initial", "", "Initial EMQX image to deploy")
	flag.StringVar(&emqxImageUpgrade, "emqx-image-upgrade", "", "EMQX image to upgrade to")
}

const (
	// projectImage is the name of the image which will be build and loaded
	// with the code source changes to be tested.
	projectImage = "emqx/emqx-operator:0.0.1"

	// namespace where the project is deployed in
	namespace = Namespace
)

func TestUpgrade(t *testing.T) {
	RegisterFailHandler(Fail)
	// Set the default timeout and interval for async assertions
	SetDefaultEventuallyTimeout(time.Minute * 5)
	SetDefaultEventuallyPollingInterval(time.Second * 3)
	// Run tests
	RunSpecs(t, "Upgrade")
}

var _ = BeforeSuite(func() {
	By("generate manifests")
	Expect(Run("make", "manifests")).To(Succeed())

	By("build emqx-operator docker image")
	Expect(Run("make", "docker-build-coverage",
		fmt.Sprintf("OPERATOR_IMAGE=%s", projectImage),
	)).To(Succeed())

	By("load emqx-operator docker image into kind cluster")
	Expect(LoadImageToKindClusterWithName(projectImage)).To(Succeed())
})

var _ = Describe("EMQX Upgrade", Ordered, func() {
	// Number of core and replicant replicas:
	var coreReplicas = 2
	var replicantReplicas = 2

	const emqxCRBasic = "test/e2e/files/resources/emqx.yaml"

	BeforeAll(func() {
		if emqxImageInitial == "" || emqxImageUpgrade == "" {
			Fail("Both `-emqx-image-initial` and `-emqx-image-upgrade` should be set")
		}

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
		emqxCR := SpecFromYAMLFile(emqxCRBasic).
			WithImage(emqxImageInitial).
			WithCores(coreReplicas).
			WithReplicants(replicantReplicas).
			WithDS().
			ToJSONDocument()
		Expect(KubectlStdin(emqxCR, "apply", "-f", "-")).To(Succeed())
		By("wait for EMQX cluster to be ready")
		Eventually(EMQXReady).Should(Succeed())
		Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
		Eventually(ReplicantsStable).WithArguments(replicantReplicas).Should(Succeed())
		Eventually(DSReplicationStable).WithArguments(coreReplicas).Should(Succeed())
		Eventually(DSReplicationHealthy).Should(Succeed())
	})

	It("upgrade EMQX version", func() {
		By("create client workload")
		Expect(Kubectl("apply", "-f", "test/e2e/files/resources/mqttx.yaml")).To(Succeed())
		defer Kubectl("delete", "-f", "test/e2e/files/resources/mqttx.yaml") // nolint:errcheck
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
		Eventually(EMQXReady).WithArguments(changingTime).Should(Succeed())
		Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
		Eventually(ReplicantsStable).WithArguments(replicantReplicas).Should(Succeed())
		Eventually(DSReplicationStable).WithArguments(coreReplicas).Should(Succeed())
		Eventually(DSReplicationHealthy).Should(Succeed())
	})

	AfterEach(func() {
		if CurrentSpecReport().Failed() {
			DumpDiagnosticReport(namespace, "emqx-upgrade", CurrentSpecReport().StartTime)
		}
	})

})
