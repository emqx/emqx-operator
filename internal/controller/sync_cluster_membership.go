package controller

import (
	"fmt"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
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

func (s *syncClusterMembership) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	// Instantiate API requester.
	req := r.oldestCoreRequester()
	if req == nil {
		return reconcilePostpone()
	}

	desiredReplicas := int(instance.Spec.NumCoreReplicas())
	staleNodes := []*crd.EMQXNode{}
	for _, node := range instance.Status.CoreNodes {
		// Skip: node is running.
		if node.Status != api.NodeStatusStopped {
			continue
		}
		pod := r.state.podWithName(node.PodName)
		nodeName := parseNodeName(node.Name, instance)
		retiring := pod != nil && isPodScaleDownRetiring(pod)
		ordinal := util.PodOrdinal(node.PodName)
		if ordinal < 0 && nodeName != nil {
			ordinal = util.PodOrdinal(nodeName.podName)
		}
		// Skip: running non-retiring cores should not be force-left.
		if !retiring && pod != nil {
			continue
		}
		// Skip: non-retiring cores with pod ordinal under desired number of replicas should not
		// be force-left. If ordinal is -1, stay on the safe side and let the user decide.
		if !retiring && ordinal < desiredReplicas {
			continue
		}
		// Cores having higher pod ordinals should be force-left:
		staleNodes = append(staleNodes, &node)
	}

	for _, node := range instance.Status.ReplicantNodes {
		// Running replicants / replicants still having respective pods should not be force-left:
		if node.Status != api.NodeStatusStopped {
			continue
		}
		pod := r.state.podWithName(node.PodName)
		if pod != nil {
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
