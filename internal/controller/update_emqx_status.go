package controller

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	emperror "emperror.dev/errors"
	crdv2 "github.com/emqx/emqx-operator/api/v2"
	"github.com/emqx/emqx-operator/internal/emqx/api"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type updateStatus struct {
	*EMQXReconciler
}

func (u *updateStatus) reconcile(r *reconcileRound, instance *crdv2.EMQX) subResult {
	status := &instance.Status

	// Core: count pods on each revision for rolling update progress.
	coreSet := r.state.coreSet()
	status.CoreNodesStatus.UpdatedReplicas = 0
	status.CoreNodesStatus.CurrentReplicas = 0
	for _, pod := range r.state.podsManagedBy(r.state.coreSet()) {
		if r.state.partOfCoreSetRevision(pod, coreSet.Status.UpdateRevision) {
			status.CoreNodesStatus.UpdatedReplicas++
		}
		if r.state.partOfCoreSetRevision(pod, coreSet.Status.CurrentRevision) {
			status.CoreNodesStatus.CurrentReplicas++
		}
	}

	// Replicant: multi-ReplicaSet pattern retained.
	currentReplicantSet, updateReplicantSet := switchReplicantSet(r, instance)

	status.ReplicantNodesStatus.ReadyReplicas = 0
	if currentReplicantSet != nil {
		status.ReplicantNodesStatus.CurrentReplicas = currentReplicantSet.Status.Replicas
	}
	if updateReplicantSet != nil {
		status.ReplicantNodesStatus.UpdateReplicas = updateReplicantSet.Status.Replicas
	}

	req := r.oldestCoreRequester()

	// check emqx node status
	if req != nil {
		nodes, err := api.Nodes(req)
		if err != nil {
			return subResult{err: emperror.Wrap(err, "failed to get node status")}
		}
		u.updateEMQXNodesStatus(r, instance, nodes)
	}

	status.CoreNodesStatus.ReadyReplicas = 0
	for _, node := range status.CoreNodes {
		if node.Status == "running" {
			status.CoreNodesStatus.ReadyReplicas++
		}
	}
	for _, node := range status.ReplicantNodes {
		if node.Status == "running" {
			status.ReplicantNodesStatus.ReadyReplicas++
		}
	}

	if req != nil {
		clusterEvacuationsStatus, err := api.ClusterEvacuationStatus(req)
		if err == nil {
			status.NodeEvacuations = []crdv2.NodeEvacuationStatus{}
			for _, ns := range clusterEvacuationsStatus {
				status.NodeEvacuations = append(status.NodeEvacuations, crdv2.NodeEvacuationStatus{
					NodeName:               ns.Node,
					State:                  ns.State,
					SessionRecipients:      ns.SessionRecipients,
					SessionEvictionRate:    ns.SessionEvictionRate,
					ConnectionEvictionRate: ns.ConnectionEvictionRate,
					// Stats
					InitialSessions:    ns.Stats.InitialSessions,
					InitialConnections: ns.Stats.InitialConnected,
				})
			}
		} else {
			return subResult{err: emperror.Wrap(err, "failed to get node evacuation status")}
		}
	}

	// Reflect the status of the DS replication in the resource status.
	var dsReplicationStatus api.DSReplicationStatus
	if req != nil {
		var err error
		dsReplicationStatus, err = api.GetDSReplicationStatus(req)
		if err != nil {
			return subResult{err: emperror.Wrap(err, "failed to get DS replication status")}
		}
	}
	if len(dsReplicationStatus.DBs) > 0 {
		status.DSReplication.DBs = make([]crdv2.DSDBReplicationStatus, len(dsReplicationStatus.DBs))
	}
	for i, db := range dsReplicationStatus.DBs {
		minReplicas := 0
		maxReplicas := 0
		numTransitions := 0
		numShardReplicas := 0
		lostShardReplicas := 0
		if len(db.Shards) > 0 {
			minReplicas = len(db.Shards[0].Replicas)
			maxReplicas = len(db.Shards[0].Replicas)
		}
		for _, shard := range db.Shards {
			minReplicas = min(minReplicas, len(shard.Replicas))
			maxReplicas = max(maxReplicas, len(shard.Replicas))
			numTransitions += len(shard.Transitions)
			numShardReplicas += len(shard.Replicas)
			for _, replica := range shard.Replicas {
				if replica.Status == "lost" {
					lostShardReplicas += 1
				}
			}
		}
		status.DSReplication.DBs[i] = crdv2.DSDBReplicationStatus{
			Name:              db.Name,
			NumShards:         int32(len(db.Shards)),
			NumShardReplicas:  int32(numShardReplicas),
			LostShardReplicas: int32(lostShardReplicas),
			NumTransitions:    int32(numTransitions),
			MinReplicas:       int32(minReplicas),
			MaxReplicas:       int32(maxReplicas),
		}
	}

	// update status condition
	evaluateStatusConditions(r.state, instance)

	if err := u.Client.Status().Update(r.ctx, instance); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to update status")}
	}
	return subResult{}
}

