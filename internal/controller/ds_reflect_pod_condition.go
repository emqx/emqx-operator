package controller

import (
	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v3alpha1"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type dsReflectPodCondition struct {
	*EMQXReconciler
}

func (u *dsReflectPodCondition) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	// If DS cluster state is not loaded, skip the reconciliation step.
	if r.dsCluster == nil {
		return subResult{}
	}

	for _, pod := range r.state.pods {
		node := instance.Status.FindNodeByPodName(pod.Name)
		if node == nil {
			continue
		}
		condition := corev1.PodCondition{
			Type:               crd.DSReplicationSite,
			Status:             corev1.ConditionUnknown,
			LastTransitionTime: metav1.Now(),
		}
		site := r.dsCluster.FindSite(node.Name)
		if site != nil {
			if len(site.Shards) > 0 {
				condition.Status = corev1.ConditionTrue
			} else {
				condition.Status = corev1.ConditionFalse
			}
		} else {
			// No DS site for the node.
			// This is unlikely to happen, but should be safe to assume the node is not
			// a DS replication site.
			condition.Status = corev1.ConditionFalse
		}
		existing := util.FindPodCondition(pod, crd.DSReplicationSite)
		if existing == nil || existing.Status != condition.Status {
			err := util.UpdatePodCondition(r.ctx, u.Client, pod, condition)
			if err != nil {
				return subResult{err: emperror.Wrapf(err, "failed to update pod %s status", pod.Name)}
			}
		}
	}

	return subResult{}
}
