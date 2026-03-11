package controller

import (
	"reflect"
	"sort"
	"strconv"
	"strings"

	emperror "emperror.dev/errors"
	crdv2 "github.com/emqx/emqx-operator/api/v2"
	"github.com/emqx/emqx-operator/internal/emqx/api"
)

type dsUpdateReplicaSets struct {
	*EMQXReconciler
}

func (u *dsUpdateReplicaSets) reconcile(r *reconcileRound, instance *crdv2.EMQX) subResult {
	// If DS cluster state is not loaded, skip the reconciliation.
	if r.dsCluster == nil || r.dsReplication == nil {
		return subResult{}
	}

	// Get the single core StatefulSet.
	coreSet := r.state.coreSet()
	if coreSet == nil {
		return subResult{}
	}

	// Instantiate API requester for a node that is part of the core StatefulSet.
	req := r.requester.forOldestCore(r.state, &managedByFilter{coreSet})

	// If there's no EMQX API to query, skip the reconciliation.
	if req == nil {
		return subResult{}
	}

	// If there are no known DS DBs, skip the reconciliation.
	if len(r.dsReplication.DBs) == 0 {
		return subResult{}
	}

	// Wait until all pods are ready.
	desiredReplicas := instance.Spec.NumCoreReplicas()
	if coreSet.Status.AvailableReplicas < desiredReplicas {
		return subResult{}
	}

	// Compute the current sites.
	currentSites := r.dsReplication.TargetSites()

	// Compute the target sites.
	targetSites := []string{}
	for _, node := range instance.Status.CoreNodes {
		if r.state.podWithName(node.PodName) != nil {
			site := r.dsCluster.FindSite(node.Name)
			if site == nil {
				return subResult{err: emperror.Errorf("no site for node %s", node.Name)}
			}
			if getPodIndex(node.PodName) < desiredReplicas {
				targetSites = append(targetSites, site.ID)
			}
		}
	}

	sort.Strings(targetSites)
	sort.Strings(currentSites)

	// Target sites are the same as current sites, no need to update.
	if reflect.DeepEqual(targetSites, currentSites) {
		return subResult{}
	}

	// Update replica sets for each DB.
	r.log.V(1).Info("updating DS replica sets", "targetSites", targetSites, "currentSites", currentSites)
	for _, db := range r.dsReplication.DBs {
		err := api.UpdateDSReplicaSet(req, db.Name, targetSites)
		if err != nil {
			return subResult{err: emperror.Wrapf(err, "failed to update DB %s replica set", db.Name)}
		}
	}

	return subResult{}
}

func getPodIndex(podName string) int32 {
	parts := strings.Split(podName, "-")
	if len(parts) < 2 {
		return -1
	}
	indexPart := parts[len(parts)-1]
	index, err := strconv.Atoi(indexPart)
	if err != nil {
		return -1
	}
	return int32(index)
}
