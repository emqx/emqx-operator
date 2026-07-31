package controller

import (
	"reflect"
	"sort"
	"strings"

	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v3beta1"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	api "github.com/emqx/emqx-operator/internal/emqx/api"
)

type dsUpdateReplicaSets struct {
	*EMQXReconciler
}

func (u *dsUpdateReplicaSets) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
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
	// Prefer EMQX 6.x requester first: EMQX starting from 6.0.0 has separate cluster view.
	req := r.requester.forCore(r.state, &podsWithEMQXVersion{instance: instance, prefix: "6."})
	if req == nil {
		req = r.preferredCoreRequester()
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
			return reconcileError(emperror.Errorf("unrecognized DS site node name: %s", site.Node))
		}
		// Exclude non-core nodes:
		if !strings.HasPrefix(nodeName.podName, instance.CoreName()) {
			continue
		}
		// Exclude retiring core nodes:
		pod := r.state.podWithName(nodeName.podName)
		ordinal := util.PodOrdinal(nodeName.podName)
		if pod != nil && isPodScaleDownRetiring(pod) {
			continue
		}
		// Exclude unrecognizable core pods:
		if ordinal < 0 {
			continue
		}
		// Include core nodes matching desired number of replicas:
		if ordinal < int(desiredReplicas) {
			targetSites = append(targetSites, site.ID)
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
