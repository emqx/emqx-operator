package v2beta1

import (
	"context"

	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type syncSets struct {
	*EMQXReconciler
}

func (s *syncSets) reconcile(ctx context.Context, instance *appsv2beta1.EMQX, r innerReq.RequesterInterface) subResult {
	if !instance.Status.IsConditionTrue(appsv2beta1.Ready) {
		return subResult{}
	}
	logger := log.FromContext(ctx)

	_, _, oldRsList := getReplicaSetList(ctx, s.Client, instance)
	rsDiff := int32(len(oldRsList)) - *instance.Spec.RevisionHistoryLimit
	if rsDiff > 0 {
		for i := 0; i < int(rsDiff); i++ {
			rs := oldRsList[i].DeepCopy()
			// Avoid delete replica set with non-zero replica counts
			if rs.Status.Replicas != 0 || *(rs.Spec.Replicas) != 0 || rs.Generation > rs.Status.ObservedGeneration || rs.DeletionTimestamp != nil {
				continue
			}
			logger.Info("trying to cleanup replica set for EMQX", "replicaSet", klog.KObj(rs), "EMQX", klog.KObj(instance))
			if err := s.Client.Delete(ctx, rs); err != nil && !k8sErrors.IsNotFound(err) {
				return subResult{err: err}
			}
		}
	}

	_, _, oldStsList := getStateFulSetList(ctx, s.Client, instance)
	stsDiff := int32(len(oldStsList)) - *instance.Spec.RevisionHistoryLimit
	if stsDiff > 0 {
		for i := 0; i < int(stsDiff); i++ {
			sts := oldStsList[i].DeepCopy()
			// Avoid delete stateful set with non-zero replica counts
			if sts.Status.Replicas != 0 || *(sts.Spec.Replicas) != 0 || sts.Generation > sts.Status.ObservedGeneration || sts.DeletionTimestamp != nil {
				continue
			}
			logger.Info("trying to cleanup stateful set for EMQX", "statefulSet", klog.KObj(sts), "EMQX", klog.KObj(instance))
			if err := s.Client.Delete(ctx, sts); err != nil && !k8sErrors.IsNotFound(err) {
				return subResult{err: err}
			}

			// Delete PVCs
			pvcList := &corev1.PersistentVolumeClaimList{}
			_ = s.Client.List(ctx, pvcList,
				client.InNamespace(instance.Namespace),
				client.MatchingLabels(sts.Spec.Selector.MatchLabels),
			)

			for _, p := range pvcList.Items {
				pvc := p.DeepCopy()
				if pvc.DeletionTimestamp != nil {
					continue
				}
				logger.Info("trying to cleanup pvc for EMQX", "pvc", klog.KObj(pvc), "EMQX", klog.KObj(instance))
				if err := s.Client.Delete(ctx, pvc); err != nil && !k8sErrors.IsNotFound(err) {
					return subResult{err: err}
				}
			}
		}
	}

	return subResult{}
}
