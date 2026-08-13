package controller

import (
	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v3beta1"
	"github.com/emqx/emqx-operator/internal/emqx/api"
	"github.com/emqx/emqx-operator/internal/emqx/ctl"
)

// Responsibilities:
// - Forget DS sites that are considered lost and do not have any shards.
type dsCleanupSites struct {
	*EMQXReconciler
}

func (c *dsCleanupSites) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	// If DS cluster state is not loaded, skip the reconciliation.
	if r.dsCluster == nil {
		return subResult{}
	}

	coreSet := r.state.coreSet()
	if coreSet == nil {
		return subResult{}
	}

	// Instantiate API requester for a node that is part of update StatefulSet.
	// Required API operation is available only since EMQX 6.0.0.
	req := r.requester.forCore(r.state, &podsWithEMQXVersion{instance: instance, prefix: "6."})

	lostSites := []*api.DSSite{}
	for _, site := range r.dsCluster.Sites {
		if site.Up || (len(site.Shards) > 0) {
			continue
		}
		node := instance.Status.FindNode(site.Node)
		if node != nil {
			continue
		}
		lostSites = append(lostSites, &site)
	}

	if len(lostSites) == 0 {
		return subResult{}
	}

	forgetSite := func(site *api.DSSite) error {
		return api.ForgetDSSite(req, site.ID)
	}

	// If there's no suitable EMQX API to query, switch to `emqx ctl`.
	if req == nil {
		pods := preferredCorePods(r.state, podsManagedBy{coreSet})
		if len(pods) == 0 {
			r.log.V(1).Info("skipping DS site cleanup", "reason", "no core pods")
			return reconcilePostpone()
		}
		forgetSite = func(site *api.DSSite) error {
			return ctl.Ctl(r.ctx, c.RESTConfig, pods[0], "ds", "forget", site.ID)
		}
	}

	for _, site := range lostSites {
		err := forgetSite(site)
		if err == nil {
			r.log.V(1).Info("cleaned up lost DS site", "site", site)
		} else {
			return subResult{err: emperror.Wrapf(err, "failed to forget DS site %s", site.ID)}
		}
	}

	return subResult{}
}
