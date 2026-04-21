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
	"fmt"
	"time"

	emperror "emperror.dev/errors"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	crdv2beta1 "github.com/emqx/emqx-operator/api/v2beta1"
	crd "github.com/emqx/emqx-operator/api/v3beta1"

	config "github.com/emqx/emqx-operator/internal/controller/config"
	"github.com/emqx/emqx-operator/internal/emqx/api"
	req "github.com/emqx/emqx-operator/internal/requester"
)

// RebalanceReconciler reconciles a Rebalance object
type RebalanceReconciler struct {
	Client        client.Client
	Clientset     *kubernetes.Clientset
	EventRecorder record.EventRecorder
}

func NewRebalanceReconciler(mgr manager.Manager) *RebalanceReconciler {
	return &RebalanceReconciler{
		Clientset:     kubernetes.NewForConfigOrDie(mgr.GetConfig()),
		Client:        mgr.GetClient(),
		EventRecorder: mgr.GetEventRecorderFor("rebalance-controller"),
	}
}

// +kubebuilder:rbac:groups=apps.emqx.io,resources=rebalances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.emqx.io,resources=rebalances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.emqx.io,resources=rebalances/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *RebalanceReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	var err error
	var finalizer string = "apps.emqx.io/finalizer"
	var req req.RequesterInterface

	logger := log.FromContext(ctx)
	logger.V(1).Info("Reconcile rebalance")

	rebalance := &crdv2beta1.Rebalance{}
	if err := r.Client.Get(ctx, request.NamespacedName, rebalance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	emqx := &crd.EMQX{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name:      rebalance.Spec.InstanceName,
		Namespace: rebalance.Namespace,
	}, emqx); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		if !rebalance.DeletionTimestamp.IsZero() {
			controllerutil.RemoveFinalizer(rebalance, finalizer)
			return ctrl.Result{}, r.Client.Update(ctx, rebalance)
		}
		_ = rebalance.Status.SetFailed(crdv2beta1.RebalanceCondition{
			Type:    crdv2beta1.RebalanceConditionFailed,
			Status:  corev1.ConditionTrue,
			Message: fmt.Sprintf("EMQX %s is not found", rebalance.Spec.InstanceName),
		})
		return ctrl.Result{}, r.Client.Status().Update(ctx, rebalance)
	}

	// check if emqx is ready
	if !emqx.Status.IsConditionTrue(crd.Ready) {
		// return ctrl.Result{}, emperror.New("EMQX is not ready")
		_ = rebalance.Status.SetFailed(crdv2beta1.RebalanceCondition{
			Type:    crdv2beta1.RebalanceConditionFailed,
			Status:  corev1.ConditionTrue,
			Message: fmt.Sprintf("EMQX %s is not ready", rebalance.Spec.InstanceName),
		})
		return ctrl.Result{}, r.Client.Status().Update(ctx, rebalance)
	}

	conf, err := config.EMQXConfigWithDefaults(emqx.Spec.Config.Data)
	if err != nil {
		return ctrl.Result{}, emperror.New("failed to parse config")
	}

	bootstrapAPIKey, err := getBootstrapAPIKey(ctx, r.Client, emqx)
	if err != nil {
		return ctrl.Result{}, emperror.Wrap(err, "failed to get bootstrap API key")
	}

	requester, err := newAPIRequesterBuilder(conf, bootstrapAPIKey)
	if err != nil {
		return ctrl.Result{}, emperror.Wrap(err, "failed to create EMQX API requester")
	}

	state, err := loadReconcileState(ctx, r.Client, emqx)
	if err != nil {
		return ctrl.Result{}, emperror.New("failed to load reconcile round state")
	}

	req = requester.forOldestCore(state)
	if req == nil {
		return ctrl.Result{}, emperror.New("EMQX API requester unavailable")
	}

	if !rebalance.DeletionTimestamp.IsZero() {
		if rebalance.Status.Phase == crdv2beta1.RebalancePhaseProcessing {
			coordinatorNode := rebalance.Status.RebalanceStates[0].CoordinatorNode
			_ = api.StopRebalance(req, coordinatorNode)
		}
		controllerutil.RemoveFinalizer(rebalance, finalizer)
		return ctrl.Result{}, r.Client.Update(ctx, rebalance)
	}

	if !controllerutil.ContainsFinalizer(rebalance, finalizer) {
		controllerutil.AddFinalizer(rebalance, finalizer)
		if err := r.Client.Update(ctx, rebalance); err != nil {
			return ctrl.Result{}, err
		}
	}

	rebalanceStatusHandler(emqx, rebalance, req)
	if err := r.Client.Status().Update(ctx, rebalance); err != nil {
		return ctrl.Result{}, err
	}

	switch rebalance.Status.Phase {
	case "Failed":
		r.EventRecorder.Event(rebalance, corev1.EventTypeWarning, "Rebalance", "rebalance failed")
		controllerutil.RemoveFinalizer(rebalance, finalizer)
		return ctrl.Result{}, nil
	case "Completed":
		r.EventRecorder.Event(rebalance, corev1.EventTypeNormal, "Rebalance", "rebalance completed")
		controllerutil.RemoveFinalizer(rebalance, finalizer)
		return ctrl.Result{}, nil
	case "Processing":
		r.EventRecorder.Event(rebalance, corev1.EventTypeNormal, "Rebalance", "rebalance is processing")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	default:
		panic("unknown rebalance phase")
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *RebalanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&crdv2beta1.Rebalance{}).
		Named("rebalance").
		Complete(r)
}