// evaluateStatusConditions evaluates all conditions independently from current state.
func evaluateStatusConditions(s *reconcileState, instance *crdv2.EMQX) {
	evaluateCoreNodesProgressing(s, instance)
	evaluateReplicantNodesProgressing(s, instance)
	evaluateAvailable(s, instance)
	evaluateReady(s, instance)
}

func evaluateCoreNodesProgressing(s *reconcileState, instance *crdv2.EMQX) {
	cond := crdv2.CoreNodesProgressing
	status := &instance.Status

	coreSet := s.coreSet()
	if coreSet == nil {
		status.SetCondition(cond, metav1.ConditionTrue, "Create", "spinning up core set")
		return
	}

	desired := instance.Spec.NumCoreReplicas()
	updated := coreSet.Status.UpdatedReplicas
	total := coreSet.Status.Replicas

	switch {
	case total < desired:
		status.SetCondition(cond, metav1.ConditionTrue, "ScalingUp",
			fmt.Sprintf("%d/%d core pods", total, desired))
	case total > desired:
		status.SetCondition(cond, metav1.ConditionTrue, "ScalingDown",
			fmt.Sprintf("%d/%d core pods", total, desired))
	case updated < desired:
		status.SetCondition(cond, metav1.ConditionTrue, "RollingUpdate",
			fmt.Sprintf("%d/%d core pods updated", updated, desired))
	default:
		status.SetCondition(cond, metav1.ConditionFalse, "Converged",
			fmt.Sprintf("%d core pods up to date", desired))
	}
}

func forceCoreNodesProgressing(instance *crdv2.EMQX) {
	instance.Status.SetCondition(crdv2.CoreNodesProgressing, metav1.ConditionTrue, "RollingUpdate",
		"0 core pods updated")
	instance.Status.SetCondition(crdv2.Ready, metav1.ConditionFalse, "CoreNodesProgressing",
		"Core nodes are progressing")
}

func evaluateReplicantNodesProgressing(s *reconcileState, instance *crdv2.EMQX) {
	cond := crdv2.ReplicantNodesProgressing
	status := &instance.Status

	if !instance.Spec.HasReplicants() {
		status.RemoveCondition(crdv2.ReplicantNodesProgressing)
		return
	}

	updateSet := s.updateReplicantSet(instance)
	if updateSet == nil {
		status.SetCondition(cond, metav1.ConditionTrue, "Create", "spinning up replicant set")
		return
	}

	desired := instance.Spec.NumReplicantReplicas()
	currentRevision := status.ReplicantNodesStatus.CurrentRevision
	updateRevision := status.ReplicantNodesStatus.UpdateRevision
	total := updateSet.Status.Replicas

	switch {
	case currentRevision != updateRevision:
		currentSet := s.currentReplicantSet(instance)
		status.SetCondition(cond, metav1.ConditionTrue, "RollingUpdate",
			fmt.Sprintf("%d/%d replicant pods updated", total-*currentSet.Spec.Replicas, total))
	case total > desired:
		status.SetCondition(cond, metav1.ConditionTrue, "ScalingDown",
			fmt.Sprintf("%d/%d replicant pods", total, desired))
	case total < desired:
		status.SetCondition(cond, metav1.ConditionTrue, "ScalingUp",
			fmt.Sprintf("%d/%d replicant pods", total, desired))
	default:
		status.SetCondition(cond, metav1.ConditionFalse, "Converged",
			fmt.Sprintf("%d replicant pods up to date", desired))
	}
}

func forceReplicantNodesProgressing(instance *crdv2.EMQX) {
	instance.Status.SetCondition(crdv2.ReplicantNodesProgressing, metav1.ConditionTrue, "RollingUpdate",
		"0 replicant pods updated")
	instance.Status.SetCondition(crdv2.Ready, metav1.ConditionFalse, "ReplicantNodesProgressing",
		"Replicant nodes are progressing")
}

