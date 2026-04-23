/*
Copyright 2025-2026.

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

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	. "github.com/emqx/emqx-operator/test/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func withReplicantResources(cpuRequest, memRequest, cpuLimit, memLimit string) []byte {
	return fmt.Appendf(nil,
		`{"spec": {"replicantTemplate": {"spec": {"resources": {
			"requests": {"cpu": "%s", "memory": "%s"},
			"limits": {"cpu": "%s", "memory": "%s"}
		}}}}}`,
		cpuRequest, memRequest, cpuLimit, memLimit,
	)
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
var _ = Describe("EMQX Cluster", Label("emqx"), Ordered, func() {

	const (
		emqxCRBasic      = "test/e2e/files/resources/emqx.yaml"
		emqxImage        = "emqx/emqx:5.10.0"
		emqxImageUpgrade = "emqx/emqx:5.10.1"
	)

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
			Eventually(EMQXReady).Should(Succeed())
			Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
			Eventually(NoReplicants).Should(Succeed())
		})

		It("scale cluster up", func() {
			coreReplicas = 4
			scaleupStartedAt := metav1.Now()
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", fmt.Sprintf(
					`[{"op": "replace", "path": "/spec/coreTemplate/spec/replicas", "value": %d}]`,
					coreReplicas,
				),
			)).To(Succeed(), "Failed to scale up EMQX cluster")
			Eventually(EMQXReady).WithArguments(scaleupStartedAt).Should(Succeed())
			Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
			Eventually(NoReplicants).Should(Succeed())
		})

		It("scale cluster down", func() {
			coreReplicas = 3
			scaledownStartedAt := metav1.Now()
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", fmt.Sprintf(
					`[{"op": "replace", "path": "/spec/coreTemplate/spec/replicas", "value": %d}]`,
					coreReplicas,
				),
			)).To(Succeed(), "Failed to scale down EMQX cluster")
			Eventually(EMQXReady).WithArguments(scaledownStartedAt).Should(Succeed())
			Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
			Eventually(NoReplicants).Should(Succeed())
		})

		It("change image to trigger rolling update", func() {
			By("create MQTTX client")
			Expect(Kubectl("apply", "-f", "test/e2e/files/resources/mqttx.yaml")).To(Succeed())
			defer Kubectl("delete", "-f", "test/e2e/files/resources/mqttx.yaml")
			Expect(Kubectl("wait", "pod",
				"--selector=app=mqttx",
				"--for=condition=Ready",
				"--timeout=1m",
			)).To(Succeed(), "Timed out waiting MQTTX to be ready")

			By("fetch core StatefulSet")
			var stsListBefore appsv1.StatefulSetList
			Expect(KubectlOut("get", "statefulset",
				"--selector", crd.LabelDBRole+"=core",
				"-o", "json",
			)).To(UnmarshalInto(&stsListBefore))
			Expect(stsListBefore.Items).To(HaveLen(1), "More than one core StatefulSet")
			stsBefore := stsListBefore.Items[0].DeepCopy()

			By("change EMQX image")
			changedAt := metav1.Now()
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[{"op": "replace", "path": "/spec/image", "value": "`+emqxImageUpgrade+`"}]`)).
				To(Succeed())

			By("check EMQX cluster node evacuations status")
			Eventually(KubectlOut).
				WithArguments("get", "emqx", "emqx", "-o", "jsonpath={.status.nodeEvacuations}").
				Should(BeEmpty())

			Eventually(EMQXReady).WithArguments(changedAt).Should(Succeed())
			Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
			Eventually(NoReplicants).Should(Succeed())

			By("verify exactly one core StatefulSet was updated")
			var stsList appsv1.StatefulSetList
			Expect(KubectlOut("get", "statefulset",
				"--selector", crd.LabelDBRole+"=core",
				"-o", "json",
			)).To(UnmarshalInto(&stsList))

			Expect(stsList.Items).To(
				ConsistOf(And(
					HaveField("ObjectMeta.Name", Equal(stsBefore.ObjectMeta.Name)),
					HaveField("Status.UpdateRevision", Not(Equal(stsBefore.Status.UpdateRevision))),
				)),
				"Unexpected set of core StatefulSets",
			)
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
			Eventually(EMQXReady).Should(Succeed())
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

	Context("EMQX Cluster / Botched Rolling Updates", func() {
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
			Eventually(EMQXReady).Should(Succeed())
			Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
		})

		It("trigger botched rolling updates", func() {
			By("create MQTT workload")
			Expect(Kubectl("apply", "-f", "test/e2e/files/resources/mqttx.yaml")).To(Succeed())
			defer Kubectl("delete", "-f", "test/e2e/files/resources/mqttx.yaml")
			Expect(Kubectl("wait", "pod",
				"--selector=app=mqttx",
				"--for=condition=Ready",
				"--timeout=1m",
			)).To(Succeed(), "Timed out waiting MQTTX to be ready")

			By("lookup initial EMQX status")
			var statusInitial crd.CoreNodesStatus
			Eventually(EMQXReady).Should(Succeed())
			Expect(KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.coreNodesStatus}")).
				To(UnmarshalInto(&statusInitial))

			By("specify incorrect EMQX image")
			changedAt1 := metav1.Now()
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[
					{"op": "replace", "path": "/spec/image", "value": "emqx/emqx:5.Y.ZZZ"}
				]`)).
				To(Succeed())
			Consistently(EMQXReady, "30s", "3s").WithArguments(changedAt1).Should(Not(Succeed()))

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
			Consistently(EMQXReady, "30s", "3s").WithArguments(changedAt2).Should(Not(Succeed()))

			var status crd.CoreNodesStatus
			Expect(KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.coreNodesStatus}")).
				To(
					BeUnmarshalledAs(&status,
						HaveField("ReadyReplicas", Equal(statusInitial.ReadyReplicas-1)),
					),
					"no more than 1 replica went unavailable",
				)

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
			Eventually(EMQXReady).WithArguments(changedAt3).Should(Succeed())
			Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
		})

		It("delete cluster", func() {
			Expect(Kubectl("delete", "emqx", "emqx")).To(Succeed())
		})
	})

	Context("EMQX Core-Replicant Cluster", func() {
		// Initial number of core and replicant replicas:
		var coreReplicas int = 2
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
			Eventually(EMQXReady).Should(Succeed())
			Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
			Eventually(ReplicantsStable).WithArguments(replicantReplicas).Should(Succeed())
		})

		It("scale cluster up", func() {
			coreReplicas = 3
			replicantReplicas = 3
			scaleupStartedAt := metav1.Now()
			By("change number of core replicas")
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[{"op": "replace", "path": "/spec/coreTemplate/spec/replicas", "value": 3}]`)).
				To(Succeed(), "Failed to scale emqx cluster")
			By("change number of replicant replicas")
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[{"op": "replace", "path": "/spec/replicantTemplate/spec/replicas", "value": 3}]`)).
				To(Succeed(), "Failed to scale emqx cluster")
			By("wait for EMQX cluster to be ready after scaling")
			Eventually(EMQXReady).WithArguments(scaleupStartedAt).Should(Succeed())
			Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
			Eventually(ReplicantsStable).WithArguments(replicantReplicas).Should(Succeed())
		})

		It("scale cluster down", func() {
			coreReplicas = 2
			replicantReplicas = 2
			scaledownStartedAt := metav1.Now()
			By("change number of core replicas")
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[{"op": "replace", "path": "/spec/coreTemplate/spec/replicas", "value": 2}]`)).
				To(Succeed(), "Failed to scale emqx cluster")
			By("change number of replicant replicas")
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[{"op": "replace", "path": "/spec/replicantTemplate/spec/replicas", "value": 2}]`)).
				To(Succeed(), "Failed to scale emqx cluster")
			By("wait for EMQX cluster to be ready after scaling")
			Eventually(EMQXReady).WithArguments(scaledownStartedAt).Should(Succeed())
			Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
			Eventually(ReplicantsStable).WithArguments(replicantReplicas).Should(Succeed())
		})

		It("change image to trigger rolling update", func() {
			By("create MQTTX client")
			Expect(Kubectl("apply", "-f", "test/e2e/files/resources/mqttx.yaml")).To(Succeed())
			defer Kubectl("delete", "-f", "test/e2e/files/resources/mqttx.yaml")
			Expect(Kubectl("wait", "pod",
				"--selector=app=mqttx",
				"--for=condition=Ready",
				"--timeout=1m",
			)).To(Succeed(), "Timed out waiting for MQTTX to be ready")

			By("fetch core StatefulSet")
			var stsListBefore appsv1.StatefulSetList
			Expect(KubectlOut("get", "statefulset",
				"--selector", crd.LabelDBRole+"=core",
				"-o", "json",
			)).To(UnmarshalInto(&stsListBefore))
			Expect(stsListBefore.Items).To(HaveLen(1), "More than one core StatefulSet")

			By("fetch current replicant ReplicaSet")
			var rsList appsv1.ReplicaSetList
			replRev, err := KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.replicantNodesStatus.currentRevision}")
			Expect(err).NotTo(HaveOccurred(), "Failed to get EMQX status")
			Expect(KubectlOut("get", "replicaset",
				"--selector", crd.LabelPodTemplateHash+"="+replRev,
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
				WithArguments("get", "emqx", "emqx", "-o", "jsonpath={.status.nodeEvacuations}").
				ShouldNot(ContainSubstring("connection_eviction_rate"))

			By("wait for EMQX cluster to be ready again")
			Eventually(EMQXReady).WithArguments(changingTime).Should(Succeed())
			Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
			Eventually(ReplicantsStable).WithArguments(replicantReplicas).Should(Succeed())

			By("verify exactly one core StatefulSet was updated")
			var stsList appsv1.StatefulSetList
			Expect(KubectlOut("get", "statefulset",
				"--selector", crd.LabelDBRole+"=core",
				"-o", "json",
			)).To(UnmarshalInto(&stsList))

			stsBefore := stsListBefore.Items[0]
			Expect(stsList.Items).To(
				ConsistOf(And(
					HaveField("ObjectMeta.Name", Equal(stsBefore.ObjectMeta.Name)),
					HaveField("Status.UpdateRevision", Not(Equal(stsBefore.Status.UpdateRevision))),
				)),
				"Unexpected set of core StatefulSets",
			)

			By("check previous replicantSet has been scaled down to 0")
			Expect(KubectlOut("get", "replicaset", rsList.Items[0].Name,
				"-o", "jsonpath={.status.replicas}",
			)).To(Equal("0"))
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
				withConfig(ConfigDS()),
			)
			Expect(KubectlStdin(emqxCR, "apply", "-f", "-")).To(Succeed())

			By("wait for EMQX cluster to be ready")
			Eventually(EMQXReady).Should(Succeed())
			Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
			Eventually(ReplicantsStable).WithArguments(replicantReplicas).Should(Succeed())
			Eventually(DSReplicationStable).WithArguments(coreReplicas).Should(Succeed())
			Eventually(DSReplicationHealthy).Should(Succeed())

			By("verify EMQX pods have relevant conditions")
			var pods corev1.PodList
			Expect(KubectlOut("get", "pods",
				"--selector", crd.LabelManagedBy+"=emqx-operator",
				"-o", "json",
			)).To(UnmarshalInto(&pods), "Failed to list EMQX pods")
			Expect(pods.Items).To(HaveLen(4), "EMQX cluster does not have 4 pods")
			for _, pod := range pods.Items {
				if pod.Labels[crd.LabelDBRole] == "core" {
					Expect(pod.Status.Conditions).To(ContainElement(And(
						HaveField("Type", Equal(crd.DSReplicationSite)),
						HaveField("Status", Equal(corev1.ConditionTrue)),
					)))
				}
				if pod.Labels[crd.LabelDBRole] == "replicant" {
					Expect(pod.Status.Conditions).To(ContainElement(And(
						HaveField("Type", Equal(crd.DSReplicationSite)),
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
			Eventually(EMQXReady).WithArguments(scaleStartedAt).Should(Succeed())
			Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
			Eventually(ReplicantsStable).WithArguments(replicantReplicas).Should(Succeed())
			Eventually(DSReplicationStable).WithArguments(coreReplicas).Should(Succeed())
			Eventually(DSReplicationHealthy).Should(Succeed())
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
			Eventually(EMQXReady).WithArguments(scaleStartedAt).Should(Succeed())
			Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
			Eventually(ReplicantsStable).WithArguments(replicantReplicas).Should(Succeed())
			Eventually(DSReplicationStable).WithArguments(coreReplicas).Should(Succeed())
			// EMQX 5.10.1: Lost sites are expected to hang around.
			// Eventually(DSReplicationHealthy).Should(Succeed())
		})

		It("perform a rolling update", func() {
			By("fetch core StatefulSet")
			var stsListBefore appsv1.StatefulSetList
			Expect(KubectlOut("get", "statefulset",
				"--selector", crd.LabelDBRole+"=core",
				"-o", "json",
			)).To(UnmarshalInto(&stsListBefore))
			Expect(stsListBefore.Items).To(HaveLen(1), "More than one core StatefulSet")

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

			By("wait for EMQX cluster to be ready again")
			Eventually(EMQXReady).WithArguments(changedAt).Should(Succeed())
			Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
			Eventually(ReplicantsStable).WithArguments(replicantReplicas).Should(Succeed())

			By("verify exactly one core StatefulSet was updated")
			var stsList appsv1.StatefulSetList
			Expect(KubectlOut("get", "statefulset",
				"--selector", crd.LabelDBRole+"=core",
				"-o", "json",
			)).To(UnmarshalInto(&stsList))

			stsBefore := stsListBefore.Items[0]
			Expect(stsList.Items).To(
				ConsistOf(And(
					HaveField("ObjectMeta.Name", Equal(stsBefore.ObjectMeta.Name)),
					HaveField("Status.UpdateRevision", Not(Equal(stsBefore.Status.UpdateRevision))),
				)),
				"Unexpected set of core StatefulSets",
			)

			By("wait for DS replication status to be stable")
			Eventually(DSReplicationStable).WithArguments(coreReplicas).Should(Succeed())
			// EMQX 5.10.1: Lost sites are expected to hang around.
			// Eventually(DSReplicationHealthy).Should(Succeed())
		})

		It("delete core-replicant EMQX cluster", func() {
			Expect(Kubectl("delete", "emqx", "emqx")).To(Succeed())
			Expect(Kubectl("get", "emqx", "emqx")).To(HaveOccurred(), "EMQX cluster still exists")
		})

	})

	Context("EMQX Core-Replicant Cluster / Runtime-enabled DS Replication", func() {
		// Initial number of core and replicant replicas:
		var coreReplicas int = 2
		var replicantReplicas int = 2

		const emqxImage = "emqx/emqx:5.10.2"

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
			Eventually(EMQXReady).Should(Succeed())
			Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
			Eventually(ReplicantsStable).WithArguments(replicantReplicas).Should(Succeed())
		})

		It("enable DS replication", func() {
			By("change config + add label to trigger rolling update")
			configDs := string(intoJsonString(ConfigDS()))
			changedAt := metav1.Now()
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[
					{"op": "replace", "path": "/spec/config/data", "value": `+configDs+`},
					{"op": "add", "path": "/spec/coreTemplate/metadata/labels", "value": {"e2e/ds-replication": "true"}}
				]`,
			)).To(Succeed())

			By("wait for EMQX cluster to become ready again")
			Eventually(EMQXReady).WithArguments(changedAt).Should(Succeed())
			Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
			Eventually(ReplicantsStable).WithArguments(replicantReplicas).Should(Succeed())
			Eventually(DSReplicationStable).WithArguments(coreReplicas).Should(Succeed())
			// EMQX 5.10.1: Initial cluster's sites are expected to hang around.
			// Eventually(DSReplicationHealthy).Should(Succeed())

			By("verify EMQX pods have relevant conditions")
			var pods corev1.PodList
			Expect(KubectlOut("get", "pods",
				"--selector", crd.LabelManagedBy+"=emqx-operator",
				"-o", "json",
			)).To(UnmarshalInto(&pods), "Failed to list EMQX pods")
			Expect(pods.Items).To(HaveLen(4))
			for _, pod := range pods.Items {
				if pod.Labels[crd.LabelDBRole] == "core" {
					Expect(pod.Status.Conditions).To(ContainElement(And(
						HaveField("Type", Equal(crd.DSReplicationSite)),
						HaveField("Status", Equal(corev1.ConditionTrue)),
					)))
				}
				if pod.Labels[crd.LabelDBRole] == "replicant" {
					Expect(pod.Status.Conditions).To(ContainElement(And(
						HaveField("Type", Equal(crd.DSReplicationSite)),
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

	Context("EMQX Cluster Scaling / HPA", Label("scale", "hpa"), func() {
		// Initial number of core and replicant replicas:
		var coreReplicas int = 2
		var replicantReplicas int = 2

		const (
			hpaName = "emqx-replicant"
		)

		It("deploy core-replicant cluster", func() {
			By("create EMQX cluster with resource requests for HPA")
			emqxCR := PatchDocument(
				FromYAMLFile(emqxCRBasic),
				withImage(emqxImage),
				withCores(coreReplicas),
				withReplicants(replicantReplicas),
				withReplicantResources("500m", "512Mi", "1", "2Gi"),
				withConfig(),
				// Make EMQX react to scaling decisions quicker:
				[]byte(`
                {"spec": {"updateStrategy": {
                    "replicants": {"maxUnavailable": 3, "maxSurge": 1}
                }}}`),
			)
			Expect(KubectlStdin(emqxCR, "apply", "-f", "-")).To(Succeed())
			By("wait for EMQX cluster to be ready")
			Eventually(EMQXReady).Should(Succeed())
			Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
			Eventually(ReplicantsStable).WithArguments(replicantReplicas).Should(Succeed())
		})

		It("scale subresource reports correct replica counts and selector", func() {
			By("read the scale subresource via kubectl")
			var scale autoscalingv1.Scale
			Eventually(KubectlOut).WithArguments(
				"get", "--raw", "/apis/apps.emqx.io/v3beta1/namespaces/default/emqxes/emqx/scale",
			).Should(BeUnmarshalledAs(&scale, And(
				HaveField("Spec.Replicas", BeEquivalentTo(replicantReplicas)),
				HaveField("Status.Replicas", BeEquivalentTo(replicantReplicas)),
				HaveField("Status.Selector", Not(BeEmpty())),
			)), "Scale subresource should report replicant replica counts and a non-empty selector")

			By("verify the selector matches replicant pods")
			Expect(scale.Status.Selector).To(
				ContainSubstring(crd.LabelDBRole+"=replicant"),
				"Scale selector should include the replicant role label",
			)
			Expect(scale.Status.Selector).To(
				ContainSubstring(crd.LabelInstance+"=emqx"),
				"Scale selector should include the instance label",
			)

			By("verify replicant pods match the scale selector")
			var podList corev1.PodList
			Expect(KubectlOut("get", "pods",
				"--selector", scale.Status.Selector,
				"-o", "json",
			)).To(UnmarshalInto(&podList), "Failed to list pods matching scale selector")
			Expect(podList.Items).To(
				HaveLen(replicantReplicas),
				"Scale selector should match exactly %d replicant pods", replicantReplicas,
			)
		})

		It("kubectl scale changes replicant replica count", func() {
			newReplicas := 4
			By("scale replicants via kubectl scale")
			Expect(Kubectl("scale", "emqx", "emqx", "--replicas="+fmt.Sprint(newReplicas))).
				To(Succeed(), "kubectl scale should succeed")

			By("wait for EMQX cluster to be ready after scaling")
			Eventually(EMQXReady).Should(Succeed())
			Eventually(ReplicantsStable).WithArguments(newReplicas).Should(Succeed())

			By("verify scale subresource reflects the new replica count")
			var scale autoscalingv1.Scale
			Eventually(KubectlOut).WithArguments(
				"get", "--raw", "/apis/apps.emqx.io/v3beta1/namespaces/default/emqxes/emqx/scale",
			).Should(BeUnmarshalledAs(&scale, And(
				HaveField("Spec.Replicas", BeEquivalentTo(newReplicas)),
				HaveField("Status.Replicas", BeEquivalentTo(newReplicas)),
			)), "Scale subresource should reflect the new replica count after kubectl scale")

			replicantReplicas = newReplicas
		})

		It("create HPA targeting replicants", func() {
			By("create a HorizontalPodAutoscaler")
			hpaCreatedAt := metav1.Now()
			minReplicas := 1
			maxReplicas := 6
			hpa := FromYAMLString(`
			apiVersion: autoscaling/v2
			kind: HorizontalPodAutoscaler
			metadata:
			  name: ` + hpaName + `
			spec:
			  minReplicas: ` + fmt.Sprint(minReplicas) + `
			  maxReplicas: ` + fmt.Sprint(maxReplicas) + `
			  scaleTargetRef:
			    apiVersion: apps.emqx.io/v3beta1
			    kind: EMQX
			    name: emqx
			  metrics:
			  - type: Resource
			    resource:
			      name: cpu
			      target:
			        averageUtilization: 80
			        type: Utilization
			  behavior:
			    scaleUp:
				  stabilizationWindowSeconds: 0
			    scaleDown:
				  stabilizationWindowSeconds: 30
			`)
			Expect(KubectlStdin(hpa, "apply", "-f", "-")).To(Succeed(), "Failed to create HPA")

			By("verify HPA can read the scale subresource")
			var hpaOut autoscalingv2.HorizontalPodAutoscaler
			Eventually(KubectlOut).WithArguments("get", "hpa", hpaName, "-o", "json").
				Should(BeUnmarshalledAs(&hpaOut, And(
					HaveField("Status.DesiredReplicas", BeEquivalentTo(replicantReplicas)),
					HaveField("Status.CurrentReplicas", BeEquivalentTo(replicantReplicas)),
				)), "HPA should observe current replicant replica count via scale subresource")

			By("verify HPA eventually scales replicants down")
			Eventually(KubectlOut).WithArguments("get", "hpa", hpaName, "-o", "json").
				Should(BeUnmarshalledAs(&hpaOut,
					HaveField("Status.CurrentReplicas", BeEquivalentTo(minReplicas)),
				), "HPA should scale replicant set down")

			Eventually(EMQXReady).WithArguments(hpaCreatedAt).Should(Succeed())
			Eventually(ReplicantsStable).WithArguments(minReplicas).Should(Succeed())
		})

		It("delete cluster and HPA", func() {
			Expect(Kubectl("delete", "hpa", hpaName)).To(Succeed())
			Expect(Kubectl("delete", "emqx", "emqx")).To(Succeed())
			Expect(Kubectl("get", "emqx", "emqx")).To(HaveOccurred(), "EMQX cluster still exists")
		})

	})
})
