package e2e

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"

	appsv2beta1 "github.com/emqx/emqx-operator/api/v2beta1"
	"github.com/emqx/emqx-operator/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("E2E Test", Label("base"), Ordered, func() {
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
				GinkgoWriter.Print("Controller logs:\n", controllerLogs)
			} else {
				GinkgoWriter.Printf("Failed to get Controller logs: %s", err)
			}

			By("Listing managed EMQX resources")
			resources, err := kubectlOut("get", "all", "--selector", appsv2beta1.LabelsManagedByKey+"=emqx-operator")
			if err == nil {
				GinkgoWriter.Print("Managed EMQX resources:\n", resources)
			} else {
				GinkgoWriter.Printf("Failed to list managed resources: %s", err)
			}

			By("Checking EMQX CR")
			cmd = exec.Command("kubectl", "get", "emqx", "emqx", "-o", "yaml")
			emqxCR, err := utils.Run(cmd)
			if err == nil {
				GinkgoWriter.Print("EMQX CR:\n", emqxCR)
			} else {
				GinkgoWriter.Printf("Failed to get EMQX CR: %s", err)
			}

			By("Fetching EMQX pod logs")
			cmd = exec.Command(
				"kubectl", "logs",
				"-l", "apps.emqx.io/instance=emqx,apps.emqx.io/managed-by=emqx-operator",
			)
			emqxLogs, err := utils.Run(cmd)
			if err == nil {
				GinkgoWriter.Print("EMQX logs:\n", emqxLogs)
			} else {
				GinkgoWriter.Printf("Failed to get EMQX logs: %s", err)
			}

			By("Fetching Kubernetes events in default namespace")
			cmd = exec.Command("kubectl", "get", "events", "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				GinkgoWriter.Print("Kubernetes events:\n", eventsOutput)
			} else {
				GinkgoWriter.Printf("Failed to get Kubernetes events: %s", err)
			}
		}
	})

	Context("EMQX Cluster", func() {
		var coreReplicas int = 2
		It("deploy EMQX cluster without replicant node", func() {
			By("creating EMQX cluster")
			cmd := exec.Command("kubectl", "apply", "-f", "test/e2e/files/resources/emqx.yaml")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply emqx.yaml")

			verifyEMQXStatus(coreReplicas, nil)
			verifyNoReplicants()
		})

		It("scale EMQX cluster without replicant node", func() {
			By("scaling up EMQX cluster")
			coreReplicas = 3
			changingTime := metav1.Now()
			cmd := exec.Command(
				"kubectl", "patch", "emqx", "emqx",
				"--type", "json",
				"-p", `[{"op": "replace", "path": "/spec/coreTemplate/spec/replicas", "value": 3}]`,
			)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to scale emqx cluster")

			verifyEMQXStatus(coreReplicas, &changingTime)
			verifyNoReplicants()
		})

		It("change EMQX image for target blue-green update", func() {
			By("creating MQTTX client")
			cmd := exec.Command("kubectl", "apply", "-f", "test/e2e/files/resources/mqttx.yaml")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply mqttx.yaml")
			cmd = exec.Command(
				"kubectl", "wait", "pod",
				"--selector=app=mqttx",
				"--for=condition=Ready",
				"--timeout=5m",
			)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to wait for mqttx pods")

			By("getting storage StatefulSet")
			cmd = exec.Command("kubectl", "get", "emqx", "emqx", "-o", "jsonpath={.status.coreNodesStatus.currentRevision}")
			out, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get emqx status")

			cmd = exec.Command("kubectl", "get", "statefulset", "-l", appsv2beta1.LabelsPodTemplateHashKey+"="+out, "-o", "json")
			out, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get statefulset")

			stsList := &appsv1.StatefulSetList{}
			_ = json.Unmarshal([]byte(out), &stsList)
			storageSts := &stsList.Items[0]

			By("changing EMQX image")
			changingTime := metav1.Now()
			cmd = exec.Command(
				"kubectl", "patch", "emqx", "emqx",
				"--type", "json",
				"-p", `[{"op": "replace", "path": "/spec/image", "value": "emqx/emqx-enterprise:latest-elixir"}]`,
			)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to patch emqx cluster")

			By("checking the EMQX cluster node evacuations status")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "emqx", "emqx", "-o", "jsonpath={.status.nodeEvacuationsStatus}")
				out, _ := utils.Run(cmd)
				return out
			}).WithTimeout(timeout).WithPolling(interval).ShouldNot(ContainSubstring("connection_eviction_rate"))

			By(("deleting old storage statefulSet pods by hands, so that can be running faster"))
			cmd = exec.Command("kubectl", "get", "sts", storageSts.Name, "-o", "jsonpath={.status.currentRevision}")
			storageStsCurrentRevision, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get statefulset currentRevision")
			cmd = exec.Command(
				"kubectl", "delete", "pod",
				"-l", "controller-revision-hash="+storageStsCurrentRevision,
			)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete storage pods")

			verifyEMQXStatus(coreReplicas, &changingTime)
			verifyNoReplicants()

			By("checking the storage StatefulSet has been scaled down to 0")
			cmd = exec.Command("kubectl", "get", "statefulset", storageSts.Name, "-o", "jsonpath={.status.replicas}")
			out, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get statefulset replicas")
			Expect(out).To(Equal("0"), "storage StatefulSet replicas is not 0")

			By("deleting MQTTX client workload")
			Expect(kubectl("delete", "-f", "test/e2e/files/resources/mqttx.yaml")).
				To(Succeed(), "Failed to delete MQTTX workload")
		})

		It("delete EMQX cluster without replicant node", func() {
			By("deleting EMQX cluster")
			cmd := exec.Command("kubectl", "delete", "emqx", "emqx")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete emqx cluster")

			By("checking the EMQX cluster has been deleted")
			cmd = exec.Command("kubectl", "get", "emqx", "emqx")
			_, err = utils.Run(cmd)
			Expect(err).To(HaveOccurred(), "EMQX cluster is not deleted")
		})
	})

	Context("EMQX Cluster with replicant Node", func() {
		var coreReplicas int = 2
		var replicantReplicas int = 2
		It("deploy EMQX cluster with replicant node", func() {
			By("creating EMQX cluster")
			cmd := exec.Command("kubectl", "apply", "-f", "test/e2e/files/resources/emqx.yaml")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply emqx.yaml")

			cmd = exec.Command(
				"kubectl", "patch", "emqx", "emqx",
				"--type", "json",
				"-p", `[{"op": "replace", "path": "/spec/replicantTemplate", "value": {"spec": {"replicas": 2}}}]`,
			)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to patch emqx cluster")

			verifyEMQXStatus(coreReplicas, nil)
			verifyReplicantStatus(replicantReplicas)
		})

		It("scale EMQX cluster with replicant node", func() {
			By("scaling up EMQX cluster")
			coreReplicas = 2
			replicantReplicas = 3
			changingTime := metav1.Now()
			cmd := exec.Command(
				"kubectl", "patch", "emqx", "emqx",
				"--type", "json",
				"-p", `[{"op": "replace", "path": "/spec/coreTemplate/spec/replicas", "value": 2}]`,
			)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to scale emqx cluster")

			cmd = exec.Command(
				"kubectl", "patch", "emqx", "emqx",
				"--type", "json",
				"-p", `[{"op": "replace", "path": "/spec/replicantTemplate/spec/replicas", "value": 3}]`,
			)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to scale emqx cluster")

			verifyEMQXStatus(coreReplicas, &changingTime)
			verifyReplicantStatus(replicantReplicas)
		})

		It("change EMQX image for target blue-green update", func() {
			By("creating MQTTX client")
			cmd := exec.Command("kubectl", "apply", "-f", "test/e2e/files/resources/mqttx.yaml")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply mqttx.yaml")
			cmd = exec.Command(
				"kubectl", "wait", "pod",
				"--selector=app=mqttx",
				"--for=condition=Ready",
				"--timeout=5m",
			)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to wait for mqttx pods")

			By("getting storage StatefulSet")
			cmd = exec.Command("kubectl", "get", "emqx", "emqx", "-o", "jsonpath={.status.coreNodesStatus.currentRevision}")
			out, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get emqx status")

			cmd = exec.Command("kubectl", "get", "statefulset", "-l", appsv2beta1.LabelsPodTemplateHashKey+"="+out, "-o", "json")
			out, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get statefulset")

			stsList := &appsv1.StatefulSetList{}
			_ = json.Unmarshal([]byte(out), &stsList)
			storageSts := &stsList.Items[0]

			By("getting storage ReplicaSet")
			cmd = exec.Command("kubectl", "get", "emqx", "emqx", "-o", "jsonpath={.status.replicantNodesStatus.currentRevision}")
			out, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get emqx status")

			cmd = exec.Command("kubectl", "get", "replicaset", "-l", appsv2beta1.LabelsPodTemplateHashKey+"="+out, "-o", "json")
			out, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get replicaset")

			rsList := &appsv1.ReplicaSetList{}
			_ = json.Unmarshal([]byte(out), &rsList)
			storageRs := &rsList.Items[0]

			By("changing EMQX image")
			changingTime := metav1.Now()
			cmd = exec.Command(
				"kubectl", "patch", "emqx", "emqx",
				"--type", "json",
				"-p", `[{"op": "replace", "path": "/spec/image", "value": "emqx/emqx-enterprise:latest-elixir"}]`,
			)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to patch emqx cluster")

			By("checking the EMQX cluster node evacuations status")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", "emqx", "emqx", "-o", "jsonpath={.status.nodeEvacuationsStatus}")
				out, _ := utils.Run(cmd)
				return out
			}).WithTimeout(timeout).WithPolling(interval).ShouldNot(ContainSubstring("connection_eviction_rate"))

			verifyEMQXStatus(coreReplicas, &changingTime)
			verifyReplicantStatus(replicantReplicas)

			By("checking the storage StatefulSet has been scaled down to 0")
			cmd = exec.Command("kubectl", "get", "statefulset", storageSts.Name, "-o", "jsonpath={.status.replicas}")
			out, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get statefulset replicas")
			Expect(out).To(Equal("0"), "storage StatefulSet replicas is not 0")

			By("checking the storage ReplicaSet has been scaled down to 0")
			cmd = exec.Command("kubectl", "get", "replicaset", storageRs.Name, "-o", "jsonpath={.status.replicas}")
			out, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get replicaset replicas")
			Expect(out).To(Equal("0"), "storage ReplicaSet replicas is not 0")

			By("deleting MQTTX client workload")
			Expect(kubectl("delete", "-f", "test/e2e/files/resources/mqttx.yaml")).
				To(Succeed(), "Failed to delete MQTTX workload")
		})

		It("delete EMQX cluster with replicant node", func() {
			By("deleting EMQX cluster")
			cmd := exec.Command("kubectl", "delete", "emqx", "emqx")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete emqx cluster")

			By("checking the EMQX cluster has been deleted")
			cmd = exec.Command("kubectl", "get", "emqx", "emqx")
			_, err = utils.Run(cmd)
			Expect(err).To(HaveOccurred(), "EMQX cluster is not deleted")
		})
	})

	Context("EMQX Core-Replicant Cluster / DS Enabled", func() {
		var coreReplicas int = 2
		var replicantReplicas int = 2

		It("deploy core-replicant EMQX cluster", func() {
			By("creating EMQX cluster")
			Expect(kubectl("apply", "-f", "test/e2e/files/resources/emqx-ds.yaml")).
				To(Succeed(), "Failed to apply emqx.yaml")

			verifyEMQXStatus(coreReplicas, nil)
			verifyReplicantStatus(replicantReplicas)

			By("checking DS replication status")
			verifyDSReplicationStatus(coreReplicas)

			By("checking pods have relevant conditions")
			pods := &corev1.PodList{}
			Expect(kubectlOut("get", "pods", "-l", appsv2beta1.LabelsManagedByKey+"=emqx-operator", "-o", "json")).
				To(utils.UnmarshalInto(&pods), "Failed to get pods")
			Expect(pods.Items).To(HaveLen(4), "EMQX cluster does not have 4 pods")
			for _, pod := range pods.Items {
				if pod.Labels[appsv2beta1.LabelsDBRoleKey] == "core" {
					Expect(pod.Status.Conditions).To(ContainElement(And(
						HaveField("Type", Equal(appsv2beta1.DSReplicationSite)),
						HaveField("Status", Equal(corev1.ConditionTrue)),
					)))
				}
			}
			for _, pod := range pods.Items {
				if pod.Labels[appsv2beta1.LabelsDBRoleKey] == "replicant" {
					Expect(pod.Status.Conditions).To(ContainElement(And(
						HaveField("Type", Equal(appsv2beta1.DSReplicationSite)),
						HaveField("Status", Equal(corev1.ConditionFalse)),
					)))
				}
			}
		})

		It("scale up core EMQX cluster", func() {
			By("scaling up EMQX cluster")
			coreReplicas = 4
			changingTime := metav1.Now()
			Expect(kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"-p", `[{"op": "replace", "path": "/spec/coreTemplate/spec/replicas", "value": 4}]`,
			)).To(Succeed(), "Failed to scale EMQX cluster")

			verifyEMQXStatus(coreReplicas, &changingTime)
			verifyReplicantStatus(replicantReplicas)

			By("checking DS replicated across 4 core nodes")
			verifyDSReplicationStatus(coreReplicas)
		})

		It("scale down core EMQX cluster", func() {
			By("scaling down EMQX cluster")
			coreReplicas = 2
			changingTime := metav1.Now()
			Expect(kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"-p", `[{"op": "replace", "path": "/spec/coreTemplate/spec/replicas", "value": 2}]`,
			)).To(Succeed(), "Failed to scale EMQX cluster")

			verifyEMQXStatus(coreReplicas, &changingTime)
			verifyReplicantStatus(replicantReplicas)

			By("checking DS replicated across 2 core nodes")
			verifyDSReplicationStatus(coreReplicas)
		})

		It("perform a blue-green update", func() {
			By("looking up existing StatefulSet")
			hashLabel := appsv2beta1.LabelsPodTemplateHashKey
			coreRev, err := kubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.coreNodesStatus.currentRevision}")
			Expect(err).NotTo(HaveOccurred(), "Failed to get EMQX status")

			stsList := &appsv1.StatefulSetList{}
			Expect(kubectlOut("get", "statefulset", "-l", hashLabel+"="+coreRev, "-o", "json")).
				To(utils.UnmarshalInto(&stsList), "Failed to list statefulSets")
			storageSts := &stsList.Items[0]

			By("changing EMQX image + number of replicas")
			coreReplicas = 2
			changingTime := metav1.Now()
			Expect(kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"-p", `[
					{"op": "replace", "path": "/spec/image", "value": "emqx/emqx-enterprise:latest-elixir"},
					{"op": "replace", "path": "/spec/coreTemplate/spec/replicas", "value": 2}
				]`,
			)).To(Succeed(), "Failed to patch EMQX cluster")

			By("checking the new StatefulSet is spinning up")
			EventuallySoon(func(g Gomega) {
				g.Expect(kubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.coreNodesStatus.updateRevision}")).
					NotTo(Equal(coreRev), "New StatefulSet has not been spun up")
			}).Should(Succeed())

			verifyEMQXStatus(coreReplicas, &changingTime)
			verifyReplicantStatus(replicantReplicas)

			By("checking the storage StatefulSet has been scaled down to 0")
			Expect(kubectlOut("get", "statefulset", storageSts.Name, "-o", "jsonpath={.status.replicas}")).
				To(Equal("0"), "storage StatefulSet replicas is not 0")

			By("checking DS replication status")
			verifyDSReplicationStatus(coreReplicas)
		})

		It("disable DS", func() {
			By("disabling DS")
			coreReplicas = 1
			changingTime := metav1.Now()
			Expect(kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"-p", `[
					{"op": "replace", "path": "/spec/image", "value": "emqx/emqx-enterprise:latest"},
					{"op": "replace", "path": "/spec/coreTemplate/spec/replicas", "value": 1},
					{"op": "replace", "path": "/spec/config/data", "value": ""}
				]`,
			)).To(Succeed(), "Failed to patch EMQX cluster")

			verifyEMQXStatus(coreReplicas, &changingTime)
			verifyReplicantStatus(replicantReplicas)

			By("checking DS is non-functional on core nodes")
			pods := &corev1.PodList{}
			Expect(kubectlOut("get", "pods", "-l", appsv2beta1.LabelsManagedByKey+"=emqx-operator", "-o", "json")).
				To(utils.UnmarshalInto(&pods), "Failed to get pods")
			for _, pod := range pods.Items {
				if pod.Labels[appsv2beta1.LabelsDBRoleKey] == "core" {
					Expect(pod.Status.Conditions).To(ContainElement(And(
						HaveField("Type", Equal(appsv2beta1.DSReplicationSite)),
						HaveField("Status", Equal(corev1.ConditionFalse)),
					)))
				}
			}

			// TODO
			// Process of disabling is not super smooth currently. There's no way to wipe knowledge of DS DBs
			// from the cluster, even those that are irreversibly lost.
			// By("checking DS has been turned off smoothly")
			// Expect(kubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.dsReplication}")).
			// 	To(Equal("{}"), "DS replication status is not empty")
			// Expect(kubectlOut("exec", "service/emqx-listeners", "--", "emqx", "ctl", "ds", "info")).
			// 	NotTo(ContainSubstring("(!)"))
		})

		It("delete core-replicant EMQX cluster", func() {
			By("deleting EMQX cluster")
			cmd := exec.Command("kubectl", "delete", "emqx", "emqx")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete emqx cluster")

			By("checking the EMQX cluster has been deleted")
			cmd = exec.Command("kubectl", "get", "emqx", "emqx")
			_, err = utils.Run(cmd)
			Expect(err).To(HaveOccurred(), "EMQX cluster is not deleted")
		})

	})
})

