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
)

type updateStatus struct {
	*EMQXReconciler
}

func (u *updateStatus) reconcile(r *reconcileRound, instance *crdv2.EMQX) subResult {
	status := &instance.Status

	status.CoreNodesStatus.Replicas = 1
	if instance.Spec.CoreTemplate.Spec.Replicas != nil {
		status.CoreNodesStatus.Replicas = *instance.Spec.CoreTemplate.Spec.Replicas
	}
	if instance.Spec.ReplicantTemplate != nil && instance.Spec.ReplicantTemplate.Spec.Replicas != nil {
		status.ReplicantNodesStatus.Replicas = *instance.Spec.ReplicantTemplate.Spec.Replicas
	}

	// Core: single StatefulSet, revision tracking from StatefulSet status.
	status.CoreNodesStatus.UpdateReplicas = 0
	status.CoreNodesStatus.CurrentReplicas = 0
	// Count pods on each revision.
	for _, pod := range r.state.podsManagedBy(r.state.coreSet()) {
		if r.state.partOfCoreSetLatestRevision(pod) {
			status.CoreNodesStatus.UpdateReplicas++
		} else {
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
			status.NodeEvacuationsStatus = []crdv2.NodeEvacuationStatus{}
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
	u.updateStatusCondition(r, instance)

	if err := u.Client.Status().Update(r.ctx, instance); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to update status")}
	}
	return subResult{}
}

func (u *updateStatus) updateStatusCondition(r *reconcileRound, instance *crdv2.EMQX) {
	status := &instance.Status

	condition := status.GetLastTrueCondition()
	if condition == nil {
		instance.Status.SetTrueCondition(crdv2.Initialized)
		u.updateStatusCondition(r, instance)
		return
	}

	sts := r.state.coreSet()

	switch condition.Type {

	case crdv2.Initialized:
		if sts != nil {
			u.statusTransition(r, instance, crdv2.CoreNodesProgressing)
		}

	case crdv2.CoreNodesProgressing:
		if sts != nil &&
			sts.Status.ReadyReplicas > 0 &&
			sts.Status.ReadyReplicas == status.CoreNodesStatus.Replicas {
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
		if sts != nil &&
			sts.Status.ReadyReplicas != status.CoreNodesStatus.Replicas {
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
	if !instance.Spec.HasReplicants() {
		instance.Status.RemoveCondition(crdv2.ReplicantNodesProgressing)
		instance.Status.RemoveCondition(crdv2.ReplicantNodesReady)
	}
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
		host := extractHostname(n.Node)
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

func extractHostname(node string) string {
	// Example: emqx@emqx-core-557c8b7684-0.emqx-headless.default.svc.cluster.local
	// Example: emqx@10.244.0.23
	return strings.Split(node[strings.Index(node, "@")+1:], ":")[0]
}
