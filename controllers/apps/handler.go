package apps

import (
	"context"
	"fmt"

	"github.com/emqx/emqx-operator/apis/apps/v1beta1"
	"github.com/emqx/emqx-operator/pkg/cache"
	"github.com/emqx/emqx-operator/pkg/client/k8s"
	"github.com/emqx/emqx-operator/pkg/service"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	mgr "sigs.k8s.io/controller-runtime/pkg/manager"
)

type EmqxHandler interface {
	Do(emqx v1beta1.Emqx) error
	Ensure(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
}

type Handler struct {
	client    service.Client
	checker   service.Checker
	eventsCli k8s.Event
	logger    logr.Logger
	metaCache *cache.MetaMap
}

func NewHandler(mgr mgr.Manager) *Handler {
	manager := *k8s.NewManager(mgr.GetClient(), log)

	return &Handler{
		client:    *service.NewClient(manager),
		checker:   *service.NewChecker(manager),
		metaCache: new(cache.MetaMap),
		eventsCli: k8s.NewEvent(mgr.GetEventRecorderFor("emqx-operator"), log),
		logger:    log,
	}
}

// Do will ensure the EMQ X Cluster is in the expected state and update the EMQ X Cluster status.
func (handler *Handler) Do(emqx v1beta1.Emqx) error {
	handler.logger.WithValues("namespace", emqx.GetNamespace(), "name", emqx.GetName()).Info("handler doing")
	if err := emqx.Validate(); err != nil {
		// TODO
		// metrics.ClusterMetrics.SetClusterError(emqx.GetNamespace(), emqx.GetName())
		return err
	}

	// diff new and new EMQ X Cluster, then update status
	meta := handler.metaCache.Cache(emqx)
	handler.logger.WithValues("namespace", emqx.GetNamespace(), "name", emqx.GetName()).V(3).
		Info(fmt.Sprintf("meta status:%s, mes:%s, state:%s", meta.Status, meta.Message, meta.State))
	if err := handler.updateStatus(meta); err != nil {
		return err
	}

	// Create owner refs so the objects manager by this handler have ownership to the
	// received rc.
	oRefs := handler.createOwnerReferences(emqx)

	// Create the labels every object derived from this need to have.
	labels := emqx.GetLabels()

	handler.logger.WithValues("namespace", emqx.GetNamespace(), "name", emqx.GetName()).V(2).Info("Ensure...")
	handler.eventsCli.EnsureCluster(emqx)
	if err := handler.Ensure(meta.Obj, labels, oRefs); err != nil {
		handler.eventsCli.FailedCluster(emqx, err.Error())
		emqx.SetFailedCondition(err.Error())
		_ = handler.updateEmqxStatus(emqx)
		// TODO
		// metrics.ClusterMetrics.SetClusterError(emqx.GetNamespace(), emqx.GetName())
		return err
	}

	handler.logger.WithValues("namespace", emqx.GetNamespace(), "name", emqx.GetName()).V(2).Info("CheckAndHeal...")
	handler.eventsCli.CheckCluster(emqx)

	// TODO
	// if err := handler.CheckAndHeal(meta); err != nil {
	// 	metrics.ClusterMetrics.SetClusterError(emqx.GetNamespace(), emqx.GetName())
	// 	if err.Error() != needRequeueMsg {
	// 		handler.eventsCli.FailedCluster(emqx, err.Error())
	// 		handler.Status.SetFailedCondition(err.Error())
	// 		handler.client.UpdateCluster(emqx.GetNamespace(), emqx)
	// 		return err
	// 	}
	// 	// if user delete statefulset or deployment, set status
	// 	status := emqx.Status.Conditions
	// 	if len(status) > 0 && status[0].Type == v1beta1.ClusterConditionHealthy {
	// 		handler.eventsCli.CreateCluster(emqx)
	// 		handler.Status.SetCreateCondition("emqx server be removed by user, restart")
	// 		handler.client.UpdateCluster(emqx.GetNamespace(), emqx)
	// 	}
	// 	return err
	// }

	handler.logger.WithValues("namespace", emqx.GetNamespace(), "name", emqx.GetName()).V(2).Info("SetReadyCondition...")
	handler.eventsCli.HealthCluster(emqx)
	emqx.SetReadyCondition("Cluster ok")
	return handler.updateEmqxStatus(emqx)
}

func (handler *Handler) updateStatus(meta *cache.Meta) error {
	emqx := meta.Obj

	if meta.State != cache.Check {
		switch meta.Status {
		case v1beta1.ClusterConditionCreating:
			handler.eventsCli.CreateCluster(emqx)
			emqx.SetCreateCondition(meta.Message)
		case v1beta1.ClusterConditionScaling:
			handler.eventsCli.NewNodeAdd(emqx, meta.Message)
			emqx.SetScalingUpCondition(meta.Message)
		case v1beta1.ClusterConditionScalingDown:
			handler.eventsCli.NodeRemove(emqx, meta.Message)
			emqx.SetScalingDownCondition(meta.Message)
		case v1beta1.ClusterConditionUpgrading:
			handler.eventsCli.UpdateCluster(emqx, meta.Message)
			emqx.SetUpgradingCondition(meta.Message)
		default:
			handler.eventsCli.UpdateCluster(emqx, meta.Message)
			emqx.SetUpdatingCondition(meta.Message)
		}
		return handler.updateEmqxStatus(emqx)
	}
	return nil
}

func (handler *Handler) createOwnerReferences(emqx v1beta1.Emqx) []metav1.OwnerReference {
	emqxGroupVersionKind := v1beta1.VersionKind(emqx.GetKind())
	return []metav1.OwnerReference{
		*metav1.NewControllerRef(emqx, emqxGroupVersionKind),
	}
}

func (handler *Handler) updateEmqxStatus(emqx v1beta1.Emqx) error {
	if emqxBroker, ok := emqx.(*v1beta1.EmqxBroker); ok {
		return handler.client.EmqxBroker.UpdateStatus(emqxBroker)
	}
	if emqxEnterprise, ok := emqx.(*v1beta1.EmqxEnterprise); ok {
		return handler.client.EmqxEnterprise.UpdateStatus(emqxEnterprise)
	}
	return nil
}
