package e2e

import (
	"fmt"

	crd "github.com/emqx/emqx-operator/api/v2beta1"
	. "github.com/emqx/emqx-operator/test/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
)

// NOTE: Rebalance controller is disabled in this release. See api/v2beta1/rebalance_types.go.

//nolint:errcheck
var _ = Describe("Rebalance Test", Label("rebalance"), Ordered, Pending, func() {

	BeforeAll(func() {
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
			"--timeout", "5m",
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

	AfterEach(func() {
		if CurrentSpecReport().Failed() {
			PrintDiagnosticReport(namespace)
		}
	})

	It("No EMQX exists", func() {
		By("create Rebalance CR")
		Expect(Kubectl("apply", "-f", "test/e2e/files/resources/rebalance.yaml")).To(Succeed())
		defer Kubectl("delete", "-f", "test/e2e/files/resources/rebalance.yaml")
		By("wait for Rebalance to be failed")
		Eventually(RebalanceStatus).Should(
			And(
				HaveField("Phase", Equal(crd.RebalancePhaseFailed)),
				HaveField("RebalanceStates", BeEmpty()),
				HaveRebalanceCondition(
					crd.RebalanceConditionFailed,
					HaveField("Status", Equal(corev1.ConditionTrue)),
				),
			),
		)
	})

	It("EMQX exists / nothing to rebalance", func() {
		By("create EMQX CR")
		Expect(Kubectl("apply", "-f", "test/e2e/files/resources/emqx.yaml")).To(Succeed())
		defer Kubectl("delete", "-f", "test/e2e/files/resources/emqx.yaml")
		By("wait for EMQX to be ready")
		Eventually(checkEMQXReady).Should(Succeed())

		By("create Rebalance CR")
		Expect(Kubectl("apply", "-f", "test/e2e/files/resources/rebalance.yaml")).
			To(Succeed(), "Failed to apply resources/rebalance.yaml")
		defer Kubectl("delete", "-f", "test/e2e/files/resources/rebalance.yaml")

		By("wait for Rebalance to become failed")
		Eventually(RebalanceStatus).Should(And(
			HaveField("Phase", Equal(crd.RebalancePhaseFailed)),
			HaveField("RebalanceStates", BeEmpty()),
			HaveRebalanceCondition(
				crd.RebalanceConditionFailed,
				HaveField("Status", Equal(corev1.ConditionTrue)),
			),
		))
	})

	It("EMQX exists / connections should be rebalanced", func() {
		By("create EMQX CR")
		Expect(Kubectl("apply", "-f", "test/e2e/files/resources/emqx.yaml")).To(Succeed())
		defer Kubectl("delete", "emqx", "emqx")
		Eventually(checkEMQXReady).Should(Succeed())

		By("create MQTTX client workload")
		Expect(Kubectl("apply", "-f", "test/e2e/files/resources/mqttx.yaml")).To(Succeed())
		defer Kubectl("delete", "-f", "test/e2e/files/resources/mqttx.yaml")
		Expect(Kubectl("wait", "pod",
			"--selector=app=mqttx",
			"--for=condition=Ready",
			"--timeout=1m",
		)).To(Succeed(), "Timed out waiting for MQTTX to be ready")

		By("scale up EMQX cluster to introduce imbalance")
		Expect(Kubectl("patch", "emqx", "emqx",
			"--type", "json",
			"--patch", `[{"op": "replace", "path": "/spec/coreTemplate/spec/replicas", "value": 3}]`,
		)).To(Succeed())

		By("wait for EMQX to be ready after scaling")
		Eventually(checkEMQXReady).Should(Succeed())
		Eventually(checkEMQXStatus).WithArguments(3).Should(Succeed())

		By("create Rebalance CR")
		Expect(Kubectl("apply", "-f", "test/e2e/files/resources/rebalance.yaml")).To(Succeed())
		defer Kubectl("delete", "-f", "test/e2e/files/resources/rebalance.yaml")

		By("check Rebalance CR state")
		Eventually(RebalanceStatus).Should(And(
			HaveField("Phase", Equal(crd.RebalancePhaseProcessing)),
			HaveField("RebalanceStates", Not(BeEmpty())),
		))
		Eventually(RebalanceStatus).Should(And(
			HaveField("Phase", Equal(crd.RebalancePhaseCompleted)),
			HaveRebalanceCondition(
				crd.RebalanceConditionCompleted,
				HaveField("Status", Equal(corev1.ConditionTrue)),
			),
		))
	})
})

func RebalanceStatus(g Gomega) crd.RebalanceStatus {
	var status crd.RebalanceStatus
	out, err := KubectlOut("get", "rebalance", "rebalance", "-o", "jsonpath={.status}")
	g.Expect(err).NotTo(HaveOccurred(), "Failed to get rebalance status")
	g.Expect(out).To(UnmarshalInto(&status))
	return status
}

func HaveRebalanceCondition(
	conditionType crd.RebalanceConditionType,
	matcher types.GomegaMatcher,
) types.GomegaMatcher {
	return WithTransform(
		func(s crd.RebalanceStatus) *crd.RebalanceCondition {
			for _, c := range s.Conditions {
				if c.Type == conditionType {
					return &c
				}
			}
			return nil
		},
		matcher,
	)
}