func evaluateAvailable(s *reconcileState, instance *crdv2.EMQX) {
	cond := crdv2.Available
	status := &instance.Status
	if instance.Spec.HasReplicants() {
		replicantSet := s.updateReplicantSet(instance)
		desired := instance.Spec.NumReplicantReplicas()
		available := int32(0)
		if replicantSet != nil {
			available = s.numAvailablePods(replicantSet, instance)
		}
		if available >= desired {
			status.SetCondition(cond, metav1.ConditionTrue, "ReplicantPodsAvailable",
				fmt.Sprintf("%d/%d replicant pods available", available, desired))
		} else {
			status.SetCondition(cond, metav1.ConditionFalse, "ReplicantPodsUnavailable",
				fmt.Sprintf("%d/%d replicant pods available", available, desired))
		}
	} else {
		coreSet := s.coreSet()
		desired := instance.Spec.NumCoreReplicas()
		available := int32(0)
		if coreSet != nil {
			available = s.numAvailablePods(coreSet, instance)
		}
		if available >= desired {
			status.SetCondition(cond, metav1.ConditionTrue, "CorePodsAvailable",
				fmt.Sprintf("%d/%d core pods available", available, desired))
		} else {
			status.SetCondition(cond, metav1.ConditionFalse, "CorePodsUnavailable",
				fmt.Sprintf("%d/%d core pods available", available, desired))
		}
	}
}

func evaluateReady(s *reconcileState, instance *crdv2.EMQX) {
	status := &instance.Status

	if !s.areCoresReady(instance) {
		status.SetCondition(crdv2.Ready, metav1.ConditionFalse,
			"CoreNodesProgressing",
			"Core nodes are progressing",
		)
		return
	}

	if instance.Spec.HasReplicants() {
		if !s.areReplicantsReady(instance) {
			status.SetCondition(crdv2.Ready, metav1.ConditionFalse,
				"ReplicantNodesProgressing",
				"Replicant nodes are progressing",
			)
			return
		}
	}

	status.SetCondition(crdv2.Ready, metav1.ConditionTrue, "Ready", "Cluster is ready")
}

func switchReplicantSet(
	r *reconcileRound,
	instance *crdv2.EMQX,
) (*appsv1.ReplicaSet, *appsv1.ReplicaSet) {
	current := r.state.currentReplicantSet(instance)
	update := r.state.updateReplicantSet(instance)
	if (current == nil || current.Status.Replicas == 0) && update != nil {
		current = nil
		for _, replicantSet := range r.state.replicantSets {
			// Adopt oldest non-empty replicantSet if there are more than 2 (current and update) replicantSets:
			if replicantSet.UID != update.UID && replicantSet.Status.Replicas > 0 {
				r.log.V(1).Info("adopting non-empty current replicantSet", "replicaSet", klog.KObj(replicantSet))
				current = replicantSet
				break
			}
		}
		if current == nil {
			r.log.V(1).Info("switching update -> current replicantSet", "replicaSet", klog.KObj(update))
			current = update
		}
	}
	if current != nil {
		instance.Status.ReplicantNodesStatus.CurrentRevision = current.Labels[crdv2.LabelPodTemplateHash]
	}
	return current, update
}

func (u *updateStatus) updateEMQXNodesStatus(r *reconcileRound, instance *crdv2.EMQX, nodes []api.EMQXNode) {
	status := &instance.Status
	status.CoreNodes = []crdv2.EMQXNode{}
	status.ReplicantNodes = []crdv2.EMQXNode{}
	slices.SortFunc(nodes, func(a, b api.EMQXNode) int {
		// Use seconds granularity to avoid jitter in ordering
		asec := a.Uptime / 1000
		bsec := b.Uptime / 1000
		if asec == bsec {
			return cmp.Compare(a.Node, b.Node)
		}
		return cmp.Compare(asec, bsec)
	})
	for _, n := range nodes {
		node := crdv2.EMQXNode{
			Name:        n.Node,
			Status:      n.NodeStatus,
			OTPRelease:  n.OTPRelease,
			Version:     n.Version,
			Role:        n.Role,
			Sessions:    n.Connections,
			Connections: n.LiveConnections,
		}
		list := &status.CoreNodes
		host := parseNodeName(n.Node, instance).hostName
		if node.Role == "replicant" {
			list = &status.ReplicantNodes
		}
		for _, pod := range r.state.pods {
			if node.Role == "core" && strings.HasPrefix(host, pod.Name) {
				node.PodName = pod.Name
				break
			}
			if node.Role == "replicant" && host == pod.Status.PodIP {
				node.PodName = pod.Name
				break
			}
		}
		*list = append(*list, node)
	}
}