func rebalanceStatusHandler(emqx *crd.EMQX, rebalance *crdv2beta1.Rebalance, req req.RequesterInterface) {
	switch rebalance.Status.Phase {
	case "":
		if err := startRebalance(emqx, rebalance, req); err != nil {
			_ = rebalance.Status.SetFailed(crdv2beta1.RebalanceCondition{
				Type:    crdv2beta1.RebalanceConditionFailed,
				Status:  corev1.ConditionTrue,
				Message: fmt.Sprintf("Failed to start rebalance: %v", err.Error()),
			})
			rebalance.Status.RebalanceStates = nil
		}
		_ = rebalance.Status.SetProcessing(crdv2beta1.RebalanceCondition{
			Type:   crdv2beta1.RebalanceConditionProcessing,
			Status: corev1.ConditionTrue,
		})
	case crdv2beta1.RebalancePhaseProcessing:
		rebalanceStates, err := getRebalanceStatus(req)
		if err != nil {
			_ = rebalance.Status.SetFailed(crdv2beta1.RebalanceCondition{
				Type:    crdv2beta1.RebalanceConditionFailed,
				Status:  corev1.ConditionTrue,
				Message: fmt.Sprintf("Failed to get rebalance status: %s", err.Error()),
			})
		}

		if len(rebalanceStates) == 0 {
			_ = rebalance.Status.SetCompleted(crdv2beta1.RebalanceCondition{
				Type:   crdv2beta1.RebalanceConditionCompleted,
				Status: corev1.ConditionTrue,
			})
			rebalance.Status.RebalanceStates = nil
		}

		_ = rebalance.Status.SetProcessing(crdv2beta1.RebalanceCondition{
			Type:   crdv2beta1.RebalanceConditionProcessing,
			Status: corev1.ConditionTrue,
		})
		rebalance.Status.RebalanceStates = rebalanceStates
	case crdv2beta1.RebalancePhaseFailed, crdv2beta1.RebalancePhaseCompleted:
		rebalance.Status.RebalanceStates = nil
	default:
		panic("unknown rebalance phase")
	}
}

func startRebalance(emqx *crd.EMQX, rebalance *crdv2beta1.Rebalance, req req.RequesterInterface) error {
	nodes := []string{}
	if len(emqx.Status.ReplicantNodes) == 0 {
		for _, node := range emqx.Status.CoreNodes {
			nodes = append(nodes, node.Name)
		}
	} else {
		for _, node := range emqx.Status.ReplicantNodes {
			nodes = append(nodes, node.Name)
		}
	}
	return api.StartRebalance(req, rebalance.Spec.RebalanceStrategy, nodes)
}

func getRebalanceStatus(req req.RequesterInterface) ([]crdv2beta1.RebalanceState, error) {
	return api.GetRebalanceStatus(req)
}
