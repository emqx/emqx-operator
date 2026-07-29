package controller

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v3beta1"
	"github.com/emqx/emqx-operator/internal/emqx/api"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
)

type updateStatus struct {
	*EMQXReconciler
}

func (u *updateStatus) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	status := &instance.Status

	// Core: count pods on each revision for rolling update progress.
	coreSet := r.state.coreSet()
	status.CoreReplicas = 0
	status.CoreSelector = labels.Set(instance.DefaultLabelsWith(crd.CoreLabels())).String()
	status.CoreNodesStatus.UpdatedReplicas = 0
	status.CoreNodesStatus.CurrentReplicas = 0
	if coreSet != nil {
		status.CoreReplicas = coreSet.Status.Replicas
		for _, pod := range r.state.podsManagedBy(coreSet) {
			if r.state.partOfCoreSetRevision(pod, coreSet.Status.UpdateRevision) {
				status.CoreNodesStatus.UpdatedReplicas++
			}
			if r.state.partOfCoreSetRevision(pod, coreSet.Status.CurrentRevision) {
				status.CoreNodesStatus.CurrentReplicas++
			}
		}
	}

	status.ReplicantReplicas = 0
	status.ReplicantSelector = ""
	if instance.Spec.HasReplicants() {
		status.ReplicantReplicas = r.state.numReplicants()
		status.ReplicantSelector = labels.Set(instance.DefaultLabelsWith(crd.ReplicantLabels())).String()
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
		if node.Status == api.NodeStatusRunning {
			status.CoreNodesStatus.ReadyReplicas++
		}
	}
	for _, node := range status.ReplicantNodes {
		if node.Status == api.NodeStatusRunning {
			status.ReplicantNodesStatus.ReadyReplicas++
		}
	}

	if req != nil {
		clusterEvacuationsStatus, err := api.ClusterEvacuationStatus(req)
		if err == nil {
			status.NodeEvacuations = []crd.NodeEvacuationStatus{}
			for _, ns := range clusterEvacuationsStatus {
				status.NodeEvacuations = append(status.NodeEvacuations, crd.NodeEvacuationStatus{
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
		status.DSReplication.DBs = make([]crd.DSDBReplicationStatus, len(dsReplicationStatus.DBs))
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
		status.DSReplication.DBs[i] = crd.DSDBReplicationStatus{
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
func evaluateStatusConditions(s *reconcileState, instance *crd.EMQX) {
	evaluateCoreNodesProgressing(s, instance)
	evaluateReplicantNodesProgressing(s, instance)
	evaluateAvailable(s, instance)
	evaluateReady(s, instance)
}

func evaluateCoreNodesProgressing(s *reconcileState, instance *crd.EMQX) {
	cond := crd.CoreNodesProgressing
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

func forceCoreNodesProgressing(instance *crd.EMQX) {
	instance.Status.SetCondition(crd.CoreNodesProgressing, metav1.ConditionTrue, "RollingUpdate",
		"0 core pods updated")
	instance.Status.SetCondition(crd.Ready, metav1.ConditionFalse, "CoreNodesProgressing",
		"Core nodes are progressing")
}

func evaluateReplicantNodesProgressing(s *reconcileState, instance *crd.EMQX) {
	cond := crd.ReplicantNodesProgressing
	status := &instance.Status

	if !instance.Spec.HasReplicants() {
		status.RemoveCondition(crd.ReplicantNodesProgressing)
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
		currentReplicas := total
		if currentSet != nil && currentSet.Spec.Replicas != nil {
			currentReplicas = *currentSet.Spec.Replicas
		}
		status.SetCondition(cond, metav1.ConditionTrue, "RollingUpdate",
			fmt.Sprintf("%d/%d replicant pods updated", total-currentReplicas, total))
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

func forceReplicantNodesProgressing(instance *crd.EMQX) {
	instance.Status.SetCondition(crd.ReplicantNodesProgressing, metav1.ConditionTrue, "RollingUpdate",
		"0 replicant pods updated")
	instance.Status.SetCondition(crd.Ready, metav1.ConditionFalse, "ReplicantNodesProgressing",
		"Replicant nodes are progressing")
}

func evaluateAvailable(s *reconcileState, instance *crd.EMQX) {
	cond := crd.Available
	status := &instance.Status
	if instance.Spec.HasReplicants() {
		desired := instance.Spec.NumReplicantReplicas()
		available := s.numAvailableReplicants()
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
			available = coreSet.Status.AvailableReplicas
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

func evaluateReady(s *reconcileState, instance *crd.EMQX) {
	status := &instance.Status

	if !evaluateCoresReady(s, instance) {
		status.SetCondition(crd.Ready, metav1.ConditionFalse,
			"CoreNodesProgressing",
			"Core nodes are progressing",
		)
		return
	}

	if instance.Spec.HasReplicants() {
		if !evaluateReplicantsReady(s, instance) {
			status.SetCondition(crd.Ready, metav1.ConditionFalse,
				"ReplicantNodesProgressing",
				"Replicant nodes are progressing",
			)
			return
		}
	}

	if !instance.Status.DSReplication.IsStable() {
		status.SetCondition(crd.Ready, metav1.ConditionFalse,
			"DSReplicationProgressing",
			"Durable storage membership transitions are in progress",
		)
		return
	}

	status.SetCondition(crd.Ready, metav1.ConditionTrue, "Ready", "Cluster is ready")
}

func evaluateCoresReady(r *reconcileState, instance *crd.EMQX) bool {
	desired := instance.Spec.NumCoreReplicas()
	coreSet := r.coreSet()
	coresReady := int32(0)
	coresUpdated := int32(0)
	coresTotal := int32(0)
	nodesTotal := int32(len(instance.Status.CoreNodes))
	nodesReady := instance.Status.CoreNodesStatus.ReadyReplicas
	if coreSet != nil {
		coresTotal = coreSet.Status.Replicas
		coresReady = coreSet.Status.ReadyReplicas
		coresUpdated = coreSet.Status.UpdatedReplicas
	}
	return coresTotal == desired && coresReady == desired && coresUpdated == desired &&
		nodesTotal == desired && nodesReady == desired
}

func evaluateReplicantsReady(s *reconcileState, instance *crd.EMQX) bool {
	desired := instance.Spec.NumReplicantReplicas()
	replicantSet := s.updateReplicantSet(instance)
	replicantsTotal := int32(0)
	replicantsReady := int32(0)
	nodesTotal := int32(len(instance.Status.ReplicantNodes))
	nodesReady := instance.Status.ReplicantNodesStatus.ReadyReplicas
	if replicantSet != nil {
		replicantsTotal = replicantSet.Status.Replicas
		replicantsReady = replicantSet.Status.ReadyReplicas
	}
	return replicantsTotal == desired && replicantsReady == desired &&
		nodesTotal == desired && nodesReady == desired
}

func switchReplicantSet(
	r *reconcileRound,
	instance *crd.EMQX,
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
		instance.Status.ReplicantNodesStatus.CurrentRevision = current.Labels[crd.LabelPodTemplateHash]
	}
	return current, update
}

func (u *updateStatus) updateEMQXNodesStatus(r *reconcileRound, instance *crd.EMQX, nodes []api.EMQXNode) {
	status := &instance.Status
	status.CoreNodes = []crd.EMQXNode{}
	status.ReplicantNodes = []crd.EMQXNode{}
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
		node := crd.EMQXNode{
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
		if node.Role == crd.RoleReplicant {
			list = &status.ReplicantNodes
		}
		for _, pod := range r.state.pods {
			if node.Role == crd.RoleCore && strings.HasPrefix(host, pod.Name) {
				node.PodName = pod.Name
				break
			}
			if node.Role == crd.RoleReplicant && host == pod.Status.PodIP {
				node.PodName = pod.Name
				break
			}
		}
		*list = append(*list, node)
	}
}
