package e2e

import (
	crdv2 "github.com/emqx/emqx-operator/api/v2"
	. "github.com/emqx/emqx-operator/test/util"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	var status crdv2.CoreNodesStatus
	var nodes []crdv2.EMQXNode
	var podList corev1.PodList
	var pvcList corev1.PersistentVolumeClaimList
	g.Expect(KubectlOut("get", "pod",
		"--selector", "apps.emqx.io/instance=emqx,apps.emqx.io/managed-by=emqx-operator",
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
			HaveField("ReadyReplicas", BeEquivalentTo(coreReplicas)),
			HaveField("UpdatedReplicas", BeEquivalentTo(coreReplicas)),
		),
		"EMQX status does not have expected number of core nodes",
	)
	g.Expect(KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.coreNodes}")).
		To(UnmarshalInto(&nodes), "Failed to get EMQX cluster nodes")
	g.Expect(nodes).To(
		HaveEach(HaveField("Status", Equal("running"))),
		"EMQX cluster contains stopped nodes",
	)
	g.Expect(nodes).To(
		HaveEach(HaveField("PodName", Not(BeEmpty()))),
		"EMQX cluster contains nodes without pods",
	)
	g.Expect(KubectlOut("get", "pvc",
		"--selector", crdv2.LabelDBRole+"=core,"+crdv2.LabelManagedBy+"=emqx-operator",
		"-o", "json",
	)).To(UnmarshalInto(&pvcList), "Failed to list core PVCs")
	g.Expect(pvcList.Items).To(
		HaveLen(coreReplicas),
		"Expected %d core PVCs", coreReplicas,
	)
	g.Expect(pvcList.Items).To(
		HaveEach(HaveField("Status.Phase", Equal(corev1.ClaimBound))),
		"Not all core PVCs are bound",
	)
}

func checkNoReplicants(g Gomega) {
	g.Expect(KubectlOut("get", "emqx", "emqx",
		"-o", "jsonpath={.status.replicantNodesStatus.currentReplicas}",
	)).To(Equal("0"), "EMQX cluster status has replicant replicas")
	g.Expect(KubectlOut("get", "emqx", "emqx",
		"-o", "jsonpath={.status.replicantNodes}",
	)).To(BeEmpty(), "EMQX cluster status lists replicant nodes")
}

func checkReplicantStatus(g Gomega, replicantReplicas int) {
	var status crdv2.ReplicantNodesStatus
	g.Expect(KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.replicantNodesStatus}")).
		To(UnmarshalInto(&status), "Failed to get EMQX replicant nodes status")
	g.Expect(status).To(
		And(
			HaveField("ReadyReplicas", BeEquivalentTo(replicantReplicas)),
			HaveField("CurrentReplicas", BeEquivalentTo(replicantReplicas)),
			HaveField("UpdateReplicas", BeEquivalentTo(replicantReplicas)),
		),
		"EMQX status does not have expected number of replicant nodes",
	)
	checkReplicantNodesStatusRevision(g, status, replicantReplicas)
}

func checkReplicantNodesStatusRevision(g Gomega, status crdv2.ReplicantNodesStatus, replicas int) {
	var podList corev1.PodList
	g.Expect(status).To(
		And(
			HaveField("CurrentRevision", Not(BeEmpty())),
			HaveField("UpdateRevision", Not(BeEmpty())),
		),
		"EMQX replicant nodes status does not have expected revision",
	)
	g.Expect(status.CurrentRevision).To(
		Equal(status.UpdateRevision),
		"EMQX replicant nodes current and update revisions are different",
	)
	g.Expect(KubectlOut("get", "pods",
		"--selector", crdv2.LabelPodTemplateHash+"="+status.CurrentRevision,
		"--field-selector", "status.phase==Running",
		"-o", "json",
	)).To(UnmarshalInto(&podList), "Failed to list replicant pods")
	g.Expect(podList.Items).To(
		HaveLen(replicas),
		"EMQX cluster does not have %d current revision replicant pods", replicas,
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
