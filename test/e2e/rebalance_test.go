package e2e

import (
	"fmt"
	"os/exec"

	appsv2beta1 "github.com/emqx/emqx-operator/api/v2beta1"
	"github.com/emqx/emqx-operator/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Rebalance Test", Label("rebalance"), Ordered, func() {
	BeforeAll(func() {
		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command(
			"make", "deploy",
			fmt.Sprintf("IMG=%s", projectImage),
			fmt.Sprintf("KUSTOMIZATION_FILE_PATH=%s", "test/e2e/files/manager"),
		)
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")

		By("waiting for controller-manager deployment")
		cmd = exec.Command(
			"kubectl", "wait",
			"deployment", "emqx-operator-controller-manager",
			"--for", "condition=Available",
			"-n", namespace,
			"--timeout", "5m",
		)
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to wait for controller-manager deployment")
	})

	AfterAll(func() {
		By("undeploying the controller-manager")
		cmd := exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	Context("EMQX Rebalance", func() {
		AfterEach(func() {
			cmd := exec.Command("kubectl", "delete", "-f", "test/e2e/files/resources/emqx.yaml")
			_, _ = utils.Run(cmd)
			cmd = exec.Command("kubectl", "delete", "-f", "config/samples/apps_v2beta1_rebalance.yaml")
			_, _ = utils.Run(cmd)
		})

		It("EMQX is not found", func() {
			By("creating Rebalance CR")
			cmd := exec.Command("kubectl", "apply", "-f", "config/samples/apps_v2beta1_rebalance.yaml")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply rebalance.yaml")

			By("Rebalance will failed, because the EMQX is not found")

			By("checking Rebalance CR phase")
			cmd = exec.Command("kubectl", "get", "rebalance", "rebalance", "-o", "jsonpath={.status.phase}")
			out, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get rebalance status")
			Expect(out).To(BeEquivalentTo(appsv2beta1.RebalancePhaseFailed), "Rebalance should be failed")

			By("checking Rebalance CR state")
			cmd = exec.Command("kubectl", "get", "rebalance", "rebalance", "-o", "jsonpath={.status.rebalanceStates}")
			out, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get rebalance states")
			Expect(out).To(BeEmpty(), "Rebalance states should be empty")

			By("checking Rebalance CR conditions")
			cmd = exec.Command(
				"kubectl", "get", "rebalance", "rebalance",
				"-o", fmt.Sprintf("jsonpath={.status.conditions[?(@.type==\"%s\")].status}", appsv2beta1.RebalanceConditionFailed),
			)
			out, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get rebalance conditions")
			Expect(out).To(Equal("True"), "Rebalance condition should be true")
		})

		It("EMQX is not enterprise", func() {
			By("creating EMQX CR")
			cmd := exec.Command("kubectl", "apply", "-f", "test/e2e/files/resources/emqx-ce.yaml")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply emqx.yaml")

			By("creating Rebalance CR")
			cmd = exec.Command("kubectl", "apply", "-f", "config/samples/apps_v2beta1_rebalance.yaml")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply rebalance.yaml")

			By("Rebalance will failed, because the EMQX is not enterprise")

			By("checking Rebalance CR phase")
			cmd = exec.Command("kubectl", "get", "rebalance", "rebalance", "-o", "jsonpath={.status.phase}")
			out, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get rebalance status")
			Expect(out).To(BeEquivalentTo(appsv2beta1.RebalancePhaseFailed), "Rebalance should be failed")

			By("checking Rebalance CR state")
			cmd = exec.Command("kubectl", "get", "rebalance", "rebalance", "-o", "jsonpath={.status.rebalanceStates}")
			out, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get rebalance states")
			Expect(out).To(BeEmpty(), "Rebalance states should be empty")

			By("checking Rebalance CR conditions")
			cmd = exec.Command(
				"kubectl", "get", "rebalance", "rebalance",
				"-o", fmt.Sprintf(
					"jsonpath={.status.conditions[?(@.type==\"%s\")].status}",
					appsv2beta1.RebalanceConditionFailed,
				),
			)
			out, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get rebalance conditions")
			Expect(out).To(Equal("True"), "Rebalance condition should be true")
		})

		It("EMQX is exist", func() {
			By("creating EMQX CR")
			cmd := exec.Command("kubectl", "apply", "-f", "test/e2e/files/resources/emqx.yaml")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply emqx.yaml")

			By("creating Rebalance CR")
			cmd = exec.Command("kubectl", "apply", "-f", "config/samples/apps_v2beta1_rebalance.yaml")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply rebalance.yaml")

			By("checking Rebalance CR phase")
			cmd = exec.Command("kubectl", "get", "rebalance", "rebalance", "-o", "jsonpath={.status.phase}")
			out, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get rebalance status")
			Expect(out).To(BeEquivalentTo(appsv2beta1.RebalancePhaseFailed), "Rebalance should be failed")

			By("checking Rebalance CR state")
			cmd = exec.Command("kubectl", "get", "rebalance", "rebalance", "-o", "jsonpath={.status.rebalanceStates}")
			out, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get rebalance states")
			Expect(out).To(BeEmpty(), "Rebalance states should be empty")

			By("checking Rebalance CR conditions")
			cmd = exec.Command(
				"kubectl", "get", "rebalance", "rebalance",
				"-o", fmt.Sprintf("jsonpath={.status.conditions[?(@.type==\"%s\")].status}", appsv2beta1.RebalanceConditionFailed),
			)
			out, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get rebalance conditions")
			Expect(out).To(Equal("True"), "Rebalance condition should be true")
		})
	})
})
