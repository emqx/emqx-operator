package controller

import (
	crd "github.com/emqx/emqx-operator/api/v3beta1"
	appsv1 "k8s.io/api/apps/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
)

type cleanupOutdatedSets struct {
	*EMQXReconciler
}

func (s *cleanupOutdatedSets) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	// Postpone cleanups until the instance is ready:
	if !instance.Status.IsConditionTrue(crd.Ready) {
		return subResult{}
	}

	// List outdated replicantSets, preserving order by creation timestamp:
	currentRs := r.state.currentReplicantSet(instance)
	updateRs := r.state.updateReplicantSet(instance)
	prevRsList := []*appsv1.ReplicaSet{}
	for _, rs := range r.state.replicantSets {
		if rs.DeletionTimestamp == nil && rs != currentRs && rs != updateRs {
			prevRsList = append(prevRsList, rs)
		}
	}

	rsOutdated := len(prevRsList) - int(instance.Spec.RevisionHistoryLimit)
	for i := 0; i < rsOutdated; i++ {
		rs := prevRsList[i]
		// Avoid delete replica set with non-zero replica counts
		if rs.Status.Replicas != 0 || *(rs.Spec.Replicas) != 0 || rs.Generation > rs.Status.ObservedGeneration {
			continue
		}
		r.log.Info("removing outdated replicantSet", "replicaSet", klog.KObj(rs))
		if err := s.Client.Delete(r.ctx, rs); err != nil && !k8sErrors.IsNotFound(err) {
			return subResult{err: err}
		}
	}

	// With the single-StatefulSet model for cores, there are no outdated core StatefulSets
	// to clean up. The single StatefulSet is updated in place. Legacy StatefulSets from
	// previous operator versions (with hash-suffixed names) will be cleaned up separately
	// if needed.

	return subResult{}
}
