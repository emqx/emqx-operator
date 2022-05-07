package apps

import (
	"context"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	"github.com/emqx/emqx-operator/pkg/manager"
	"github.com/go-logr/logr"
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
