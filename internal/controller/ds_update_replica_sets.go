package controller

import (
	"reflect"
	"sort"
	"strings"

	emperror "emperror.dev/errors"
	crdv2 "github.com/emqx/emqx-operator/api/v2"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	api "github.com/emqx/emqx-operator/internal/emqx/api"
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
	// Prefer EMQX 6.x requester first: EMQX starting from 6.1.0 has separate cluster view.
	req := r.requester.forOldestCore(r.state, &emqxVersionFilter{instance: instance, prefix: "6."})
	if req == nil {
		req = r.oldestCoreRequester()
	}

	// If there's no EMQX API to query, skip the reconciliation.
	if req == nil {
		return reconcilePostpone()
	}

	// If there are no known DS DBs, skip the reconciliation.
	if len(r.dsReplication.DBs) == 0 {
		return subResult{}
	}

	// Wait until all pods are ready.
	desiredReplicas := instance.Spec.NumCoreReplicas()
	if coreSet.Status.AvailableReplicas < desiredReplicas {
		return reconcilePostpone()
	}

	// Compute the current sites.
	currentSites := r.dsReplication.TargetSites()

	// Compute the target sites.
	targetSites := []string{}
	for _, site := range r.dsCluster.Sites {
		nodeName := parseNodeName(site.Node, instance)
		if nodeName == nil {
			return subResult{err: emperror.Errorf("unrecognized DS site node name: %s", site.Node)}
		}
		if strings.HasPrefix(nodeName.podName, instance.CoreName()) {
			ordinal := util.PodOrdinal(nodeName.podName)
			if ordinal >= 0 && ordinal < int(desiredReplicas) {
				targetSites = append(targetSites, site.ID)
			}
		}
	}

	// No target sites.
	if len(targetSites) == 0 {
		return subResult{}
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