func verifyEMQXStatus(coreReplicas int, afterTime *metav1.Time) {
	By("checking the EMQX cluster status has been ready")
	Eventually(func() bool {
		cmd := exec.Command("kubectl", "get", "emqx", "emqx", "-o", "jsonpath={.status.conditions[?(@.type==\"Ready\")]}")
		out, _ := utils.Run(cmd)

		cond := &metav1.Condition{}
		_ = json.Unmarshal([]byte(out), &cond)

		if afterTime == nil {
			return cond.Status == metav1.ConditionTrue
		}
		return cond.Status == metav1.ConditionTrue && cond.LastTransitionTime.After(afterTime.Time)
	}).WithTimeout(timeout).WithPolling(interval).Should(BeTrue())

	By("checking all of the EMQX pods being ready")
	EventuallySoon(func() (string, error) {
		return kubectlOut("get", "pod",
			"--selector", "apps.emqx.io/instance=emqx,apps.emqx.io/managed-by=emqx-operator",
			"-o", "go-template="+
				`{{ range $pod := .items }}{{ range .status.conditions }}`+
				`{{ if (and (eq .type "Ready") (eq .status "False")) }}`+
				`{{ $pod.metadata.name }}{{ "\n" }}`+
				`{{ end }}`+
				`{{ end }}{{ end }}`,
		)
	}).Should(Equal(""))

	By("checking the EMQX cluster core node status has current replicas")
	verifyEMQXStatus := func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "emqx", "emqx", "-o", "jsonpath={.status.coreNodesStatus.replicas}")
		out, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to get emqx status")
		Expect(out).To(
			Equal(strconv.Itoa(coreReplicas)),
			"EMQX cluster does not have "+strconv.Itoa(coreReplicas)+" core nodes",
		)

		cmd = exec.Command("kubectl", "get", "emqx", "emqx", "-o", "jsonpath={.status.coreNodesStatus.readyReplicas}")
		out, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to get emqx status")
		Expect(out).To(
			Equal(strconv.Itoa(coreReplicas)),
			"EMQX cluster does not have "+strconv.Itoa(coreReplicas)+" core nodes",
		)

		cmd = exec.Command("kubectl", "get", "emqx", "emqx", "-o", "jsonpath={.status.coreNodesStatus.currentReplicas}")
		out, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to get emqx status")
		Expect(out).To(
			Equal(strconv.Itoa(coreReplicas)),
			"EMQX cluster does not have "+strconv.Itoa(coreReplicas)+" core nodes",
		)

		cmd = exec.Command("kubectl", "get", "emqx", "emqx", "-o", "jsonpath={.status.coreNodesStatus.updateReplicas}")
		out, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to get emqx status")
		Expect(out).To(
			Equal(strconv.Itoa(coreReplicas)),
			"EMQX cluster does not have "+strconv.Itoa(coreReplicas)+" core nodes",
		)
	}
	Eventually(verifyEMQXStatus).Should(Succeed())

	By("checking the EMQX cluster core node status has current revision")
	verifyEMQXStatus = func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "emqx", "emqx", "-o", "jsonpath={.status.coreNodesStatus.currentRevision}")
		currentRevision, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to get emqx status")
		Expect(currentRevision).NotTo(Equal(""), "EMQX cluster does not have current revision")

		cmd = exec.Command("kubectl", "get", "emqx", "emqx", "-o", "jsonpath={.status.coreNodesStatus.updateRevision}")
		updateRevision, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to get emqx status")
		Expect(updateRevision).NotTo(Equal(""), "EMQX cluster does not have update revision")

		Expect(currentRevision).To(Equal(updateRevision), "EMQX cluster current revision is not equal to update revision")

		cmd = exec.Command(
			"kubectl", "get", "pods", "-l", appsv2beta1.LabelsPodTemplateHashKey+"="+currentRevision, "-o", "json",
		)
		out, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to get pods")

		pods := &corev1.PodList{}
		_ = json.Unmarshal([]byte(out), &pods)
		Expect(pods.Items).To(HaveLen(coreReplicas), "EMQX cluster does not have 2 pods with current revision")
	}
	Eventually(verifyEMQXStatus).Should(Succeed())
}

