package e2e

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	crdv2 "github.com/emqx/emqx-operator/api/v2"
	. "github.com/emqx/emqx-operator/test/util"
	"github.com/lithammer/dedent"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func withCores(numReplicas int) []byte {
	return fmt.Appendf(nil,
		`{"spec": {"coreTemplate": {"spec": {"replicas": %d}}}}`,
		numReplicas,
	)
}

func withReplicants(numReplicas int) []byte {
	return fmt.Appendf(nil,
		`{"spec": {"replicantTemplate": {"spec": {"replicas": %d}}}}`,
		numReplicas,
	)
}

func withImage(image string) []byte {
	return fmt.Appendf(nil, `{"spec": {"image": "%s"}}`, image)
}

func withConfig(snippets ...string) []byte {
	defaults := []string{configLicense(), configConsoleLog("info")}
	config := slices.Concat(defaults, snippets)
	return fmt.Appendf(nil, `{"spec": {"config": {"data": %s}}}`, intoJsonString(config...))
}

func intoJsonString(snippets ...string) []byte {
	configStr := dedent.Dedent(strings.Join(snippets, ""))
	jsonStr, _ := json.Marshal(configStr)
	return jsonStr
}

func configLicense() string {
	return `
		license { key = "evaluation" }
	`
}

func configConsoleLog(level string) string {
	return `
		log.console { level = "` + level + `" }
	`
}

func configDS() string {
	return `
		durable_sessions { enable = true }
		durable_storage { 
			messages {
				backend = builtin_raft
				n_shards = 8
			}
		}
	`
}

//nolint:unparam
func configListener(ty string, name string, enabled bool, bind string) string {
	return fmt.Sprintf(`
		listeners.%s.%s {
			enabled = %t
			bind = "%s"
		}
	`, ty, name, enabled, bind)
}

