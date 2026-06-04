package controller

import (
	"fmt"
	"strings"

	crdv2 "github.com/emqx/emqx-operator/api/v2"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	api "github.com/emqx/emqx-operator/internal/emqx/api"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

// Responsibilities:
// - Removes EMQX cores from the cluster that should no longer be a member, because of scale-down.
type syncClusterMembership struct {
	*EMQXReconciler
}

func (s *syncClusterMembership) reconcile(r *reconcileRound, instance *crdv2.EMQX) subResult {
	req := r.oldestCoreRequester()
	if req == nil {
		return subResult{}
	}

	staleNodes := []*crdv2.EMQXNode{}
	for _, node := range instance.Status.CoreNodes {
		if node.Status != api.NodeStatusStopped {
			continue
		}
		if pod := r.state.podWithName(node.PodName); pod != nil {
			continue
		}
		if s.keepCoreNode(r, instance, &node) {
			continue
		}
		staleNodes = append(staleNodes, &node)
	}

	for _, staleNode := range staleNodes {
		err := api.ForceLeave(req, staleNode.Name)
		if err == nil {
			s.EventRecorder.Event(
				instance,
				corev1.EventTypeNormal,
				"NodeForceLeave",
				fmt.Sprintf("Stale %s node %s force-left the cluster", staleNode.Role, staleNode.Name),
			)
		} else {
			return subResult{err: err}
		}
	}

	return subResult{}
}

func (*syncClusterMembership) keepCoreNode(r *reconcileRound, instance *crdv2.EMQX, node *crdv2.EMQXNode) bool {
	// Determine pod name, either from node status or node name itself:
	podName := node.PodName
	if podName == "" {
		parsed := parseNodeName(node.Name, instance)
		if parsed != nil {
			podName = parsed.podName
		}
	}
	// Find out correspoding core set:
	var owningCoreSet *appsv1.StatefulSet
	for _, coreSet := range r.state.coreSets {
		if strings.HasPrefix(podName, coreSet.Name+"-") {
			if owningCoreSet == nil || len(coreSet.Name) > len(owningCoreSet.Name) {
				owningCoreSet = coreSet
			}
		}
	}
	// Force-leave if no correspoding core set:
	if owningCoreSet == nil {
		return false
	}
	// Keep node if it is in the range of desired number of replicas:
	ordinal := util.PodOrdinal(podName)
	return ordinal < ptr.Deref(owningCoreSet.Spec.Replicas, 1)
}
