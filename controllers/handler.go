package controllers

import (
	"fmt"

	"github.com/emqx/emqx-operator/api/v1alpha2"
	"github.com/emqx/emqx-operator/pkg/cache"
	"github.com/emqx/emqx-operator/pkg/client/k8s"
	"github.com/emqx/emqx-operator/pkg/service"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EmqxClusterHandler is the EMQ X Cluster handler. This handler will create the required
// resources that a EMQ X Cluster needs.
type EmqxClusterHandler struct {
	k8sServices k8s.Services
	eService    service.EmqxClusterClient
	eChecker    service.EmqxClusterCheck
	eventsCli   k8s.Event
	logger      logr.Logger
	metaCache   *cache.MetaMap
}

// Do will ensure the EMQ X Cluster is in the expected state and update the EMQ X Cluster status.
func (ech *EmqxClusterHandler) Do(emqx v1alpha2.Emqx) error {
	ech.logger.WithValues("namespace", emqx.GetNamespace(), "name", emqx.GetName()).Info("handler doing")
	if err := emqx.Validate(); err != nil {
		// TODO
		// metrics.ClusterMetrics.SetClusterError(emqx.GetNamespace(), emqx.GetName())
		return err
	}

	// diff new and new EMQ X Cluster, then update status
	meta := ech.metaCache.Cache(emqx)
	ech.logger.WithValues("namespace", emqx.GetNamespace(), "name", emqx.GetName()).V(3).
		Info(fmt.Sprintf("meta status:%s, mes:%s, state:%s", meta.Status, meta.Message, meta.State))
	if err := ech.updateStatus(meta); err != nil {
		return err
	}

	// Create owner refs so the objects manager by this handler have ownership to the
	// received rc.
	oRefs := ech.createOwnerReferences(emqx)

	// Create the labels every object derived from this need to have.
	labels := emqx.GetLabels()

	ech.logger.WithValues("namespace", emqx.GetNamespace(), "name", emqx.GetName()).V(2).Info("Ensure...")
	ech.eventsCli.EnsureCluster(emqx)
	if err := ech.Ensure(meta.Obj, labels, oRefs); err != nil {
		ech.eventsCli.FailedCluster(emqx, err.Error())
		emqx.SetFailedCondition(err.Error())
		_ = ech.k8sServices.UpdateCluster(emqx.GetNamespace(), emqx)
		// TODO
		// metrics.ClusterMetrics.SetClusterError(emqx.GetNamespace(), emqx.GetName())
		return err
	}

	ech.logger.WithValues("namespace", emqx.GetNamespace(), "name", emqx.GetName()).V(2).Info("CheckAndHeal...")
	ech.eventsCli.CheckCluster(emqx)

	// TODO
	// if err := ech.CheckAndHeal(meta); err != nil {
	// 	metrics.ClusterMetrics.SetClusterError(emqx.GetNamespace(), emqx.GetName())
	// 	if err.Error() != needRequeueMsg {
	// 		ech.eventsCli.FailedCluster(emqx, err.Error())
	// 		ech.Status.SetFailedCondition(err.Error())
	// 		ech.k8sServices.UpdateCluster(emqx.GetNamespace(), emqx)
	// 		return err
	// 	}
	// 	// if user delete statefulset or deployment, set status
	// 	status := emqx.Status.Conditions
	// 	if len(status) > 0 && status[0].Type == v1alpha2.ClusterConditionHealthy {
	// 		ech.eventsCli.CreateCluster(emqx)
	// 		ech.Status.SetCreateCondition("emqx server be removed by user, restart")
	// 		ech.k8sServices.UpdateCluster(emqx.GetNamespace(), emqx)
	// 	}
	// 	return err
	// }

	ech.logger.WithValues("namespace", emqx.GetNamespace(), "name", emqx.GetName()).V(2).Info("SetReadyCondition...")
	ech.eventsCli.HealthCluster(emqx)
	emqx.SetReadyCondition("Cluster ok")
	err := ech.k8sServices.UpdateCluster(emqx.GetNamespace(), emqx)
	if err != nil {
		return err
	}
	// TODO
	// metrics.ClusterMetrics.SetClusterOK(emqx.GetNamespace(), emqx.GetName())

	return nil
}

func (ech *EmqxClusterHandler) updateStatus(meta *cache.Meta) error {
	emqx := meta.Obj

	if meta.State != cache.Check {
		switch meta.Status {
		case v1alpha2.ClusterConditionCreating:
			ech.eventsCli.CreateCluster(emqx)
			emqx.SetCreateCondition(meta.Message)
		case v1alpha2.ClusterConditionScaling:
			ech.eventsCli.NewNodeAdd(emqx, meta.Message)
			emqx.SetScalingUpCondition(meta.Message)
		case v1alpha2.ClusterConditionScalingDown:
			ech.eventsCli.NodeRemove(emqx, meta.Message)
			emqx.SetScalingDownCondition(meta.Message)
		case v1alpha2.ClusterConditionUpgrading:
			ech.eventsCli.UpdateCluster(emqx, meta.Message)
			emqx.SetUpgradingCondition(meta.Message)
		default:
			ech.eventsCli.UpdateCluster(emqx, meta.Message)
			emqx.SetUpdatingCondition(meta.Message)
		}
		return ech.k8sServices.UpdateCluster(emqx.GetNamespace(), emqx)
	}
	return nil
}

func (ech *EmqxClusterHandler) createOwnerReferences(emqx v1alpha2.Emqx) []metav1.OwnerReference {
	emqxGroupVersionKind := v1alpha2.VersionKind(emqx.GetKind())
	return []metav1.OwnerReference{
		*metav1.NewControllerRef(emqx, emqxGroupVersionKind),
	}
}
