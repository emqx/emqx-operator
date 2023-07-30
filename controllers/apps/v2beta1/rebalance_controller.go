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
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	emperror "emperror.dev/errors"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	controllerv1beta4 "github.com/emqx/emqx-operator/controllers/apps/v1beta4"

	// controllerv2beta1 "github.com/emqx/emqx-operator/controllers/apps/v2beta1"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	"github.com/tidwall/gjson"
)

const (
	ApiRebalanceV4 = "api/v4/load_rebalance"
	ApiRebalanceV5 = "api/v5/load_rebalance"
)

// RebalanceReconciler reconciles a Rebalance object
type RebalanceReconciler struct {
	Client        client.Client
	Clientset     *kubernetes.Clientset
	Config        *rest.Config
	EventRecorder record.EventRecorder
}

func NewRebalanceReconciler(mgr manager.Manager) *RebalanceReconciler {
	return &RebalanceReconciler{
		Clientset:     kubernetes.NewForConfigOrDie(mgr.GetConfig()),
		Client:        mgr.GetClient(),
		Config:        mgr.GetConfig(),
		EventRecorder: mgr.GetEventRecorderFor("rebalance-controller"),
	}
}

//+kubebuilder:rbac:groups=apps.emqx.io,resources=rebalances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.emqx.io,resources=rebalances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.emqx.io,resources=rebalances/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Rebalance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile

func (r *RebalanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var err error
	var finalizer string = "apps.emqx.io/finalizer"
	var requester innerReq.RequesterInterface
	var targetEMQX client.Object

	logger := log.FromContext(ctx)
	logger.V(1).Info("Reconcile rebalance")

	rebalance := &appsv2beta1.Rebalance{}
	if err := r.Client.Get(ctx, req.NamespacedName, rebalance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// check instanceKind is v1beta4 or v2beta1
	if rebalance.Spec.InstanceKind == "EmqxEnterprise" {
		emqx := &appsv1beta4.EmqxEnterprise{}
		if err := r.Client.Get(ctx, client.ObjectKey{
			Name:      rebalance.Spec.InstanceName,
			Namespace: rebalance.Namespace,
		}, emqx); err != nil {
			if !k8sErrors.IsNotFound(err) {
				return ctrl.Result{}, emperror.Wrap(err, "failed to get EMQX Enterprise")
			}
			if !rebalance.DeletionTimestamp.IsZero() {
				controllerutil.RemoveFinalizer(rebalance, finalizer)
				return ctrl.Result{}, r.Client.Update(ctx, rebalance)
			}
			_ = rebalance.Status.SetFailed(appsv2beta1.RebalanceCondition{
				Type:    appsv2beta1.RebalanceConditionFailed,
				Status:  corev1.ConditionTrue,
				Message: fmt.Sprintf("EMQX Enterprise %s is not found", rebalance.Spec.InstanceName),
			})
			return ctrl.Result{}, r.Client.Status().Update(ctx, rebalance)
		}

		if !emqx.Status.IsConditionTrue(appsv1beta4.ConditionRunning) {
			_ = rebalance.Status.SetFailed(appsv2beta1.RebalanceCondition{
				Type:    appsv2beta1.RebalanceConditionFailed,
				Status:  corev1.ConditionTrue,
				Message: fmt.Sprintf("EMQX Enterprise %s is not ready", rebalance.Spec.InstanceName),
			})
			return ctrl.Result{}, r.Client.Status().Update(ctx, rebalance)
		}

		requester, err = controllerv1beta4.NewRequesterByPod(r.Client, emqx)
		if err != nil {
			return ctrl.Result{}, emperror.New("failed to get create emqx http API")
		}
		targetEMQX = emqx
	} else {
		emqx := &appsv2beta1.EMQX{}
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
			_ = rebalance.Status.SetFailed(appsv2beta1.RebalanceCondition{
				Type:    appsv2beta1.RebalanceConditionFailed,
				Status:  corev1.ConditionTrue,
				Message: fmt.Sprintf("EMQX %s is not found", rebalance.Spec.InstanceName),
			})
			return ctrl.Result{}, r.Client.Status().Update(ctx, rebalance)
		}

		// check if emqx is ready
		if !emqx.Status.IsConditionTrue(appsv2beta1.Ready) {
			// return ctrl.Result{}, emperror.New("EMQX is not ready")
			_ = rebalance.Status.SetFailed(appsv2beta1.RebalanceCondition{
				Type:    appsv2beta1.RebalanceConditionFailed,
				Status:  corev1.ConditionTrue,
				Message: fmt.Sprintf("EMQX %s is not ready", rebalance.Spec.InstanceName),
			})
			return ctrl.Result{}, r.Client.Status().Update(ctx, rebalance)
		}

		// check if emqx is enterprise edition
		if emqx.Status.CoreNodes[0].Edition != "Enterprise" {
			_ = rebalance.Status.SetFailed(appsv2beta1.RebalanceCondition{
				Type:    appsv2beta1.RebalanceConditionFailed,
				Status:  corev1.ConditionTrue,
				Message: "Only enterprise edition can be rebalanced",
			})
			return ctrl.Result{}, r.Client.Status().Update(ctx, rebalance)
		}

		requester, err = newRequester(r.Client, emqx)
		if err != nil {
			return ctrl.Result{}, emperror.New("failed to get create emqx http API")
		}
		targetEMQX = emqx
	}

	if !rebalance.DeletionTimestamp.IsZero() {
		if rebalance.Status.Phase == appsv2beta1.RebalancePhaseProcessing {
			_ = stopRebalance(targetEMQX, requester, rebalance)
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

	rebalanceStatusHandler(targetEMQX, rebalance, requester, startRebalance, getRebalanceStatus)
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
		For(&appsv2beta1.Rebalance{}).
		Complete(r)
}

// Rebalance Handler
type GetRebalanceStatusFunc func(emqx client.Object, requester innerReq.RequesterInterface) ([]appsv2beta1.RebalanceState, error)
type StartRebalanceFunc func(emqx client.Object, requester innerReq.RequesterInterface, rebalance *appsv2beta1.Rebalance) error
type StopRebalanceFunc func(emqx client.Object, requester innerReq.RequesterInterface, rebalance *appsv2beta1.Rebalance) error

func rebalanceStatusHandler(emqx client.Object, rebalance *appsv2beta1.Rebalance, requester innerReq.RequesterInterface,
	startFun StartRebalanceFunc, getRebalanceStatusFun GetRebalanceStatusFunc,
) {
	switch rebalance.Status.Phase {
	case "":
		if err := startFun(emqx, requester, rebalance); err != nil {
			_ = rebalance.Status.SetFailed(appsv2beta1.RebalanceCondition{
				Type:    appsv2beta1.RebalanceConditionFailed,
				Status:  corev1.ConditionTrue,
				Message: fmt.Sprintf("Failed to start rebalance: %v", err.Error()),
			})
			rebalance.Status.RebalanceStates = nil
		}
		_ = rebalance.Status.SetProcessing(appsv2beta1.RebalanceCondition{
			Type:   appsv2beta1.RebalanceConditionProcessing,
			Status: corev1.ConditionTrue,
		})
	case appsv2beta1.RebalancePhaseProcessing:
		rebalanceStates, err := getRebalanceStatusFun(emqx, requester)
		if err != nil {
			_ = rebalance.Status.SetFailed(appsv2beta1.RebalanceCondition{
				Type:    appsv2beta1.RebalanceConditionFailed,
				Status:  corev1.ConditionTrue,
				Message: fmt.Sprintf("Failed to get rebalance status: %s", err.Error()),
			})
		}

		if len(rebalanceStates) == 0 {
			_ = rebalance.Status.SetCompleted(appsv2beta1.RebalanceCondition{
				Type:   appsv2beta1.RebalanceConditionCompleted,
				Status: corev1.ConditionTrue,
			})
			rebalance.Status.RebalanceStates = nil
		}

		_ = rebalance.Status.SetProcessing(appsv2beta1.RebalanceCondition{
			Type:   appsv2beta1.RebalanceConditionProcessing,
			Status: corev1.ConditionTrue,
		})
		rebalance.Status.RebalanceStates = rebalanceStates
	case appsv2beta1.RebalancePhaseFailed, appsv2beta1.RebalancePhaseCompleted:
		rebalance.Status.RebalanceStates = nil
	default:
		panic("unknown rebalance phase")
	}
}

func startRebalance(emqx client.Object, requester innerReq.RequesterInterface, rebalance *appsv2beta1.Rebalance) error {
	nodes, err := getEmqxNodes(emqx)
	if err != nil {
		return err
	}

	path, err := rebalanceStartUrl(emqx, nodes[0])
	if err != nil {
		return err
	}

	body := getRequestBytes(rebalance, nodes)
	resp, respBody, err := requester.Request("POST", requester.GetURL(path), body, nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return emperror.Errorf("request api failed: %s", resp.Status)
	}

	code := gjson.GetBytes(respBody, "code")
	if code.String() == "400" {
		message := gjson.GetBytes(respBody, "message")
		return emperror.New(message.String())
	}
	return nil
}

func getRebalanceStatus(emqx client.Object, requester innerReq.RequesterInterface) ([]appsv2beta1.RebalanceState, error) {
	path, err := rebalanceStatusUrl(emqx)
	if err != nil {
		return nil, err
	}

	resp, body, err := requester.Request("GET", requester.GetURL(path), nil, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, emperror.Errorf("request api failed: %s", resp.Status)
	}
	rebalanceStates := []appsv2beta1.RebalanceState{}
	data := gjson.GetBytes(body, "rebalances")
	if err := json.Unmarshal([]byte(data.Raw), &rebalanceStates); err != nil {
		return nil, emperror.Wrap(err, "failed to unmarshal rebalances")
	}
	return rebalanceStates, nil
}

func stopRebalance(emqx client.Object, requester innerReq.RequesterInterface, rebalance *appsv2beta1.Rebalance) error {
	// stop rebalance should use coordinatorNode as path parameter
	path, err := rebalanceStopUrl(emqx, rebalance.Status.RebalanceStates[0].CoordinatorNode)
	if err != nil {
		return err
	}

	resp, respBody, err := requester.Request("POST", requester.GetURL(path), nil, nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return emperror.Errorf("request api failed: %s", resp.Status)
	}
	code := gjson.GetBytes(respBody, "code")
	if code.String() == "400" {
		message := gjson.GetBytes(respBody, "message")
		return emperror.New(message.String())
	}
	return nil
}

func getRequestBytes(rebalance *appsv2beta1.Rebalance, nodes []string) []byte {
	body := map[string]interface{}{
		"conn_evict_rate":    rebalance.Spec.RebalanceStrategy.ConnEvictRate,
		"sess_evict_rate":    rebalance.Spec.RebalanceStrategy.SessEvictRate,
		"wait_takeover":      rebalance.Spec.RebalanceStrategy.WaitTakeover,
		"wait_health_check":  rebalance.Spec.RebalanceStrategy.WaitHealthCheck,
		"abs_conn_threshold": rebalance.Spec.RebalanceStrategy.AbsConnThreshold,
		"abs_sess_threshold": rebalance.Spec.RebalanceStrategy.AbsSessThreshold,
		"nodes":              nodes,
	}

	if len(rebalance.Spec.RebalanceStrategy.RelConnThreshold) > 0 {
		relConnThreshold, _ := strconv.ParseFloat(rebalance.Spec.RebalanceStrategy.RelConnThreshold, 64)
		body["rel_conn_threshold"] = relConnThreshold
	}

	if len(rebalance.Spec.RebalanceStrategy.RelSessThreshold) > 0 {
		relSessThreshold, _ := strconv.ParseFloat(rebalance.Spec.RebalanceStrategy.RelSessThreshold, 64)
		body["rel_sess_threshold"] = relSessThreshold
	}

	bytes, _ := json.Marshal(body)
	return bytes
}

// helper functions
func getEmqxNodes(emqx client.Object) ([]string, error) {
	nodes := []string{}
	if e, ok := emqx.(*appsv1beta4.EmqxEnterprise); ok {
		for _, node := range e.Status.EmqxNodes {
			nodes = append(nodes, node.Node)
		}
	} else if e, ok := emqx.(*appsv2beta1.EMQX); ok {
		if len(e.Status.ReplicantNodes) == 0 {
			for _, node := range e.Status.CoreNodes {
				nodes = append(nodes, node.Node)
			}
		} else {
			for _, node := range e.Status.ReplicantNodes {
				nodes = append(nodes, node.Node)
			}
		}
	} else {
		return nil, emperror.New("emqx type error")
	}
	return nodes, nil
}

func rebalanceStartUrl(emqx client.Object, node string) (string, error) {
	if _, ok := emqx.(*appsv1beta4.EmqxEnterprise); ok {
		return fmt.Sprintf("%s/%s/start", ApiRebalanceV4, node), nil
	} else if _, ok := emqx.(*appsv2beta1.EMQX); ok {
		return fmt.Sprintf("%s/%s/start", ApiRebalanceV5, node), nil
	} else {
		return "", emperror.New("emqx type error")
	}
}

func rebalanceStopUrl(emqx client.Object, node string) (string, error) {
	if _, ok := emqx.(*appsv1beta4.EmqxEnterprise); ok {
		return fmt.Sprintf("%s/%s/stop", ApiRebalanceV4, node), nil
	} else if _, ok := emqx.(*appsv2beta1.EMQX); ok {
		return fmt.Sprintf("%s/%s/stop", ApiRebalanceV5, node), nil
	} else {
		return "", emperror.New("emqx type error")
	}
}

func rebalanceStatusUrl(emqx client.Object) (string, error) {
	if _, ok := emqx.(*appsv1beta4.EmqxEnterprise); ok {
		return fmt.Sprintf("%s/global_status", ApiRebalanceV4), nil
	} else if _, ok := emqx.(*appsv2beta1.EMQX); ok {
		return fmt.Sprintf("%s/global_status", ApiRebalanceV5), nil
	} else {
		return "", emperror.New("emqx type error")
	}
}
