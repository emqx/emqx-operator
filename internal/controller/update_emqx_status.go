package controller

import (
	"cmp"
	"slices"
	"strings"

	emperror "emperror.dev/errors"
	crdv2 "github.com/emqx/emqx-operator/api/v2"
	"github.com/emqx/emqx-operator/internal/emqx/api"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
)

type updateStatus struct {
	*EMQXReconciler
}

func (u *updateStatus) reconcile(r *reconcileRound, instance *crdv2.EMQX) subResult {
	status := u.inheritStatus(instance)
	hasReplicants := instance.Spec.HasReplicants() || r.state.hasReplicants()

	currentCoreSet, updateCoreSet := switchCoreSet(r, instance)
	var currentReplicantSet, updateReplicantSet *appsv1.ReplicaSet
	if hasReplicants {
		currentReplicantSet, updateReplicantSet = switchReplicantSet(r, instance)
	}

	if currentCoreSet != nil {
		status.CoreNodesStatus.CurrentRevision = currentCoreSet.Labels[crdv2.LabelPodTemplateHash]
		status.CoreNodesStatus.CurrentReplicas = currentCoreSet.Status.Replicas
	}
	if updateCoreSet != nil {
		status.CoreNodesStatus.UpdateRevision = updateCoreSet.Labels[crdv2.LabelPodTemplateHash]
		status.CoreNodesStatus.UpdateReplicas = updateCoreSet.Status.Replicas
	}
	if hasReplicants {
		if currentReplicantSet != nil {
			status.ReplicantNodesStatus.CurrentRevision = currentReplicantSet.Labels[crdv2.LabelPodTemplateHash]
			status.ReplicantNodesStatus.CurrentReplicas = currentReplicantSet.Status.Replicas
		}
		if updateReplicantSet != nil {
			status.ReplicantNodesStatus.UpdateRevision = updateReplicantSet.Labels[crdv2.LabelPodTemplateHash]
			status.ReplicantNodesStatus.UpdateReplicas = updateReplicantSet.Status.Replicas
		}
	}

	req := r.oldestCoreRequester()

	// check emqx node status
	if req != nil {
		nodes, err := api.Nodes(req)
		if err != nil {
			return subResult{err: emperror.Wrap(err, "failed to get node status")}
		}
		u.updateEMQXNodesStatus(r, &status, nodes)
	}
	for _, node := range status.CoreNodes {
		if node.Status == "running" {
			status.CoreNodesStatus.ReadyReplicas++
		}
	}
	if hasReplicants {
		for _, node := range status.ReplicantNodes {
			if node.Status == "running" {
				status.ReplicantNodesStatus.ReadyReplicas++
			}
		}
	}
	if req != nil {
		clusterEvacuationsStatus, err := api.ClusterEvacuationStatus(req)
		if err == nil {
			for _, ns := range clusterEvacuationsStatus {
				status.NodeEvacuationsStatus = append(status.NodeEvacuationsStatus, crdv2.NodeEvacuationStatus{
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
	instance.Status = status
	u.updateStatusCondition(r, instance)

	if err := u.Client.Status().Update(r.ctx, instance); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to update status")}
	}
	return subResult{}
}

// inheritStatus builds a minimal EMQXStatus with information that has to be
// preserved across reconcile rounds.
func (u *updateStatus) inheritStatus(instance *crdv2.EMQX) crdv2.EMQXStatus {
	status := crdv2.EMQXStatus{
		Conditions: u.inheritConditions(instance),
		CoreNodesStatus: crdv2.EMQXNodesStatus{
			Replicas:       ptr.Deref(instance.Spec.CoreTemplate.Spec.Replicas, 1),
			CollisionCount: instance.Status.CoreNodesStatus.CollisionCount,
		},
	}
	if instance.Spec.HasReplicants() {
		status.ReplicantNodesStatus = crdv2.EMQXNodesStatus{
			Replicas:       ptr.Deref(instance.Spec.ReplicantTemplate.Spec.Replicas, 1),
			CollisionCount: instance.Status.ReplicantNodesStatus.CollisionCount,
		}
	}
	return status
}

func (*updateStatus) inheritConditions(instance *crdv2.EMQX) []metav1.Condition {
	hasReplicants := instance.Spec.HasReplicants()
	if hasReplicants {
		return instance.Status.Conditions
	}
	next := make([]metav1.Condition, 0, len(instance.Status.Conditions))
	for _, condition := range instance.Status.Conditions {
		if condition.Type == crdv2.ReplicantNodesProgressing || condition.Type == crdv2.ReplicantNodesReady {
			continue
		}
		next = append(next, condition)
	}
	return next
}

func (u *updateStatus) updateStatusCondition(r *reconcileRound, instance *crdv2.EMQX) {
	status := &instance.Status

	condition := status.GetLastTrueCondition()
	if condition == nil {
		instance.Status.SetTrueCondition(crdv2.Initialized)
		u.updateStatusCondition(r, instance)
		return
	}

	switch condition.Type {

	case crdv2.Initialized:
		updateSts := r.state.updateCoreSet(instance)
		if updateSts != nil {
			u.statusTransition(r, instance, crdv2.CoreNodesProgressing)
		}

	case crdv2.CoreNodesProgressing:
		updateSts := r.state.updateCoreSet(instance)
		if updateSts != nil &&
			updateSts.Status.ReadyReplicas > 0 &&
			updateSts.Status.ReadyReplicas == status.CoreNodesStatus.UpdateReplicas {
			u.statusTransition(r, instance, crdv2.CoreNodesReady)
		}

	case crdv2.CoreNodesReady:
		if instance.Spec.HasReplicants() {
			u.statusTransition(r, instance, crdv2.ReplicantNodesProgressing)
		} else {
			u.statusTransition(r, instance, crdv2.Available)
		}

	case crdv2.ReplicantNodesProgressing:
		if instance.Spec.HasReplicants() {
			updateRs := r.state.updateReplicantSet(instance)
			if updateRs != nil &&
				updateRs.Status.ReadyReplicas > 0 &&
				updateRs.Status.ReadyReplicas == status.ReplicantNodesStatus.UpdateReplicas {
				u.statusTransition(r, instance, crdv2.ReplicantNodesReady)
			}
		} else {
			u.resetConditions(r, instance, "NoReplicants")
		}

	case crdv2.ReplicantNodesReady:
		if instance.Spec.HasReplicants() {
			u.statusTransition(r, instance, crdv2.Available)
		} else {
			u.resetConditions(r, instance, "NoReplicants")
		}

	case crdv2.Available:
		if status.CoreNodesStatus.UpdateReplicas != status.CoreNodesStatus.Replicas ||
			status.CoreNodesStatus.ReadyReplicas != status.CoreNodesStatus.Replicas ||
			status.CoreNodesStatus.UpdateRevision != status.CoreNodesStatus.CurrentRevision {
			break
		}

		if instance.Spec.HasReplicants() {
			if status.ReplicantNodesStatus.UpdateReplicas != status.ReplicantNodesStatus.Replicas ||
				status.ReplicantNodesStatus.ReadyReplicas != status.ReplicantNodesStatus.Replicas ||
				status.ReplicantNodesStatus.UpdateRevision != status.ReplicantNodesStatus.CurrentRevision {
				break
			}
		}

		status.SetCondition(metav1.Condition{
			Type:    crdv2.Ready,
			Status:  metav1.ConditionTrue,
			Reason:  crdv2.Ready,
			Message: "Cluster is ready",
		})

	case crdv2.Ready:
		updateSts := r.state.updateCoreSet(instance)
		if updateSts != nil &&
			updateSts.Status.ReadyReplicas != status.CoreNodesStatus.Replicas {
			u.resetConditions(r, instance, "CoreNodesNotReady")
			return
		}

		if instance.Spec.HasReplicants() {
			updateRs := r.state.updateReplicantSet(instance)
			if updateRs != nil &&
				updateRs.Status.ReadyReplicas != status.ReplicantNodesStatus.Replicas {
				u.resetConditions(r, instance, "ReplicantNodesNotReady")
				return
			}
		}
	}
}

func (u *updateStatus) resetConditions(
	r *reconcileRound,
	instance *crdv2.EMQX,
	reason string,
) {
	instance.Status.ResetConditions(reason)
	u.updateStatusCondition(r, instance)
}

func (u *updateStatus) statusTransition(
	r *reconcileRound,
	instance *crdv2.EMQX,
	conditionType string,
) {
	instance.Status.SetTrueCondition(conditionType)
	u.updateStatusCondition(r, instance)
}

func switchCoreSet(
	r *reconcileRound,
	instance *crdv2.EMQX,
) (*appsv1.StatefulSet, *appsv1.StatefulSet) {
	current := r.state.currentCoreSet(instance)
	update := r.state.updateCoreSet(instance)
	if (current == nil || current.Status.Replicas == 0) && update != nil {
		current = nil
		for _, coreSet := range r.state.coreSets {
			// Adopt oldest non-empty coreSet if there are more than 2 (current and update) coreSets:
			if coreSet.UID != update.UID && coreSet.Status.Replicas > 0 {
				r.log.V(1).Info("adopting non-empty current coreSet", "statefulSet", klog.KObj(coreSet))
				current = coreSet
				break
			}
		}
		if current == nil {
			r.log.V(1).Info("switching update -> current coreSet", "statefulSet", klog.KObj(update))
			current = update
		}
	}
	return current, update
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
	return current, update
}

func (u *updateStatus) updateEMQXNodesStatus(
	r *reconcileRound,
	status *crdv2.EMQXStatus,
	nodes []api.EMQXNode,
) {
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
		host := extractHostname(n.Node)
		if node.Role == crdv2.RoleReplicant {
			list = &status.ReplicantNodes
		}
		for _, pod := range r.state.pods {
			if node.Role == crdv2.RoleCore && strings.HasPrefix(host, pod.Name) {
				node.PodName = pod.Name
				break
			}
			if node.Role == crdv2.RoleReplicant && host == pod.Status.PodIP {
				node.PodName = pod.Name
				break
			}
		}
		*list = append(*list, node)
	}
}

func extractHostname(node string) string {
	// Example: emqx@emqx-core-557c8b7684-0.emqx-headless.default.svc.cluster.local
	// Example: emqx@10.244.0.23
	return strings.Split(node[strings.Index(node, "@")+1:], ":")[0]
}
