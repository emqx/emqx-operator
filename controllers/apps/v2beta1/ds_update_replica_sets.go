package v2beta1

import (
	"context"
	"reflect"
	"sort"
	"strconv"
	"strings"

	emperror "emperror.dev/errors"
	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	ds "github.com/emqx/emqx-operator/controllers/apps/v2beta1/ds"
	req "github.com/emqx/emqx-operator/internal/requester"
	"github.com/go-logr/logr"
)

type dsUpdateReplicaSets struct {
	*EMQXReconciler
}

func (u *dsUpdateReplicaSets) reconcile(
	ctx context.Context,
	logger logr.Logger,
	instance *appsv2beta1.EMQX,
	r req.RequesterInterface,
) subResult {
	// If there's no EMQX API to query, skip the reconciliation.
	if r == nil {
		return subResult{}
	}

	// If DS is not enabled, skip this reconciliation step.
	// Rely on EMQX API to check: in a mixed-release cluster actual configuration might not
	// reflect the state of the cluster well enough. What really matters is if DS API is
	// available for further operations.
	enabled, err := ds.IsDSEnabled(r)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to check if DS is enabled")}
	}
	if !enabled {
		return subResult{}
	}

	// Get the most recent stateful set.
	updateSts, _, _ := getStateFulSetList(ctx, u.Client, instance)
	if updateSts == nil {
		return subResult{}
	}

	// Wait until all pods are ready.
	desiredReplicas := instance.Status.CoreNodesStatus.Replicas
	if updateSts.Status.AvailableReplicas < desiredReplicas {
		return subResult{}
	}

	// Fetch the DS cluster info.
	cluster, err := ds.GetCluster(r)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to fetch DS cluster status")}
	}

	// Fetch the DS replication status.
	replication, err := ds.GetReplicationStatus(r)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to fetch DS cluster status")}
	}

	// Compute the current sites.
	currentSites := replication.TargetSites()

	// Compute the target sites.
	targetSites := []string{}
	for _, node := range instance.Status.CoreNodes {
		if node.ControllerUID == updateSts.UID {
			site := cluster.FindSite(node.Node)
			if site == nil {
				return subResult{err: emperror.Wrapf(err, "no site for node %s", node.Node)}
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
	for _, db := range replication.DBs {
		err := ds.UpdateReplicaSet(r, db.Name, targetSites)
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