//nolint:errcheck
var _ = Describe("EMQX Test", Label("emqx"), Ordered, func() {

	const (
		emqxCRBasic      = "test/e2e/files/resources/emqx.yaml"
		emqxImage        = "emqx/emqx:5.10.0"
		emqxImageUpgrade = "emqx/emqx:5.10.1"
	)

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

	AfterEach(func() {
		if CurrentSpecReport().Failed() {
			PrintDiagnosticReport(namespace)
		}
	})

	Context("EMQX Cluster", func() {
		// Initial number of core replicas:
		var coreReplicas int = 2

		It("deploy cluster", func() {
			By("create EMQX cluster")
			emqxCR := PatchDocument(
				FromYAMLFile(emqxCRBasic),
				withImage(emqxImage),
				withCores(coreReplicas),
				withConfig(),
			)
			Expect(KubectlStdin(emqxCR, "apply", "-f", "-")).To(Succeed())
			By("wait for EMQX cluster to be ready")
			Eventually(checkEMQXReady).Should(Succeed())
			Eventually(checkEMQXStatus).WithArguments(coreReplicas).Should(Succeed())
			checkNoReplicants(Default)
		})

		It("scale cluster up", func() {
			coreReplicas = 3
			scaleupStartedAt := metav1.Now()
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[{"op": "replace", "path": "/spec/coreTemplate/spec/replicas", "value": 3}]`,
			)).To(Succeed(), "Failed to scale up EMQX cluster")
			Eventually(checkEMQXReady).WithArguments(scaleupStartedAt).Should(Succeed())
			Eventually(checkEMQXStatus).WithArguments(coreReplicas).Should(Succeed())
			checkNoReplicants(Default)
		})

		It("change image to trigger blue-green update", func() {
			By("create MQTTX client")
			Expect(Kubectl("apply", "-f", "test/e2e/files/resources/mqttx.yaml")).To(Succeed())
			defer Kubectl("delete", "-f", "test/e2e/files/resources/mqttx.yaml")
			Expect(Kubectl("wait", "pod",
				"--selector=app=mqttx",
				"--for=condition=Ready",
				"--timeout=1m",
			)).To(Succeed(), "Timed out waiting MQTTX to be ready")

			By("fetch current core StatefulSet")
			var stsList appsv1.StatefulSetList
			coreRev, err := KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.coreNodesStatus.currentRevision}")
			Expect(err).NotTo(HaveOccurred(), "Failed to get EMQX status")
			Expect(KubectlOut("get", "statefulset",
				"--selector", crdv2.LabelPodTemplateHash+"="+coreRev,
				"-o", "json",
			)).To(UnmarshalInto(&stsList), "Failed to list statefulsets")
			Expect(stsList.Items).To(HaveLen(1))

			By("change EMQX image")
			changedAt := metav1.Now()
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[{"op": "replace", "path": "/spec/image", "value": "`+emqxImageUpgrade+`"}]`)).
				To(Succeed())

			By("check EMQX cluster node evacuations status")
			Eventually(KubectlOut).
				WithArguments("get", "emqx", "emqx", "-o", "jsonpath={.status.nodeEvacuationsStatus}").
				ShouldNot(ContainSubstring("connection_eviction_rate"))

			Eventually(checkEMQXReady).WithArguments(changedAt).Should(Succeed())
			Eventually(checkEMQXStatus).WithArguments(coreReplicas).Should(Succeed())
			checkNoReplicants(Default)

			By("check previous core StatefulSet has been scaled down to 0")
			out, err := KubectlOut("get", "statefulset", stsList.Items[0].Name, "-o", "jsonpath={.status.replicas}")
			Expect(err).NotTo(HaveOccurred(), "Failed to get core StatefulSet replicas")
			Expect(out).To(Equal("0"))
		})

		It("change config", func() {
			By("change EMQX config")
			configChange := string(intoJsonString(
				// Change listener ports:
				configListener("tcp", "default", true, "11883"),
				configListener("quic", "default", true, "14567"),
				configListener("ws", "default", false, "0"),
				configListener("wss", "default", false, "0"),
				// And also change dashboard config, should be skipped:
				"dashboard.listeners.http { bind = 28083, num_acceptors = 1 }",
			))
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[{"op": "replace", "path": "/spec/config/data", "value": `+configChange+`}]`)).
				To(Succeed())
			By("wait for EMQX cluster to be ready")
			Eventually(checkEMQXReady).Should(Succeed())
			By("wait for services to be updated")
			var servicePorts []corev1.ServicePort
			Eventually(KubectlOut).WithArguments("get", "service", "emqx-listeners", "-o", "jsonpath={.spec.ports}").
				Should(BeUnmarshalledAs(&servicePorts, ConsistOf(
					And(
						HaveField("Name", Equal("tcp-default")),
						HaveField("Port", Equal(int32(11883))),
						HaveField("Protocol", Equal(corev1.ProtocolTCP)),
					),
					And(
						HaveField("Name", Equal("ssl-default")),
						HaveField("Port", Equal(int32(8883))),
						HaveField("Protocol", Equal(corev1.ProtocolTCP)),
					),
					And(
						HaveField("Name", Equal("quic-default")),
						HaveField("Port", Equal(int32(14567))),
						HaveField("Protocol", Equal(corev1.ProtocolUDP)),
					),
				)))
		})

		It("delete cluster", func() {
			Expect(Kubectl("delete", "emqx", "emqx")).To(Succeed())
			Expect(Kubectl("get", "emqx", "emqx")).To(HaveOccurred(), "EMQX cluster still exists")
		})
	})

	Context("EMQX Cluster / Botched Blue-Green Updates", func() {
		// Initial number of core replicas:
		var coreReplicas int = 2

		It("deploy cluster", func() {
			By("create EMQX cluster")
			emqxCR := PatchDocument(
				FromYAMLFile(emqxCRBasic),
				withImage(emqxImage),
				withCores(coreReplicas),
				withConfig(),
			)
			Expect(KubectlStdin(emqxCR, "apply", "-f", "-")).To(Succeed())
			By("wait for EMQX cluster to be ready")
			Eventually(checkEMQXReady).Should(Succeed())
			Eventually(checkEMQXStatus).WithArguments(coreReplicas).Should(Succeed())
		})

		It("trigger botched blue-green updates", func() {
			By("create MQTT workload")
			Expect(Kubectl("apply", "-f", "test/e2e/files/resources/mqttx.yaml")).To(Succeed())
			defer Kubectl("delete", "-f", "test/e2e/files/resources/mqttx.yaml")
			Expect(Kubectl("wait", "pod",
				"--selector=app=mqttx",
				"--for=condition=Ready",
				"--timeout=1m",
			)).To(Succeed(), "Timed out waiting MQTTX to be ready")

			By("decrease revision history limit")
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[
					{"op": "replace", "path": "/spec/revisionHistoryLimit", "value": 1}
				]`)).
				To(Succeed())

			By("lookup initial EMQX status")
			var statusInitial crdv2.CoreNodesStatus
			Eventually(checkEMQXReady).Should(Succeed())
			Expect(KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.coreNodesStatus}")).
				To(UnmarshalInto(&statusInitial))

			By("specify incorrect EMQX image")
			coreReplicas = 1
			changedAt1 := metav1.Now()
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[
					{"op": "replace", "path": "/spec/coreTemplate/spec/replicas", "value": 1},
					{"op": "replace", "path": "/spec/image", "value": "emqx/emqx:5.Y.ZZZ"}
				]`)).
				To(Succeed())
			Consistently(checkEMQXReady, "30s", "3s").WithArguments(changedAt1).Should(Not(Succeed()))

			By("specify broken EMQX config")
			changedAt2 := metav1.Now()
			configBroken := string(intoJsonString("broker { no.such.config { k = v } }"))
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[
					{"op": "replace", "path": "/spec/config/data", "value": `+configBroken+`},
					{"op": "replace", "path": "/spec/image", "value": "`+emqxImageUpgrade+`"},
				]`)).
				To(Succeed())
			Consistently(checkEMQXReady, "30s", "3s").WithArguments(changedAt2).Should(Not(Succeed()))

			By("verify core nodes are still intact")
			var status crdv2.CoreNodesStatus
			Expect(KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.coreNodesStatus}")).
				To(BeUnmarshalledAs(&status, And(
					HaveField("Replicas", Equal(statusInitial.Replicas)),
					HaveField("ReadyReplicas", Equal(statusInitial.ReadyReplicas)),
				)))

			By("specify correct EMQX config")
			changedAt3 := metav1.Now()
			imageForceChange := "docker.io/" + emqxImageUpgrade
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[
					{"op": "replace", "path": "/spec/config/data", "value": ""},
					{"op": "replace", "path": "/spec/image", "value": "`+imageForceChange+`"},
				]`)).
				To(Succeed())
			Eventually(checkEMQXReady).WithArguments(changedAt3).Should(Succeed())
			Eventually(checkEMQXStatus).WithArguments(coreReplicas).Should(Succeed())
		})

		It("delete cluster", func() {
			Expect(Kubectl("delete", "emqx", "emqx")).To(Succeed())
		})
	})

	Context("EMQX Core-Replicant Cluster", func() {
		// Initial number of core and replicant replicas:
		var coreReplicas int = 1
		var replicantReplicas int = 2

		It("deploy cluster", func() {
			By("create EMQX cluster")
			emqxCR := PatchDocument(
				FromYAMLFile(emqxCRBasic),
				withImage(emqxImage),
				withCores(coreReplicas),
				withReplicants(replicantReplicas),
				withConfig(),
			)
			Expect(KubectlStdin(emqxCR, "apply", "-f", "-")).To(Succeed())
			By("wait for EMQX cluster to be ready")
			Eventually(checkEMQXReady).Should(Succeed())
			Eventually(checkEMQXStatus).WithArguments(coreReplicas).Should(Succeed())
			Eventually(checkReplicantStatus).WithArguments(replicantReplicas).Should(Succeed())
		})

		It("scale cluster up", func() {
			coreReplicas = 2
			replicantReplicas = 3
			scaleupStartedAt := metav1.Now()
			By("change number of core replicas")
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[{"op": "replace", "path": "/spec/coreTemplate/spec/replicas", "value": 2}]`)).
				To(Succeed(), "Failed to scale emqx cluster")
			By("change number of replicant replicas")
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[{"op": "replace", "path": "/spec/replicantTemplate/spec/replicas", "value": 3}]`)).
				To(Succeed(), "Failed to scale emqx cluster")
			By("wait for EMQX cluster to be ready after scaling")
			Eventually(checkEMQXReady).WithArguments(scaleupStartedAt).Should(Succeed())
			Eventually(checkEMQXStatus).WithArguments(coreReplicas).Should(Succeed())
			Eventually(checkReplicantStatus).WithArguments(replicantReplicas).Should(Succeed())
		})

		It("change image for target blue-green update", func() {
			By("create MQTTX client")
			Expect(Kubectl("apply", "-f", "test/e2e/files/resources/mqttx.yaml")).To(Succeed())
			defer Kubectl("delete", "-f", "test/e2e/files/resources/mqttx.yaml")
			Expect(Kubectl("wait", "pod",
				"--selector=app=mqttx",
				"--for=condition=Ready",
				"--timeout=1m",
			)).To(Succeed(), "Timed out waiting for MQTTX to be ready")

			By("fetch current core StatefulSet")
			var stsList appsv1.StatefulSetList
			coreRev, err := KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.coreNodesStatus.currentRevision}")
			Expect(err).NotTo(HaveOccurred(), "Failed to get EMQX status")
			Expect(KubectlOut("get", "statefulset",
				"--selector", crdv2.LabelPodTemplateHash+"="+coreRev,
				"-o", "json",
			)).To(UnmarshalInto(&stsList), "Failed to list statefulsets")
			Expect(stsList.Items).To(HaveLen(1))

			By("fetch current replicant ReplicaSet")
			var rsList appsv1.ReplicaSetList
			replRev, err := KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.replicantNodesStatus.currentRevision}")
			Expect(err).NotTo(HaveOccurred(), "Failed to get EMQX status")
			Expect(KubectlOut("get", "replicaset",
				"--selector", crdv2.LabelPodTemplateHash+"="+replRev,
				"-o", "json",
			)).To(UnmarshalInto(&rsList), "Failed to list replicasets")
			Expect(rsList.Items).To(HaveLen(1))

			By("change EMQX image")
			changingTime := metav1.Now()
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[{"op": "replace", "path": "/spec/image", "value": "`+emqxImageUpgrade+`"}]`,
			)).To(Succeed())

			By("check EMQX cluster node evacuations status")
			Eventually(KubectlOut).
				WithArguments("get", "emqx", "emqx", "-o", "jsonpath={.status.nodeEvacuationsStatus}").
				ShouldNot(ContainSubstring("connection_eviction_rate"))

			By("wait for EMQX cluster to be ready again")
			Eventually(checkEMQXReady).WithArguments(changingTime).Should(Succeed())
			Eventually(checkEMQXStatus).WithArguments(coreReplicas).Should(Succeed())
			Eventually(checkReplicantStatus).WithArguments(replicantReplicas).Should(Succeed())

			By("check previous coreSet has been scaled down to 0")
			out, err := KubectlOut("get", "statefulset", stsList.Items[0].Name, "-o", "jsonpath={.status.replicas}")
			Expect(err).NotTo(HaveOccurred(), "Failed to get core StatefulSet")
			Expect(out).To(Equal("0"))

			By("check previous replicantSet has been scaled down to 0")
			out, err = KubectlOut("get", "replicaset", rsList.Items[0].Name, "-o", "jsonpath={.status.replicas}")
			Expect(err).NotTo(HaveOccurred(), "Failed to get replicant ReplicaSet")
			Expect(out).To(Equal("0"))
		})

		It("delete cluster", func() {
			Expect(Kubectl("delete", "emqx", "emqx")).To(Succeed())
			Expect(Kubectl("get", "emqx", "emqx")).To(HaveOccurred(), "EMQX cluster still exists")
		})
	})

	Context("EMQX Core-Replicant DS-Enabled Cluster", func() {
		// Initial number of core and replicant replicas:
		var coreReplicas int = 2
		var replicantReplicas int = 2

		It("deploy core-replicant EMQX cluster", func() {
			By("create EMQX cluster")
			emqxCR := PatchDocument(
				FromYAMLFile(emqxCRBasic),
				withImage(emqxImage),
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

			By("verify EMQX pods have relevant conditions")
			var pods corev1.PodList
			Expect(KubectlOut("get", "pods",
				"--selector", crdv2.LabelManagedBy+"=emqx-operator",
				"-o", "json",
			)).To(UnmarshalInto(&pods), "Failed to list EMQX pods")
			Expect(pods.Items).To(HaveLen(4), "EMQX cluster does not have 4 pods")
			for _, pod := range pods.Items {
				if pod.Labels[crdv2.LabelDBRole] == "core" {
					Expect(pod.Status.Conditions).To(ContainElement(And(
						HaveField("Type", Equal(crdv2.DSReplicationSite)),
						HaveField("Status", Equal(corev1.ConditionTrue)),
					)))
				}
				if pod.Labels[crdv2.LabelDBRole] == "replicant" {
					Expect(pod.Status.Conditions).To(ContainElement(And(
						HaveField("Type", Equal(crdv2.DSReplicationSite)),
						HaveField("Status", Equal(corev1.ConditionFalse)),
					)))
				}
			}
		})

		It("scale up core EMQX cluster", func() {
			coreReplicas = 4
			scaleStartedAt := metav1.Now()
			By("change number of core replicas")
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[{"op": "replace", "path": "/spec/coreTemplate/spec/replicas", "value": 4}]`,
			)).To(Succeed())
			By("wait for EMQX cluster to be ready after scaling")
			Eventually(checkEMQXReady).WithArguments(scaleStartedAt).Should(Succeed())
			Eventually(checkEMQXStatus).WithArguments(coreReplicas).Should(Succeed())
			Eventually(checkReplicantStatus).WithArguments(replicantReplicas).Should(Succeed())
			Eventually(checkDSReplicationStatus).WithArguments(coreReplicas).Should(Succeed())
			Eventually(checkDSReplicationHealthy).Should(Succeed())
		})

		It("scale down core EMQX cluster", func() {
			coreReplicas = 2
			scaleStartedAt := metav1.Now()
			By("change number of core replicas")
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[{"op": "replace", "path": "/spec/coreTemplate/spec/replicas", "value": 2}]`,
			)).To(Succeed())
			By("wait for EMQX cluster to be ready after scaling")
			Eventually(checkEMQXReady).WithArguments(scaleStartedAt).Should(Succeed())
			Eventually(checkEMQXStatus).WithArguments(coreReplicas).Should(Succeed())
			Eventually(checkReplicantStatus).WithArguments(replicantReplicas).Should(Succeed())
			Eventually(checkDSReplicationStatus).WithArguments(coreReplicas).Should(Succeed())
			// EMQX 5.10.1: Lost sites are expected to hang around.
			// Eventually(checkDSReplicationHealthy).Should(Succeed())
		})

		It("perform a blue-green update", func() {
			By("fetch current core StatefulSet")
			var stsList appsv1.StatefulSetList
			coreRev, err := KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.coreNodesStatus.currentRevision}")
			Expect(err).NotTo(HaveOccurred(), "Failed to get EMQX status")
			Expect(KubectlOut("get", "statefulset",
				"--selector", crdv2.LabelPodTemplateHash+"="+coreRev,
				"-o", "json",
			)).To(UnmarshalInto(&stsList), "Failed to list statefulSets")

			By("change EMQX image + number of replicas")
			coreReplicas = 2
			changedAt := metav1.Now()
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[
					{"op": "replace", "path": "/spec/image", "value": "`+emqxImageUpgrade+`"},
					{"op": "replace", "path": "/spec/coreTemplate/spec/replicas", "value": 2}
				]`,
			)).To(Succeed())

			By("check new core StatefulSet is spinning up")
			Eventually(KubectlOut).
				WithArguments("get", "emqx", "emqx", "-o", "jsonpath={.status.coreNodesStatus.updateRevision}").
				ShouldNot(Equal(coreRev), "New StatefulSet has not been spun up")

			By("wait for EMQX cluster to be ready again")
			Eventually(checkEMQXReady).WithArguments(changedAt).Should(Succeed())
			Eventually(checkEMQXStatus).WithArguments(coreReplicas).Should(Succeed())
			Eventually(checkReplicantStatus).WithArguments(replicantReplicas).Should(Succeed())

			By("check previous coreSet has been scaled down to 0")
			Expect(KubectlOut("get", "statefulset", stsList.Items[0].Name, "-o", "jsonpath={.status.replicas}")).
				To(Equal("0"))

			By("wait for DS replication status to be stable")
			Eventually(checkDSReplicationStatus).WithArguments(coreReplicas).Should(Succeed())
			// EMQX 5.10.1: Lost sites are expected to hang around.
			// Eventually(checkDSReplicationHealthy).Should(Succeed())
		})

		It("delete core-replicant EMQX cluster", func() {
			Expect(Kubectl("delete", "emqx", "emqx")).To(Succeed())
			Expect(Kubectl("get", "emqx", "emqx")).To(HaveOccurred(), "EMQX cluster still exists")
		})

	})

	Context("EMQX Core-Replicant Cluster / Runtime-enabled DS Replication", func() {
		// Initial number of core and replicant replicas:
		var coreReplicas int = 1
		var replicantReplicas int = 2

		It("deploy core-replicant EMQX cluster", func() {
			By("create EMQX cluster")
			emqxCR := PatchDocument(
				FromYAMLFile(emqxCRBasic),
				withImage(emqxImage),
				withCores(coreReplicas),
				withReplicants(replicantReplicas),
				withConfig(),
			)
			Expect(KubectlStdin(emqxCR, "apply", "-f", "-")).To(Succeed())

			By("wait for EMQX cluster to be ready")
			Eventually(checkEMQXReady).Should(Succeed())
			Eventually(checkEMQXStatus).WithArguments(coreReplicas).Should(Succeed())
			Eventually(checkReplicantStatus).WithArguments(replicantReplicas).Should(Succeed())
		})

		It("enable DS replication", func() {
			By("change config + add label to trigger new deployment")
			configDs := string(intoJsonString(configDS()))
			coreReplicas = 2
			changedAt := metav1.Now()
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[
					{"op": "replace", "path": "/spec/config/data", "value": `+configDs+`},
					{"op": "add", "path": "/spec/coreTemplate/metadata/labels", "value": {"e2e/ds-replication": "true"}},
					{"op": "replace", "path": "/spec/coreTemplate/spec/replicas", "value": 2}
				]`,
			)).To(Succeed())

			By("wait for EMQX cluster to become ready again")
			Eventually(checkEMQXReady).WithArguments(changedAt).Should(Succeed())
			Eventually(checkEMQXStatus).WithArguments(coreReplicas).Should(Succeed())
			Eventually(checkReplicantStatus).WithArguments(replicantReplicas).Should(Succeed())
			Eventually(checkDSReplicationStatus).WithArguments(coreReplicas).Should(Succeed())
			// EMQX 5.10.1: Initial cluster's sites are expected to hang around.
			// Eventually(checkDSReplicationHealthy).Should(Succeed())

			By("verify EMQX pods have relevant conditions")
			var pods corev1.PodList
			Expect(KubectlOut("get", "pods",
				"--selector", crdv2.LabelManagedBy+"=emqx-operator",
				"-o", "json",
			)).To(UnmarshalInto(&pods), "Failed to list EMQX pods")
			Expect(pods.Items).To(HaveLen(4))
			for _, pod := range pods.Items {
				if pod.Labels[crdv2.LabelDBRole] == "core" {
					Expect(pod.Status.Conditions).To(ContainElement(And(
						HaveField("Type", Equal(crdv2.DSReplicationSite)),
						HaveField("Status", Equal(corev1.ConditionTrue)),
					)))
				}
				if pod.Labels[crdv2.LabelDBRole] == "replicant" {
					Expect(pod.Status.Conditions).To(ContainElement(And(
						HaveField("Type", Equal(crdv2.DSReplicationSite)),
						HaveField("Status", Equal(corev1.ConditionFalse)),
					)))
				}
			}
		})

		It("delete EMQX cluster", func() {
			Expect(Kubectl("delete", "emqx", "emqx")).To(Succeed())
			Expect(Kubectl("get", "emqx", "emqx")).To(HaveOccurred(), "EMQX cluster still exists")
		})

	})
})
