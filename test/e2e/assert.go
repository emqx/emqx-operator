/*
Copyright 2026.

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
	crd "github.com/emqx/emqx-operator/api/v3beta1"
	. "github.com/emqx/emqx-operator/test/util"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func EMQXReady(g Gomega, afterTime ...metav1.Time) {
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

func CoresStable(g Gomega, coreReplicas int) {
	var status crd.EMQXStatus
	g.Expect(KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status}")).
		To(UnmarshalInto(&status), "Failed to get EMQX status")
	g.Expect(status).To(
		And(
			HaveField("CoreReplicas", BeEquivalentTo(coreReplicas)),
			HaveField("CoreNodesStatus.ReadyReplicas", BeEquivalentTo(coreReplicas)),
			HaveField("CoreNodesStatus.UpdatedReplicas", BeEquivalentTo(coreReplicas)),
		),
		"EMQX status does not have expected number of core nodes",
	)
	g.Expect(status.CoreNodes).To(
		HaveEach(HaveField("Status", Equal("running"))),
		"EMQX cluster contains stopped nodes",
	)
	g.Expect(status.CoreNodes).To(
		HaveEach(HaveField("PodName", Not(BeEmpty()))),
		"EMQX cluster contains nodes without pods",
	)

	g.Expect(KubectlOut("get", "pod",
		"--selector", "apps.emqx.io/instance=emqx,apps.emqx.io/managed-by=emqx-operator",
		"-o", "json",
	)).To(
		BeUnmarshalledAs(&corev1.PodList{}, HaveField("Items",
			HaveEach(
				HaveField("Status.Conditions", ContainElement(And(
					HaveField("Type", Equal(corev1.PodReady)),
					HaveField("Status", Equal(corev1.ConditionTrue)),
				))),
			),
		)),
		"Not all EMQX pods are ready",
	)

	g.Expect(KubectlOut("get", "pvc",
		"--selector", crd.LabelDBRole+"=core,"+crd.LabelManagedBy+"=emqx-operator",
		"-o", "json",
	)).To(
		BeUnmarshalledAs(&corev1.PersistentVolumeClaimList{}, HaveField("Items", And(
			HaveLen(coreReplicas),
			HaveEach(HaveField("Status.Phase", Equal(corev1.ClaimBound))),
		))),
		"Not all core PVCs are bound",
	)
}

func NoReplicants(g Gomega) {
	g.Expect(KubectlOut("get", "emqx", "emqx",
		"-o", "jsonpath={.status.replicantNodesStatus.currentReplicas}",
	)).To(Equal("0"), "EMQX cluster status has replicant replicas")
	g.Expect(KubectlOut("get", "emqx", "emqx",
		"-o", "jsonpath={.status.replicantNodes}",
	)).To(BeEmpty(), "EMQX cluster status lists replicant nodes")
}

func ReplicantsStable(g Gomega, replicantReplicas int) {
	var status crd.EMQXStatus
	g.Expect(KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status}")).
		To(UnmarshalInto(&status), "Failed to get EMQX status")
	g.Expect(status).To(And(
		HaveField("ReplicantReplicas", BeEquivalentTo(replicantReplicas)),
		HaveField("ReplicantNodesStatus", And(
			HaveField("ReadyReplicas", BeEquivalentTo(replicantReplicas)),
			HaveField("CurrentReplicas", BeEquivalentTo(replicantReplicas)),
			HaveField("UpdateReplicas", BeEquivalentTo(replicantReplicas)),
		))),
		"EMQX status does not have expected number of replicant nodes",
	)
	g.Expect(status).To(
		HaveField("ReplicantNodesStatus", And(
			HaveField("CurrentRevision", Not(BeEmpty())),
			HaveField("UpdateRevision", Not(BeEmpty())),
		)),
		"EMQX replicant nodes status does not have expected revision",
	)
	g.Expect(status.ReplicantNodesStatus.CurrentRevision).To(
		Equal(status.ReplicantNodesStatus.UpdateRevision),
		"EMQX replicant nodes current and update revisions are different",
	)

	g.Expect(KubectlOut("get", "pods",
		"--selector", crd.LabelPodTemplateHash+"="+status.ReplicantNodesStatus.CurrentRevision,
		"--field-selector", "status.phase==Running",
		"-o", "json",
	)).To(
		BeUnmarshalledAs(&corev1.PodList{}, HaveField("Items", HaveLen(replicantReplicas))),
		"EMQX cluster does not have %d current revision replicant pods", replicantReplicas,
	)
}

func DSReplicationStable(g Gomega, coreReplicas int) {
	status := &crd.DSReplicationStatus{}
	replicationFactor := min(3, coreReplicas)
	g.Expect(KubectlOut("get", "emqx", "emqx", "-o", "jsonpath={.status.dsReplication}")).
		To(UnmarshalInto(&status), "Failed to get emqx status")
	g.Expect(status.DBs).NotTo(BeEmpty(), "No DS database is present")
	g.Expect(status.DBs).To(
		HaveEach(And(
			HaveField("Name", Not(BeEmpty())),
			HaveField("NumShards", Not(BeZero())),
			HaveField("NumShardReplicas", Not(BeZero())),
			Satisfy(func(db crd.DSDBReplicationStatus) bool {
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

func DSReplicationHealthy(g Gomega) {
	g.Expect(KubectlOut("exec", "service/emqx-listeners", "--", "emqx", "ctl", "ds", "info")).
		NotTo(
			ContainSubstring("(!)"),
			"EMQX DS replication is not healthy",
		)
}
