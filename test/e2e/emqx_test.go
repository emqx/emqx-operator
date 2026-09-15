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
	util "github.com/emqx/emqx-operator/internal/controller/util"
	. "github.com/emqx/emqx-operator/test/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

//nolint:errcheck
var _ = Describe("EMQX Cluster", Label("emqx"), Ordered, func() {

	const (
		emqxCRBasic      = "test/e2e/files/resources/emqx.yaml"
		emqxCRMinimal    = "test/e2e/files/resources/emqx-minimal.yaml"
		emqxImage        = "emqx/emqx:6.2.1"
		emqxImageUpgrade = "emqx/emqx:6.2.2"
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
			DumpDiagnosticReport(namespace, "emqx", CurrentSpecReport().StartTime)
		}
	})

	Context("EMQX Minimal", Label("smoke"), func() {

		It("deploy minimal", func() {
			By("create minimal EMQX")
			emqxCR := FromYAMLFile(emqxCRMinimal)
			Expect(KubectlStdin(emqxCR, "apply", "-f", "-")).To(Succeed())
			By("wait for minimal EMQX to be ready")
			Eventually(EMQXReady).Should(Succeed())
			Eventually(CoresStable).WithArguments(1).Should(Succeed())
			Eventually(NoReplicants).Should(Succeed())
		})

		It("change image to trigger update", func() {
			By("change EMQX image")
			changedAt := metav1.Now()
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[{"op": "replace", "path": "/spec/image", "value": "`+emqxImageUpgrade+`"}]`)).
				To(Succeed())
			Eventually(EMQXReady).WithArguments(changedAt).Should(Succeed())
			Eventually(CoresStable).WithArguments(1).Should(Succeed())
			Eventually(NoReplicants).Should(Succeed())
		})

		It("delete cluster", func() {
			Expect(Kubectl("delete", "emqx", "emqx")).To(Succeed())
		})
	})

	Context("Adoption", func() {

		type objectIdent struct {
			metav1.TypeMeta
			Name string
			UID  types.UID
		}

		objectIdentity := func(o metav1.PartialObjectMetadata) objectIdent {
			return objectIdent{o.TypeMeta, o.Name, o.UID}
		}

		objectIdentities := func(list metav1.PartialObjectMetadataList) []objectIdent {
			ret := []objectIdent{}
			for _, o := range list.Items {
				ret = append(ret, objectIdentity(o))
			}
			return ret
		}

		It("re-adopts resources orphaned by deleting and recreating the EMQX CR", func() {
			const managedResourceTypes = "configmaps,secrets,services,statefulsets,replicasets"

			emqxCR := PatchDocument(
				FromYAMLFile(emqxCRBasic),
				withImage(emqxImage),
				withCores(2),
				withReplicants(1),
				withConfig(),
			)
			defer func() {
				_ = Kubectl("delete", "emqx", "emqx", "--ignore-not-found")
				_ = Kubectl("delete", managedResourceTypes+",persistentvolumeclaims,pods",
					"--selector", emqxLabels.String(), "--ignore-not-found")
			}()

			By("create an EMQX cluster with all managed resource kinds")
			Expect(KubectlStdin(emqxCR, "apply", "-f", "-")).To(Succeed())
			Eventually(EMQXReady).Should(Succeed())
			Eventually(CoresStable).WithArguments(2).Should(Succeed())
			Eventually(ReplicantsStable).WithArguments(1).Should(Succeed())

			var instance crd.EMQX
			Expect(KubectlOut("get", "emqx", "emqx", "-o", "json")).
				To(UnmarshalInto(&instance))

			var originalList metav1.PartialObjectMetadataList
			Expect(KubectlOut("get", managedResourceTypes,
				"--selector", emqxLabels.String(), "-o", "json",
			)).To(
				BeUnmarshalledAs(&originalList,
					HaveField("Items", HaveEach(BeControlledBy(&instance))),
				),
			)

			kindCounts := make(map[string]int)
			for _, resource := range originalList.Items {
				if util.IsManagedBy(&resource.ObjectMeta, &instance) {
					kindCounts[resource.Kind]++
				}
			}
			Expect(kindCounts).To(Equal(map[string]int{
				"ConfigMap":   1,
				"ReplicaSet":  1,
				"Secret":      2,
				"Service":     3,
				"StatefulSet": 1,
			}), "Unexpected set of resources controlled by the EMQX CR")

			By("delete the EMQX CR and orphan its managed resources")
			Expect(Kubectl("delete", "emqx", "emqx", "--cascade=orphan")).To(Succeed())
			Expect(Kubectl("get", "emqx", "emqx")).To(HaveOccurred())

			Expect(KubectlOut("get", managedResourceTypes,
				"--selector", emqxLabels.String(), "-o", "json",
			)).To(
				BeUnmarshalledAs(&metav1.PartialObjectMetadataList{}, And(
					WithTransform(objectIdentities, ConsistOf(objectIdentities(originalList))),
					HaveField("Items", HaveEach(BeNotControlled())),
				)),
				"Orphaning the EMQX CR should preserve every directly managed resource",
			)

			By("recreate the EMQX CR")
			Expect(KubectlStdin(emqxCR, "apply", "-f", "-")).To(Succeed())

			var instanceRecreated crd.EMQX
			Expect(KubectlOut("get", "emqx", "emqx", "-o", "json")).
				To(UnmarshalInto(&instanceRecreated))
			Expect(instanceRecreated.UID).NotTo(Equal(instance.UID))

			By("verify the existing resources are re-adopted by the recreated EMQX CR")
			Eventually(KubectlOut).WithArguments("get", managedResourceTypes,
				"--selector", emqxLabels.String(), "-o", "json",
			).Should(
				BeUnmarshalledAs(&metav1.PartialObjectMetadataList{}, And(
					WithTransform(objectIdentities, ConsistOf(objectIdentities(originalList))),
					HaveField("Items", HaveEach(BeControlledBy(&instanceRecreated))),
				)),
			)

			Eventually(EMQXReady).Should(Succeed())
			Eventually(CoresStable).WithArguments(2).Should(Succeed())
			Eventually(ReplicantsStable).WithArguments(1).Should(Succeed())
		})
	})

	Context("EMQX Cluster", Label("smoke"), func() {
		// Initial number of core replicas:
		var coreReplicas = 2

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
				"--selector", crd.LabelMriaRole+"="+crd.RoleCore,
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
				"--selector", crd.LabelMriaRole+"="+crd.RoleCore,
				"-o", "json",
			)).To(UnmarshalInto(&stsList))

			Expect(stsList.Items).To(
				ConsistOf(And(
					HaveField("ObjectMeta.Name", Equal(stsBefore.Name)),
					HaveField("Status.UpdateRevision", Not(Equal(stsBefore.Status.UpdateRevision))),
				)),
				"Unexpected set of core StatefulSets",
			)
		})

		It("change config", func() {
			By("change EMQX config")
			changedAt := metav1.Now()
			configChange := string(buildConfigRoots(`
				listeners:
				  tcp:
				    default:
				      enabled: true
				      bind: "11883"
				  quic:
				    default:
				      enabled: true
				      bind: "14567"
				  ws:
				    default:
				      enabled: false
				      bind: "0"
				  wss:
				    default:
				      enabled: false
				      bind: "0"
				dashboard:
				  listeners:
				    http:
				      bind: 28083
				      num_acceptors: 1
			`))
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[{"op": "replace", "path": "/spec/config/roots", "value": `+configChange+`}]`)).
				To(Succeed())
			By("wait for EMQX cluster to be ready")
			Eventually(EMQXReady).WithArguments(changedAt).Should(Succeed())
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
		var coreReplicas = 2

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
			configBroken := string(buildConfigRoots(`
				broker:
				  no: { such: [ config ] }
			`))
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[
					{"op": "replace", "path": "/spec/config/roots", "value": `+configBroken+`},
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
					{"op": "replace", "path": "/spec/config/roots", "value": {}},
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
		var coreReplicas = 2
		var replicantReplicas = 2

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
				"--selector", crd.LabelMriaRole+"="+crd.RoleCore,
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
				"--selector", crd.LabelMriaRole+"="+crd.RoleCore,
				"-o", "json",
			)).To(UnmarshalInto(&stsList))

			stsBefore := stsListBefore.Items[0]
			Expect(stsList.Items).To(
				ConsistOf(And(
					HaveField("ObjectMeta.Name", Equal(stsBefore.Name)),
					HaveField("Status.UpdateRevision", Not(Equal(stsBefore.Status.UpdateRevision))),
				)),
				"Unexpected set of core StatefulSets",
			)

			By("check previous replicantSet has been scaled down to 0")
			Expect(KubectlOut("get", "replicaset", rsList.Items[0].Name,
				"-o", "jsonpath={.status.replicas}",
			)).To(Equal("0"))
		})

		It("scale replicants down to zero", func() {
			By("fetch current replicant nodes")
			var replicantNodes []crd.EMQXNode
			Expect(KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.replicantNodes}")).
				To(UnmarshalInto(&replicantNodes), "Failed to get EMQX replicant nodes")
			Expect(replicantNodes).NotTo(BeEmpty(), "No replicant nodes were present before scaling down")

			By("fetch existing replicant ReplicaSets")
			var rsBefore appsv1.ReplicaSetList
			Expect(KubectlOut("get", "replicaset",
				"--selector", crd.LabelInstance+"=emqx,"+crd.LabelMriaRole+"="+crd.RoleReplicant,
				"-o", "json",
			)).To(UnmarshalInto(&rsBefore), "Failed to list replicant ReplicaSets")
			Expect(rsBefore.Items).NotTo(BeEmpty(), "No replicant ReplicaSets were present before scaling down")

			By("scale replicants to zero and constrain revision history")
			const revisionHistoryLimit = 1
			replicantReplicas = 0
			changeTime := metav1.Now()
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[
						{"op": "replace", "path": "/spec/replicantTemplate/spec/replicas", "value": 0},
						{"op": "replace", "path": "/spec/revisionHistoryLimit", "value": `+fmt.Sprint(revisionHistoryLimit)+`}
					]`,
			)).To(Succeed(), "Failed to scale replicants down to 0")

			By("wait for EMQX to stabilize without replicants")
			Eventually(EMQXReady).WithArguments(changeTime).Should(Succeed())
			Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
			Eventually(NoReplicants).Should(Succeed())

			By("verify replicant ReplicaSets are retained according to revisionHistoryLimit")
			Eventually(KubectlOut).WithArguments("get", "replicaset",
				"--selector", emqxReplicantLabels.String(),
				"-o", "json",
			).Should(BeUnmarshalledAs(&appsv1.ReplicaSetList{}, HaveField("Items", And(
				HaveLen(revisionHistoryLimit),
				HaveEach(And(
					HaveField("Spec.Replicas", HaveValue(BeEquivalentTo(0))),
					HaveField("Status.Replicas", BeEquivalentTo(0)),
				)),
			))))
		})

		It("delete cluster", func() {
			Expect(Kubectl("delete", "emqx", "emqx")).To(Succeed())
			Expect(Kubectl("get", "emqx", "emqx")).To(HaveOccurred(), "EMQX cluster still exists")
		})
	})

	Context("EMQX Core-Replicant Cluster / Botched Rolling Updates", func() {
		const (
			coreReplicas      = 2
			replicantReplicas = 2
		)

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
			var statusInitial crd.EMQXStatus
			Eventually(EMQXReady).Should(Succeed())
			Expect(KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status}")).
				To(UnmarshalInto(&statusInitial))

			By("specify incorrect EMQX image")
			changedAt1 := metav1.Now()
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[
					{"op": "replace", "path": "/spec/image", "value": "emqx/emqx:6.Y.ZZZ"}
				]`)).
				To(Succeed())
			Consistently(EMQXReady, "30s", "3s").WithArguments(changedAt1).Should(Not(Succeed()))

			Expect(KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status}")).
				To(BeUnmarshalledAs(&crd.EMQXStatus{}, And(
					HaveField("CoreNodesStatus.ReadyReplicas", Equal(statusInitial.CoreNodesStatus.ReadyReplicas-1)),
					HaveField("ReplicantNodesStatus.ReadyReplicas", Equal(statusInitial.ReplicantNodesStatus.ReadyReplicas-1)),
				)), "exactly 1 core and replicant replica went unavailable")

			By("restore correct EMQX image")
			changedAt2 := metav1.Now()
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[
					{"op": "replace", "path": "/spec/image", "value": "`+emqxImageUpgrade+`"}
				]`)).
				To(Succeed())
			Eventually(EMQXReady).WithArguments(changedAt2).Should(Succeed())
			Eventually(CoresStable).WithArguments(coreReplicas).Should(Succeed())
			Eventually(ReplicantsStable).WithArguments(replicantReplicas).Should(Succeed())
		})

		It("delete cluster", func() {
			Expect(Kubectl("delete", "emqx", "emqx")).To(Succeed())
		})
	})

	Context("EMQX Core-Replicant DS-Enabled Cluster", func() {
		// Initial number of core and replicant replicas:
		var coreReplicas = 2
		var replicantReplicas = 2

		It("deploy core-replicant EMQX cluster", func() {
			By("create EMQX cluster")
			emqxCR := PatchDocument(
				FromYAMLFile(emqxCRBasic),
				withImage(emqxImage),
				withCores(coreReplicas),
				withReplicants(replicantReplicas),
				withDS(),
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
				if pod.Labels[crd.LabelMriaRole] == crd.RoleCore {
					Expect(pod.Status.Conditions).To(ContainElement(And(
						HaveField("Type", Equal(crd.DSReplicationSite)),
						HaveField("Status", Equal(corev1.ConditionTrue)),
					)))
				}
				if pod.Labels[crd.LabelMriaRole] == crd.RoleReplicant {
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
				"--selector", crd.LabelMriaRole+"="+crd.RoleCore,
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
				"--selector", crd.LabelMriaRole+"="+crd.RoleCore,
				"-o", "json",
			)).To(UnmarshalInto(&stsList))

			stsBefore := stsListBefore.Items[0]
			Expect(stsList.Items).To(
				ConsistOf(And(
					HaveField("ObjectMeta.Name", Equal(stsBefore.Name)),
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
		var coreReplicas = 2
		var replicantReplicas = 2

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
			By("change DS config to trigger rolling update")
			configDs := string(buildConfigRoots(configDS))
			changedAt := metav1.Now()
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "json",
				"--patch", `[{"op": "replace", "path": "/spec/config/roots", "value": `+configDs+`}]`,
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
				if pod.Labels[crd.LabelMriaRole] == crd.RoleCore {
					Expect(pod.Status.Conditions).To(ContainElement(And(
						HaveField("Type", Equal(crd.DSReplicationSite)),
						HaveField("Status", Equal(corev1.ConditionTrue)),
					)))
				}
				if pod.Labels[crd.LabelMriaRole] == crd.RoleReplicant {
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
		var coreReplicas = 2
		var replicantReplicas = 2

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
				ContainSubstring(crd.LabelMriaRole+"="+crd.RoleReplicant),
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
