package controller

import (
	emperror "emperror.dev/errors"
	crdv2 "github.com/emqx/emqx-operator/api/v2"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	"github.com/emqx/emqx-operator/internal/emqx/api"
	corev1 "k8s.io/api/core/v1"
)

type updatePodConditions struct {
	*EMQXReconciler
}

func (u *updatePodConditions) reconcile(r *reconcileRound, instance *crdv2.EMQX) subResult {
	for _, pod := range r.state.pods {
		onServingCondition := util.FindPodCondition(pod, crdv2.PodOnServing)
		if onServingCondition == nil {
			onServingCondition = &corev1.PodCondition{
				Type:   crdv2.PodOnServing,
				Status: corev1.ConditionUnknown,
			}
		}

		status := onServingCondition.Status
		if util.IsPodConditionTrue(pod, corev1.ContainersReady) {
			// PodOnServing logic for rolling upgrades:
			// 1. True for latest revision pods (core or replicant) - they are stable and not being evacuated.
			// 2. False for outdated core pods when AvailabilityCheck fails (evacuating).
			// 3. False for replicant pods of the current (outdated) set.
			if r.state.partOfCoreSetLatestRevision(pod) {
				// Latest revision core pod: always serving.
				status = corev1.ConditionTrue
			} else if r.state.partOfUpdateReplicantSet(pod, instance) {
				// Latest revision replicant pod: always serving.
				status = corev1.ConditionTrue
			} else if r.state.partOfCurrentReplicantSet(pod, instance) {
				// Outdated replicant pod: not serving.
				status = corev1.ConditionFalse
			} else {
				// Outdated core pod: check availability (false when evacuating).
				status = api.AvailabilityCheck(r.requester.forPod(pod))
			}
		}

		util.SwitchPodConditionStatus(onServingCondition, status)
		err := util.UpdatePodCondition(r.ctx, u.Client, pod, *onServingCondition)
		if err != nil {
			return subResult{err: emperror.Wrapf(err, "failed to update pod %s status", pod.Name)}
		}
	}
	return subResult{}
}
