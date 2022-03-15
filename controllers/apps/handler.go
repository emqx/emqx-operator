package apps

import (
	"context"
	"fmt"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	"github.com/emqx/emqx-operator/pkg/cache"
	"github.com/emqx/emqx-operator/pkg/manager"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	mgr "sigs.k8s.io/controller-runtime/pkg/manager"
)

type EmqxHandler interface {
	Do(emqx v1beta3.Emqx) error
	Ensure(emqx v1beta3.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
}

type Handler struct {
	client    client.Client
	executor  manager.Executor
	eventsCli manager.Event
	logger    logr.Logger
	metaCache *cache.MetaMap
}

func NewHandler(mgr mgr.Manager) *Handler {
	return &Handler{
		client:    mgr.GetClient(),
		executor:  *manager.NewExecutor(mgr.GetConfig()),
		metaCache: new(cache.MetaMap),
		eventsCli: manager.NewEvent(mgr.GetEventRecorderFor("emqx-operator")),
		logger:    log,
	}
}

// Do will ensure the EMQX Cluster is in the expected state and update the EMQX Cluster status.
func (handler *Handler) Do(emqx v1beta3.Emqx) error {
	// diff new and new EMQX Cluster, then update status
	meta := handler.metaCache.Cache(emqx)
	handler.logger.WithValues("namespace", emqx.GetNamespace(), "name", emqx.GetName()).V(3).
		Info(fmt.Sprintf("meta status:%s, mes:%s, state:%s", meta.Status, meta.Message, meta.State))
	if err := handler.updateStatus(meta); err != nil {
		return err
	}

	handler.logger.WithValues("namespace", emqx.GetNamespace(), "name", emqx.GetName()).V(2).Info("Ensure...")
	handler.eventsCli.EnsureCluster(emqx)
	if err := handler.Ensure(meta.Obj); err != nil {
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
	// 	if len(status) > 0 && status[0].Type == v1beta3.ClusterConditionHealthy {
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
		case v1beta3.ClusterConditionCreating:
			handler.eventsCli.CreateCluster(emqx)
			emqx.SetCreateCondition(meta.Message)
		case v1beta3.ClusterConditionScaling:
			handler.eventsCli.NewNodeAdd(emqx, meta.Message)
			emqx.SetScalingUpCondition(meta.Message)
		case v1beta3.ClusterConditionScalingDown:
			handler.eventsCli.NodeRemove(emqx, meta.Message)
			emqx.SetScalingDownCondition(meta.Message)
		case v1beta3.ClusterConditionUpgrading:
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

func (handler *Handler) updateEmqxStatus(emqx v1beta3.Emqx) error {
	emqx.DescConditionsByTime()
	err := handler.client.Status().Update(context.TODO(), emqx)
	if err != nil {
		handler.logger.WithValues(
			"kind", emqx.GetKind(),
			"apiVersion", emqx.GetAPIVersion(),
			"namespace", emqx.GetNamespace(),
			"name", emqx.GetName(),
			"conditions", emqx.GetConditions(),
		).Error(err, "Update emqx broker status unsuccessfully")
		return err
	}
	handler.logger.WithValues(
		"kind", emqx.GetKind(),
		"apiVersion", emqx.GetAPIVersion(),
		"namespace", emqx.GetNamespace(),
		"name", emqx.GetName(),
		"conditions", emqx.GetConditions(),
	).V(3).Info("Update emqx broker status successfully")
	return nil
}
