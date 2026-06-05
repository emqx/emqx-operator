package e2e

import (
	crdv2 "github.com/emqx/emqx-operator/api/v2"
	. "github.com/emqx/emqx-operator/test/util"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var (
	emqxLabels = labels.Set(map[string]string{
		crdv2.LabelInstance:  "emqx",
		crdv2.LabelManagedBy: "emqx-operator",
	})
	emqxReplicantLabels = labels.Set(map[string]string{
		crdv2.LabelInstance:  "emqx",
		crdv2.LabelManagedBy: "emqx-operator",
		crdv2.LabelMriaRole:  crdv2.RoleReplicant,
	})
)

func checkEMQXReady(g Gomega, afterTime ...metav1.Time) {
	var cond metav1.Condition
	g.Expect(KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.conditions[?(@.type==\"Ready\")]}")).
		To(UnmarshalInto(&cond), "Failed to get emqx status")
	g.Expect(cond.Status).To(
		Equal(metav1.ConditionTrue),
		"EMQX cluster has not become ready",
	)
	if len(afterTime) > 0 {
		g.Expect(cond.LastTransitionTime.After(afterTime[0].Time)).To(
			BeTrue(),
			"EMQX cluster has not become ready after specified time",
		)
	}
}

func checkEMQXStatus(g Gomega, coreReplicas int) {
	var podList corev1.PodList
	var status crdv2.EMQXNodesStatus
	g.Expect(KubectlOut("get", "pod",
		"--selector", emqxLabels.String(),
		"-o", "json",
	)).To(UnmarshalInto(&podList), "Failed to list EMQX pods")
	g.Expect(podList.Items).To(
		HaveEach(
			HaveField("Status.Conditions", ContainElement(And(
				HaveField("Type", Equal(corev1.PodReady)),
				HaveField("Status", Equal(corev1.ConditionTrue)),
			))),
		),
		"Not all EMQX pods are ready",
	)
	g.Expect(KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.coreNodesStatus}")).
		To(UnmarshalInto(&status), "Failed to get EMQX status")
	g.Expect(status).To(
		And(
			HaveField("Replicas", BeEquivalentTo(coreReplicas)),
			HaveField("ReadyReplicas", BeEquivalentTo(coreReplicas)),
			HaveField("CurrentReplicas", BeEquivalentTo(coreReplicas)),
			HaveField("UpdateReplicas", BeEquivalentTo(coreReplicas)),
		),
		"EMQX status does not have expected number of core nodes",
	)
	checkNodesStatusRevision(g, status, "core", coreReplicas)
	checkNoStoppedNodes(g, "coreNodes", "core")
}

func checkNoReplicants(g Gomega) {
	g.Expect(KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.replicantNodesStatus}")).
		To(Equal("{}"), "EMQX cluster status has replicant nodes status")
	g.Expect(KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.replicantNodes}")).
		To(BeEmpty(), "EMQX cluster status lists replicant nodes")
	g.Expect(KubectlOut("get", "pods",
		"--selector", emqxReplicantLabels.String(),
		"-o", "json",
	)).To(BeUnmarshalledAs(&corev1.PodList{},
		HaveField("Items", BeEmpty()),
	))
}

func checkReplicantStatus(g Gomega, replicantReplicas int) {
	var status crdv2.EMQXNodesStatus
	g.Expect(KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.replicantNodesStatus}")).
		To(UnmarshalInto(&status), "Failed to get EMQX replicant nodes status")
	g.Expect(status).To(
		And(
			HaveField("Replicas", BeEquivalentTo(replicantReplicas)),
			HaveField("ReadyReplicas", BeEquivalentTo(replicantReplicas)),
			HaveField("CurrentReplicas", BeEquivalentTo(replicantReplicas)),
			HaveField("UpdateReplicas", BeEquivalentTo(replicantReplicas)),
		),
		"EMQX status does not have expected number of replicant nodes",
	)
	checkNodesStatusRevision(g, status, "replicant", replicantReplicas)
	checkNoStoppedNodes(g, "replicantNodes", "replicant")
}

func checkNoStoppedNodes(g Gomega, statusField, role string) {
	var nodes []crdv2.EMQXNode
	g.Expect(KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status."+statusField+"}")).
		To(UnmarshalInto(&nodes), "Failed to get EMQX %s nodes", role)
	g.Expect(nodes).To(
		HaveEach(Not(HaveField("Status", Equal("stopped")))),
		"EMQX %s nodes status lists stopped nodes: %+v",
		role,
		nodes,
	)
}

func checkNodesStatusRevision(g Gomega, status crdv2.EMQXNodesStatus, role string, replicas int) {
	var podList corev1.PodList
	g.Expect(status).To(
		And(
			HaveField("CurrentRevision", Not(BeEmpty())),
			HaveField("UpdateRevision", Not(BeEmpty())),
		),
		"EMQX %s nodes status does not have expected revision", role,
	)
	g.Expect(status.CurrentRevision).To(
		Equal(status.UpdateRevision),
		"EMQX %s nodes current and update revisions are different", role,
	)
	g.Expect(KubectlOut("get", "pods",
		"--selector", crdv2.LabelPodTemplateHash+"="+status.CurrentRevision,
		"--field-selector", "status.phase==Running",
		"-o", "json",
	)).To(UnmarshalInto(&podList), "Failed to list %s pods", role)
	g.Expect(podList.Items).To(
		HaveLen(replicas),
		"EMQX cluster does not have %d current revision %s pods", replicas, role,
	)
}

func checkDSReplicationStatus(g Gomega, coreReplicas int) {
	status := &crdv2.DSReplicationStatus{}
	replicationFactor := min(3, coreReplicas)
	g.Expect(KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.dsReplication}")).
		To(UnmarshalInto(&status), "Failed to get emqx status")
	g.Expect(status.DBs).NotTo(BeEmpty(), "No DS database is present")
	g.Expect(status.DBs).To(
		HaveEach(And(
			HaveField("Name", Not(BeEmpty())),
			HaveField("NumShards", Not(BeZero())),
			HaveField("NumShardReplicas", Not(BeZero())),
			Satisfy(func(db crdv2.DSDBReplicationStatus) bool {
				return db.NumShardReplicas == db.NumShards*int32(replicationFactor)
			}),
			HaveField("LostShardReplicas", BeEquivalentTo(0)),
			HaveField("NumTransitions", BeEquivalentTo(0)),
			HaveField("MinReplicas", BeEquivalentTo(replicationFactor)),
			HaveField("MaxReplicas", BeEquivalentTo(replicationFactor)),
		)),
		"EMQX DS databases are not replicated correctly across %d core nodes", coreReplicas,
	)
}

func checkDSReplicationHealthy(g Gomega) {
	g.Expect(KubectlOut("exec", "service/emqx-listeners", "--", "emqx", "ctl", "ds", "info")).
		NotTo(
			ContainSubstring("(!)"),
			"EMQX DS replication is not healthy",
		)
}
