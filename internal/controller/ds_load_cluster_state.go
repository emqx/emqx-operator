package controller

import (
	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v3beta1"
	"github.com/emqx/emqx-operator/internal/emqx/api"
)

// Responsibilities:
// - Load DS cluster info and replication status into the reconcile state.
type dsLoadClusterState struct {
	*EMQXReconciler
}

func (c *dsLoadClusterState) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	// Instantiate API requester for a node that is part of the core StatefulSet.
	// Prefer EMQX 6.x requester first: EMQX starting from 6.0.0 has separate cluster view.
	req := r.requester.forCore(r.state, &podsWithEMQXVersion{instance: instance, prefix: "6."})
	if req == nil {
		req = r.preferredCoreRequester()
	}

	// If there's no suitable EMQX API to query, skip the reconciliation.
	if req == nil {
		return reconcilePostpone()
	}

	// If EMQX DS API is not available, fail the reconciliation.
	cluster, err := api.GetDSCluster(req)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to fetch DS cluster status")}
	}

	// Fetch the DS replication status.
	replication, err := api.GetDSReplicationStatus(req)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to fetch DS replication status")}
	}

	r.dsCluster = &cluster
	r.dsReplication = &replication
	return subResult{}
}
