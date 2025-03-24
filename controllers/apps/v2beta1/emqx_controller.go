/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v2beta1

import (
	"context"
		"time"

	emperror "emperror.dev/errors"
	config "github.com/emqx/emqx-operator/controllers/apps/v2beta1/config"
	innerErr "github.com/emqx/emqx-operator/internal/errors"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
		"k8s.io/client-go/kubernetes"
		"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
		"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	"github.com/emqx/emqx-operator/internal/handler"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
)

// subResult provides a wrapper around different results from a subreconciler.
type subResult struct {
	err    error
	result ctrl.Result
}

type subReconciler interface {
	reconcile(ctx context.Context, logger logr.Logger, instance *appsv2beta1.EMQX, r innerReq.RequesterInterface) subResult
}

// EMQXReconciler reconciles a EMQX object
type EMQXReconciler struct {
	*handler.Handler
	conf          *config.Conf
	Clientset     *kubernetes.Clientset
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
}

func NewEMQXReconciler(mgr manager.Manager) *EMQXReconciler {
	return &EMQXReconciler{
		Handler:       handler.NewHandler(mgr),
		Clientset:     kubernetes.NewForConfigOrDie(mgr.GetConfig()),
		Scheme:        mgr.GetScheme(),
		EventRecorder: mgr.GetEventRecorderFor("emqx-controller"),
	}
}

//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EMQX object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *EMQXReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var err error
	logger := log.FromContext(ctx)

	instance := &appsv2beta1.EMQX{}
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if instance.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
	}

	r.conf, err = config.EMQXConf(config.MergeDefaults(instance.Spec.Config.Data))
	if err != nil {
		r.EventRecorder.Event(instance, corev1.EventTypeWarning, "InvalidConfig", "the .spec.config.data is not a valid HOCON config")
		return ctrl.Result{}, emperror.Wrap(err, "failed to parse config")
	}

	requester, err := newRequester(ctx, r.Client, instance, r.conf)
	if err != nil {
		if k8sErrors.IsNotFound(emperror.Cause(err)) {
			_ = (&addBootstrap{r}).reconcile(ctx, logger, instance, nil)
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		return ctrl.Result{}, emperror.Wrap(err, "failed to get bootstrap user")
	}

	for _, subReconciler := range []subReconciler{
		&addBootstrap{r},
		&updatePodConditions{r},
		&updateStatus{r},
&syncConfig{r},
		&addHeadlessSvc{r},
		&addCore{r},
		&addRepl{r},
		&addPdb{r},
				&addSvc{r},
		&updatePodConditions{r},
		&updateStatus{r},
&dsUpdateReplicaSets{r},
		&dsReflectPodCondition{r},
		&syncPods{r},
		&syncSets{r},
	} {
		subResult := subReconciler.reconcile(ctx, logger, instance, requester)
		if !subResult.result.IsZero() {
			return subResult.result, nil
		}
		if subResult.err != nil {
			if innerErr.IsCommonError(subResult.err) {
				logger.V(1).Info("requeue reconcile", "reconciler", subReconciler, "reason", subResult.err)
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			r.EventRecorder.Event(instance, corev1.EventTypeWarning, "ReconcilerFailed", emperror.Cause(subResult.err).Error())
			return ctrl.Result{}, subResult.err
		}
	}

	isStable := instance.Status.IsConditionTrue(appsv2beta1.Ready) && instance.Status.DSReplication.IsStable()
	if !isStable {
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}
	return ctrl.Result{RequeueAfter: time.Duration(30) * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EMQXReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv2beta1.EMQX{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				// Ignore updates to CR status in which case metadata.Generation does not change
				return e.ObjectNew.GetGeneration() != e.ObjectOld.GetGeneration()
			},
		}).
		Complete(r)
}
