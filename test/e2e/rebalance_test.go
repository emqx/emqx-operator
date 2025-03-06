package e2e

import (
	"fmt"
	"os/exec"

	appsv2beta1 "github.com/emqx/emqx-operator/api/v2beta1"
	"github.com/emqx/emqx-operator/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			specReport := CurrentSpecReport()
			if specReport.Failed() {
				By("Fetching controller manager pod logs")
				cmd := exec.Command(
					"kubectl", "logs",
					"-l", "control-plane=controller-manager",
					"-n", namespace,
				)
				controllerLogs, err := utils.Run(cmd)
				if err == nil {
					_, _ = fmt.Fprint(GinkgoWriter, "Controller logs:\n", controllerLogs)
				} else {
					_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
				}

				By("Fetching Kubernetes events in default namespace")
				cmd = exec.Command("kubectl", "get", "events", "--sort-by=.lastTimestamp")
				eventsOutput, err := utils.Run(cmd)
				if err == nil {
					_, _ = fmt.Fprint(GinkgoWriter, "Kubernetes events:\n", eventsOutput)
				} else {
					_, _ = fmt.Fprint(GinkgoWriter, "Failed to get Kubernetes events: ", err)
				}

				By("Checking Rebalance CR")
				cmd = exec.Command("kubectl", "get", "rebalance", "rebalance", "-o", "yaml")
				describeOut, err := utils.Run(cmd)
				if err == nil {
					_, _ = fmt.Fprint(GinkgoWriter, "Rebalance describe:\n", describeOut)
				} else {
					_, _ = fmt.Fprint(GinkgoWriter, "Failed to describe Rebalance: ", err)
				}

				By("Checking EMQX Pod logs")
				cmd = exec.Command("kubectl", "logs", "-l", "apps.emqx.io/instance=emqx")
				emqxLogs, err := utils.Run(cmd)
				if err == nil {
					_, _ = fmt.Fprint(GinkgoWriter, "EMQX logs:\n", emqxLogs)
				} else {
					_, _ = fmt.Fprint(GinkgoWriter, "Failed to get EMQX logs: ", err)
				}
			}

			cmd := exec.Command("kubectl", "delete", "emqx", "emqx")
			_, _ = utils.Run(cmd)
			cmd = exec.Command("kubectl", "delete", "-f", "test/e2e/files/resources/rebalance.yaml")
			_, _ = utils.Run(cmd)
		})

		It("EMQX is not found", func() {
			By("creating Rebalance CR")
			cmd := exec.Command("kubectl", "apply", "-f", "test/e2e/files/resources/rebalance.yaml")
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
			Expect(out).To(BeEquivalentTo(metav1.ConditionTrue), "Rebalance condition should be true")
		})

		It("EMQX is not enterprise", func() {
			By("creating EMQX CR")
			cmd := exec.Command("kubectl", "apply", "-f", "test/e2e/files/resources/emqx-ce.yaml")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply emqx.yaml")

			By("creating Rebalance CR")
			cmd = exec.Command("kubectl", "apply", "-f", "test/e2e/files/resources/rebalance.yaml")
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
			Expect(out).To(BeEquivalentTo(metav1.ConditionTrue), "Rebalance condition should be true")
		})

		It("EMQX is exist, but no connection should be rebalance", func() {
			By("creating EMQX CR")
			cmd := exec.Command("kubectl", "apply", "-f", "test/e2e/files/resources/emqx.yaml")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply emqx.yaml")
			Eventually(func() string {
				cmd := exec.Command(
					"kubectl", "get", "emqx", "emqx",
					"-o", "jsonpath={.status.conditions[?(@.type==\"Ready\")].status}",
				)
				out, _ := utils.Run(cmd)
				return out
			}).WithTimeout(timeout).WithPolling(interval).Should(BeEquivalentTo(metav1.ConditionTrue))

			By("creating Rebalance CR")
			cmd = exec.Command("kubectl", "apply", "-f", "test/e2e/files/resources/rebalance.yaml")
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
				"-o", fmt.Sprintf(
					"jsonpath={.status.conditions[?(@.type==\"%s\")].status}",
					appsv2beta1.RebalanceConditionFailed,
				),
			)
			out, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get rebalance conditions")
			Expect(out).To(BeEquivalentTo(metav1.ConditionTrue), "Rebalance condition should be true")
		})

		It("EMQX is exist, and connection should be rebalance", func() {
			By("creating EMQX CR")
			cmd := exec.Command("kubectl", "apply", "-f", "test/e2e/files/resources/emqx.yaml")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply emqx.yaml")

			Eventually(func() string {
				cmd := exec.Command(
					"kubectl", "get", "emqx", "emqx",
					"-o", "jsonpath={.status.conditions[?(@.type==\"Ready\")].status}",
				)
				out, _ := utils.Run(cmd)
				return out
			}).WithTimeout(timeout).WithPolling(interval).Should(BeEquivalentTo(metav1.ConditionTrue))
			Expect(err).NotTo(HaveOccurred(), "Failed to wait for emqx pods")

			By("creating MQTTX client")
			cmd = exec.Command("kubectl", "apply", "-f", "test/e2e/files/resources/mqttx.yaml")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply mqttx.yaml")
			cmd = exec.Command(
				"kubectl", "wait", "pod",
				"--selector=app=mqttx",
				"--for=condition=Ready",
				"--timeout=5m",
			)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to wait for mqttx pods")

			By("scaling EMQX replicas for create unbalance case")
			cmd = exec.Command(
				"kubectl", "patch", "emqx", "emqx",
				"--type", "json",
				"-p", `[{"op": "replace", "path": "/spec/coreTemplate/spec/replicas", "value": 3}]`,
			)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to scale emqx replicas")
			Eventually(func() string {
				cmd = exec.Command("kubectl", "get", "emqx", "emqx", "-o", "jsonpath={.status.coreNodesStatus.readyReplicas}")
				out, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred(), "Failed to get emqx status")
				return out
			}).WithTimeout(timeout).WithPolling(interval).Should(BeEquivalentTo("3"))
			Eventually(func() string {
				cmd := exec.Command(
					"kubectl", "get", "emqx", "emqx",
					"-o", "jsonpath={.status.conditions[?(@.type==\"Ready\")].status}",
				)
				out, _ := utils.Run(cmd)
				return out
			}).WithTimeout(timeout).WithPolling(interval).Should(BeEquivalentTo(metav1.ConditionTrue))
			Expect(err).NotTo(HaveOccurred(), "Failed to wait for emqx pods")

			By("creating Rebalance CR")
			cmd = exec.Command("kubectl", "apply", "-f", "test/e2e/files/resources/rebalance.yaml")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply rebalance.yaml")

			By("checking Rebalance CR state")
			Eventually(func() string {
				cmd = exec.Command("kubectl", "get", "rebalance", "rebalance", "-o", "jsonpath={.status.rebalanceStates}")
				out, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred(), "Failed to get rebalance states")
				return out
			}).WithTimeout(timeout).WithPolling(interval).ShouldNot(BeEmpty())

			By("checking Rebalance CR phase")
			Eventually(func() string {
				cmd = exec.Command("kubectl", "get", "rebalance", "rebalance", "-o", "jsonpath={.status.phase}")
				out, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred(), "Failed to get rebalance status")
				return out
			}).WithTimeout(timeout).WithPolling(interval).Should(BeEquivalentTo(appsv2beta1.RebalancePhaseProcessing))
			Eventually(func() string {
				cmd = exec.Command("kubectl", "get", "rebalance", "rebalance", "-o", "jsonpath={.status.phase}")
				out, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred(), "Failed to get rebalance status")
				return out
			}).WithTimeout(timeout).WithPolling(interval).Should(BeEquivalentTo(appsv2beta1.RebalancePhaseCompleted))

			By("checking Rebalance CR conditions")
			cmd = exec.Command(
				"kubectl", "get", "rebalance", "rebalance",
				"-o", fmt.Sprintf(
					"jsonpath={.status.conditions[?(@.type==\"%s\")].status}",
					appsv2beta1.RebalanceConditionCompleted,
				),
			)
			out, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get rebalance conditions")
			Expect(out).To(BeEquivalentTo(metav1.ConditionTrue), "Rebalance condition should be true")
		})
	})
})
