package e2e

import (
	"fmt"

	appsv2beta1 "github.com/emqx/emqx-operator/api/v2beta1"
	. "github.com/emqx/emqx-operator/test/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
)

//nolint:errcheck
var _ = Describe("Rebalance Test", Label("rebalance"), Ordered, func() {
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
				HaveField("Phase", Equal(appsv2beta1.RebalancePhaseFailed)),
				HaveField("RebalanceStates", BeEmpty()),
				HaveRebalanceCondition(
					appsv2beta1.RebalanceConditionFailed,
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
			HaveField("Phase", Equal(appsv2beta1.RebalancePhaseFailed)),
			HaveField("RebalanceStates", BeEmpty()),
			HaveRebalanceCondition(
				appsv2beta1.RebalanceConditionFailed,
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
		Expect(Kubectl("wait", "deployment/mqttx",
			"--for=condition=Available",
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
			HaveField("Phase", Equal(appsv2beta1.RebalancePhaseProcessing)),
			HaveField("RebalanceStates", Not(BeEmpty())),
		))
		Eventually(RebalanceStatus).Should(And(
			HaveField("Phase", Equal(appsv2beta1.RebalancePhaseCompleted)),
			HaveRebalanceCondition(
				appsv2beta1.RebalanceConditionCompleted,
				HaveField("Status", Equal(corev1.ConditionTrue)),
			),
		))
	})
})

func RebalanceStatus(g Gomega) appsv2beta1.RebalanceStatus {
	var status appsv2beta1.RebalanceStatus
	out, err := KubectlOut("get", "rebalance", "rebalance", "-o", "jsonpath={.status}")
	g.Expect(err).NotTo(HaveOccurred(), "Failed to get rebalance status")
	g.Expect(out).To(UnmarshalInto(&status))
	return status
}

func HaveRebalanceCondition(
	conditionType appsv2beta1.RebalanceConditionType,
	matcher types.GomegaMatcher,
) types.GomegaMatcher {
	return WithTransform(
		func(s appsv2beta1.RebalanceStatus) *appsv2beta1.RebalanceCondition {
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
