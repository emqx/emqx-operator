/*
Copyright 2025-2026.

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

package controller

import (
	"context"
	"reflect"
	"time"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	"github.com/emqx/emqx-operator/internal/emqx/api"
	req "github.com/emqx/emqx-operator/internal/requester"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/emqx/emqx-operator/internal/errors"
	"github.com/emqx/emqx-operator/internal/handler"
)

// Currently executing round of reconciliation
type reconcileRound struct {
	ctx context.Context
	log logr.Logger
	// Populated by `loadConfig` reconciler:
	conf *config.EMQX
	// Populated by `setupAPIRequester` reconciler:
	requester apiRequester
	// Populated by loadState reconciler:
	state *reconcileState
	// Populated by dsLoadClusterState reconciler:
	dsCluster     *api.DSCluster
	dsReplication *api.DSReplicationStatus
}

// Instantiate default API requester for a core node.
// Picks oldest core node that is considered ready: up and running, not evacuating.
func (r *reconcileRound) oldestCoreRequester() req.RequesterInterface {
	return r.requester.forOldestCore(r.state, &podsWithCondition{corev1.ContainersReady})
}

// subResult provides a wrapper around different results from a subreconciler.
type subResult struct {
	err error
	// If `true`, short timeout requeue is needed:
	needRequeue bool
	// Immediately report controller `Result` if not nil:
	immediateResult *ctrl.Result
}

func reconcileError(err error) subResult {
	return subResult{err: err}
}

func reconcileRequeue() subResult {
	return subResult{immediateResult: &ctrl.Result{Requeue: true}}
}

func reconcileRequeueAfter(duration time.Duration) subResult {
	return subResult{immediateResult: &ctrl.Result{RequeueAfter: duration}}
}

func reconcilePostpone() subResult {
	return subResult{needRequeue: true}
}

type subReconciler interface {
	reconcile(*reconcileRound, *crd.EMQX) subResult
}

func subReconcilerName(s subReconciler) string {
	return reflect.TypeOf(s).Elem().Name()
}

// EMQXReconciler reconciles a EMQX object
type EMQXReconciler struct {
	*handler.Handler
	RESTConfig    *rest.Config
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
}

func NewEMQXReconciler(mgr manager.Manager) *EMQXReconciler {
	restConfig := mgr.GetConfig()
	_ = kubernetes.NewForConfigOrDie(restConfig)
	return &EMQXReconciler{
		Handler:       handler.NewHandler(mgr.GetClient()),
		RESTConfig:    restConfig,
		Scheme:        mgr.GetScheme(),
		EventRecorder: mgr.GetEventRecorderFor("emqx-controller"),
	}
}

// +kubebuilder:rbac:groups=apps.emqx.io,resources=emqxes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.emqx.io,resources=emqxes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.emqx.io,resources=emqxes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *EMQXReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	instance := &crd.EMQX{}
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if instance.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
	}

	logger := log.FromContext(ctx)
	round := reconcileRound{ctx: ctx, log: logger}
	needRequeue := false

	for _, subReconciler := range []subReconciler{
		// Load EMQX configuration defined in the spec.config.data:
		&loadConfig{r},
		// Load the current state of the resources managed by the controller:
		&loadState{r},
		// Setup secrets with bootstrap API keys / node cookie:
		&addBootstrap{r},
		// Set up API requester builder for the current round:
		&setupAPIRequester{r},
		// Perform reconciliation steps:
		&updateStatus{r},
		&syncConfig{r},
		&addHeadlessService{r},
		&addCoreSet{r},
		&addReplicantSet{r},
		&addPdb{r},
		&addService{r},
		&dsLoadClusterState{r},
		&dsCleanupSites{r},
		&dsUpdateReplicaSets{r},
		&dsReflectPodCondition{r},
		&syncCoreSet{r},
		&syncReplicantSets{r},
		&syncClusterMembership{r},
		&retireCorePods{r},
		&cleanupOutdatedSets{r},
	} {
		round.log = logger.WithValues("reconciler", subReconcilerName(subReconciler))
		subResult := subReconciler.reconcile(&round, instance)
		needRequeue = needRequeue || subResult.needRequeue
		if subResult.err != nil {
			if errors.IsCommonError(subResult.err) {
				round.log.Info("reconciler requeue", "reason", subResult.err)
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			r.EventRecorder.Eventf(instance, corev1.EventTypeWarning,
				"ReconcilerFailed",
				"reconcile failed at step %s, reason: %s", subReconcilerName(subReconciler), subResult.err.Error(),
			)
			return ctrl.Result{}, subResult.err
		}
		if subResult.immediateResult != nil {
			return *subResult.immediateResult, nil
		}
	}

	if !instance.Status.IsConditionTrue(crd.Ready) || needRequeue {
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	return ctrl.Result{RequeueAfter: time.Duration(30) * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EMQXReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&crd.EMQX{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Named("emqx").
		Complete(r)
}