func verifyNoReplicants() {
	By("checking the EMQX cluster replicant node status is nil")
	Expect(kubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.replicantNodesStatus}")).
		To(Equal("{}"), "EMQX cluster replicant node status is not nil")
	Expect(kubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.replicantNodes}")).
		To(BeEmpty(), "EMQX cluster replicant nodes is not nil")
}

func verifyReplicantStatus(replicantReplicas int) {
	By("checking the EMQX cluster replicant node status has current replicas")
	verifyEMQXStatus := func(g Gomega) {
		Expect(kubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.replicantNodesStatus.replicas}")).
			To(Equal(strconv.Itoa(replicantReplicas)),
				"EMQX cluster does not have %d replicant nodes",
				replicantReplicas,
			)
		Expect(kubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.replicantNodesStatus.readyReplicas}")).
			To(Equal(strconv.Itoa(replicantReplicas)),
				"EMQX cluster does not have %d ready replicant nodes",
				replicantReplicas,
			)
		Expect(kubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.replicantNodesStatus.currentReplicas}")).
			To(Equal(strconv.Itoa(replicantReplicas)),
				"EMQX cluster does not have %d replicant nodes running current revision",
				replicantReplicas,
			)
		Expect(kubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.replicantNodesStatus.updateReplicas}")).
			To(Equal(strconv.Itoa(replicantReplicas)),
				"EMQX cluster does not have %d replicant nodes running update revision",
				replicantReplicas,
			)
	}
	Eventually(verifyEMQXStatus).Should(Succeed())

	By("checking the EMQX cluster replicant node status has current revision")
	verifyEMQXStatus = func(g Gomega) {
		currentRevision, err := kubectlOut(
			"get", "emqx", "emqx", "-o", "jsonpath={.status.replicantNodesStatus.currentRevision}",
		)
		Expect(err).NotTo(HaveOccurred(), "Failed to get emqx status")
		Expect(currentRevision).NotTo(Equal(""), "EMQX cluster does not have current revision")

		updateRevision, err := kubectlOut(
			"get", "emqx", "emqx", "-o", "jsonpath={.status.replicantNodesStatus.updateRevision}",
		)
		Expect(err).NotTo(HaveOccurred(), "Failed to get emqx status")
		Expect(updateRevision).NotTo(Equal(""), "EMQX cluster does not have update revision")

		Expect(currentRevision).To(Equal(updateRevision), "EMQX cluster current revision is not equal to update revision")

		pods := &corev1.PodList{}
		Expect(kubectlOut("get", "pods",
			"-l", appsv2beta1.LabelsPodTemplateHashKey+"="+currentRevision,
			"--field-selector", "status.phase==Running",
			"-o", "json",
		)).To(utils.UnmarshalInto(&pods), "Failed to get pods")
		Expect(pods.Items).To(
			HaveLen(replicantReplicas),
			"EMQX cluster does not have %d pods with current revision",
			replicantReplicas,
		)
	}
	Eventually(verifyEMQXStatus).Should(Succeed())
}

