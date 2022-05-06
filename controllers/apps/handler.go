package apps

import (
	"context"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	"github.com/emqx/emqx-operator/pkg/manager"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
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
	eventsCli record.EventRecorder
	logger    logr.Logger
}

func NewHandler(mgr mgr.Manager) *Handler {
	return &Handler{
		client:    mgr.GetClient(),
		executor:  *manager.NewExecutor(mgr.GetConfig()),
		eventsCli: mgr.GetEventRecorderFor("emqx-operator"),
		logger:    log,
	}
}

// Do will ensure the EMQX Cluster is in the expected state and update the EMQX Cluster status.
func (handler *Handler) Do(emqx v1beta3.Emqx) error {
	handler.logger.WithValues("namespace", emqx.GetNamespace(), "name", emqx.GetName()).V(2).Info("Ensure...")
	if err := handler.Ensure(emqx); err != nil {
		handler.eventsCli.Event(emqx, corev1.EventTypeWarning, "Ensure", err.Error())
		emqx.SetFailedCondition(err.Error())
		_ = handler.updateEmqxStatus(emqx)
		return err
	}

	emqx.SetRunningCondition("Cluster ok")
	return handler.updateEmqxStatus(emqx)
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
