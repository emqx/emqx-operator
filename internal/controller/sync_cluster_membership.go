package controller

import (
	"fmt"

	crdv2 "github.com/emqx/emqx-operator/api/v2"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	api "github.com/emqx/emqx-operator/internal/emqx/api"
	corev1 "k8s.io/api/core/v1"
)

// Responsibilities:
// - Removes EMQX cores from the cluster that should no longer be a member, because of scale-down.
// - Removes EMQX replicants that no longer have a corresponding pod.
type syncClusterMembership struct {
	*EMQXReconciler
}

func (s *syncClusterMembership) reconcile(r *reconcileRound, instance *crdv2.EMQX) subResult {
	// Instantiate API requester.
	req := r.oldestCoreRequester()
	if req == nil {
		return reconcilePostpone()
	}

	staleNodes := []*crdv2.EMQXNode{}
	for _, node := range instance.Status.CoreNodes {
		// Running cores / cores still having respective pods should not be force-left:
		if node.Status != "stopped" || node.PodName != "" || r.state.podWithName(node.PodName) != nil {
			continue
		}
		// Cores with node name having pod ordinal under desired number of replicas should not be force-left:
		nodeName := parseNodeName(node.Name, instance)
		ordinal := util.PodOrdinal(nodeName.podName)
		if ordinal < int(instance.Spec.NumCoreReplicas()) {
			continue
		}
		// Cores having higher pod ordinals should be force-left:
		staleNodes = append(staleNodes, &node)
	}

	for _, node := range instance.Status.ReplicantNodes {
		// Running replicants / replicants still having respective pods should not be force-left:
		if node.Status != "stopped" || node.PodName != "" || r.state.podWithName(node.PodName) != nil {
			continue
		}
		// Stopped replicants w/o respective pods should be force-left:
		staleNodes = append(staleNodes, &node)
	}

	for _, staleNode := range staleNodes {
		err := api.ForceLeave(r.oldestCoreRequester(), staleNode.Name)
		if err == nil {
			s.EventRecorder.Event(
				instance,
				corev1.EventTypeNormal,
				"NodeForceLeave",
				fmt.Sprintf("Stale %s node %s force-left the cluster", staleNode.Role, staleNode.Name),
			)
		} else {
			return reconcileError(err)
		}
	}

	return subResult{}
}
