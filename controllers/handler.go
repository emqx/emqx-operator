package controllers

import (
	"fmt"

	"github.com/emqx/emqx-operator/api/v1alpha1"
	"github.com/emqx/emqx-operator/pkg/cache"
	"github.com/emqx/emqx-operator/pkg/client/k8s"
	"github.com/emqx/emqx-operator/pkg/service"
	"github.com/emqx/emqx-operator/pkg/util"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	defaultLabels = map[string]string{}
)

// EmqxClusterHandler is the EMQ X Cluster handler. This handler will create the required
// resources that a EMQ X Cluster needs.
type EmqxClusterHandler struct {
	k8sServices k8s.Services
	eService    service.EmqxClusterClient
	// TODO
	// eChecker    service.EmqxClusterCheck
	eventsCli k8s.Event
	logger    logr.Logger
	metaCache *cache.MetaMap
}

// Do will ensure the EMQ X Cluster is in the expected state and update the EMQ X Cluster status.
func (ech *EmqxClusterHandler) Do(e *v1alpha1.Emqx) error {
	ech.logger.WithValues("namespace", e.Namespace, "name", e.Name).Info("handler doing")
	if err := e.Validate(); err != nil {
		// TODO
		// metrics.ClusterMetrics.SetClusterError(e.Namespace, e.Name)
		return err
	}

	// diff new and new EMQ X Cluster, then update status
	meta := ech.metaCache.Cache(e)
	ech.logger.WithValues("namespace", e.Namespace, "name", e.Name).V(3).
		Info(fmt.Sprintf("meta status:%s, mes:%s, state:%s", meta.Status, meta.Message, meta.State))
	ech.updateStatus(meta)

	// Create owner refs so the objects manager by this handler have ownership to the
	// received rc.
	oRefs := ech.createOwnerReferences(e)

	// Create the labels every object derived from this need to have.
	labels := ech.getLabels(e)

	ech.logger.WithValues("namespace", e.Namespace, "name", e.Name).V(2).Info("Ensure...")
	ech.eventsCli.EnsureCluster(e)
	if err := ech.Ensure(meta.Obj, labels, oRefs); err != nil {
		ech.eventsCli.FailedCluster(e, err.Error())
		e.Status.SetFailedCondition(err.Error())
		ech.k8sServices.UpdateCluster(e.Namespace, e)
		// TODO
		// metrics.ClusterMetrics.SetClusterError(e.Namespace, e.Name)
		return err
	}

	ech.logger.WithValues("namespace", e.Namespace, "name", e.Name).V(2).Info("CheckAndHeal...")
	ech.eventsCli.CheckCluster(e)

	// TODO
	// if err := ech.CheckAndHeal(meta); err != nil {
	// 	metrics.ClusterMetrics.SetClusterError(e.Namespace, e.Name)
	// 	if err.Error() != needRequeueMsg {
	// 		ech.eventsCli.FailedCluster(e, err.Error())
	// 		ech.Status.SetFailedCondition(err.Error())
	// 		ech.k8sServices.UpdateCluster(e.Namespace, e)
	// 		return err
	// 	}
	// 	// if user delete statefulset or deployment, set status
	// 	status := e.Status.Conditions
	// 	if len(status) > 0 && status[0].Type == v1alpha1.ClusterConditionHealthy {
	// 		ech.eventsCli.CreateCluster(e)
	// 		ech.Status.SetCreateCondition("emqx server be removed by user, restart")
	// 		ech.k8sServices.UpdateCluster(e.Namespace, e)
	// 	}
	// 	return err
	// }

	ech.logger.WithValues("namespace", e.Namespace, "name", e.Name).V(2).Info("SetReadyCondition...")
	ech.eventsCli.HealthCluster(e)
	e.Status.SetReadyCondition("Cluster ok")
	ech.k8sServices.UpdateCluster(e.Namespace, e)
	// TODO
	// metrics.ClusterMetrics.SetClusterOK(e.Namespace, e.Name)

	return nil
}

func (ech *EmqxClusterHandler) updateStatus(meta *cache.Meta) {
	e := meta.Obj

	if meta.State != cache.Check {
		switch meta.Status {
		case v1alpha1.ClusterConditionCreating:
			ech.eventsCli.CreateCluster(e)
			e.Status.SetCreateCondition(meta.Message)
		case v1alpha1.ClusterConditionScaling:
			ech.eventsCli.NewNodeAdd(e, meta.Message)
			e.Status.SetScalingUpCondition(meta.Message)
		case v1alpha1.ClusterConditionScalingDown:
			ech.eventsCli.NodeRemove(e, meta.Message)
			e.Status.SetScalingDownCondition(meta.Message)
		case v1alpha1.ClusterConditionUpgrading:
			ech.eventsCli.UpdateCluster(e, meta.Message)
			e.Status.SetUpgradingCondition(meta.Message)
		default:
			ech.eventsCli.UpdateCluster(e, meta.Message)
			e.Status.SetUpdatingCondition(meta.Message)
		}
		ech.k8sServices.UpdateCluster(e.Namespace, e)
	}
}

// getLabels merges all the labels (dynamic and operator static ones).
func (ech *EmqxClusterHandler) getLabels(e *v1alpha1.Emqx) map[string]string {
	dynLabels := map[string]string{}
	return util.MergeLabels(defaultLabels, dynLabels, e.Labels)
}

func (ech *EmqxClusterHandler) createOwnerReferences(e *v1alpha1.Emqx) []metav1.OwnerReference {
	egvk := v1alpha1.VersionKind(v1alpha1.Kind)
	return []metav1.OwnerReference{
		*metav1.NewControllerRef(e, egvk),
	}
}