func verifyDSReplicationStatus(coreReplicas int) {
	EventuallySoon(func(g Gomega) {
		status := &appsv2beta1.DSReplicationStatus{}
		g.Expect(kubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.dsReplication}")).
			To(utils.UnmarshalInto(&status), "Failed to get emqx status")
		g.Expect(status.DBs).To(HaveLen(1), "Only 'messages' DB is expected to be present")
		g.Expect(status.DBs[0]).To(And(
			HaveField("Name", Equal("messages")),
			HaveField("NumShards", BeEquivalentTo(8)),
			HaveField("NumShardReplicas", BeEquivalentTo(8*min(3, coreReplicas))),
			HaveField("LostShardReplicas", BeEquivalentTo(0)),
			HaveField("NumTransitions", BeEquivalentTo(0)),
			HaveField("MinReplicas", BeEquivalentTo(min(3, coreReplicas))),
			HaveField("MaxReplicas", BeEquivalentTo(min(3, coreReplicas))),
		))
	}).Should(Succeed())
	Expect(kubectlOut("exec", "service/emqx-listeners", "--", "emqx", "ctl", "ds", "info")).
		NotTo(ContainSubstring("(!)"))
}

func kubectl(args ...string) error {
	cmd := exec.Command("kubectl", args...)
	_, err := utils.Run(cmd)
	return err
}

func kubectlOut(args ...string) (string, error) {
	cmd := exec.Command("kubectl", args...)
	return utils.Run(cmd)
}
