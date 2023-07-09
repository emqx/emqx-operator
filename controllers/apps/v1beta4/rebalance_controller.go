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

package v1beta4

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
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	"github.com/tidwall/gjson"
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
// the EmqxRebalance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile

func (r *RebalanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	finalizer := "apps.emqx.io/finalizer"
	logger := log.FromContext(ctx)
	logger.V(1).Info("Reconcile rebalance")

	rebalance := &appsv1beta4.Rebalance{}
	if err := r.Client.Get(ctx, req.NamespacedName, rebalance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	emqx := &appsv1beta4.EmqxEnterprise{}
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
		_ = rebalance.Status.SetFailed(appsv1beta4.RebalanceCondition{
			Type:    appsv1beta4.RebalanceConditionFailed,
			Status:  corev1.ConditionTrue,
			Message: fmt.Sprintf("EMQX Enterprise %s is not found", rebalance.Spec.InstanceName),
		})
		return ctrl.Result{}, r.Client.Status().Update(ctx, rebalance)
	}

	readyPod := r.getReadyPod(emqx)
	if readyPod == nil {
		return ctrl.Result{}, emperror.New("failed to get ready pod")
	}

	requester, err := newRequesterByPod(r.Client, emqx, readyPod)
	if err != nil {
		return ctrl.Result{}, emperror.New("failed to get create emqx http API")
	}

	if !rebalance.DeletionTimestamp.IsZero() {
		if rebalance.Status.Phase == appsv1beta4.RebalancePhaseProcessing {
			_ = stopRebalance(requester, rebalance)
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

	rebalanceStatusHandler(rebalance, emqx, requester, startRebalance, getRebalanceStatus)
	if err := r.Client.Status().Update(ctx, rebalance); err != nil {
		return ctrl.Result{}, err
	}

	switch rebalance.Status.Phase {
	case "Failed":
		r.EventRecorder.Event(rebalance, corev1.EventTypeWarning, "Rebalance", "rebalance failed")
		return ctrl.Result{}, nil
	case "Completed":
		r.EventRecorder.Event(rebalance, corev1.EventTypeNormal, "Rebalance", "rebalance completed")
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
		For(&appsv1beta4.Rebalance{}).
		Complete(r)
}

func (r *RebalanceReconciler) getReadyPod(emqxEnterprise *appsv1beta4.EmqxEnterprise) *corev1.Pod {
	podList := &corev1.PodList{}
	_ = r.Client.List(context.Background(), podList,
		client.InNamespace(emqxEnterprise.GetNamespace()),
		client.MatchingLabels(emqxEnterprise.GetSpec().GetTemplate().Labels),
	)
	for _, pod := range podList.Items {
		for _, c := range pod.Status.Conditions {
			if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
				return pod.DeepCopy()
			}
		}
	}
	return nil
}

// Rebalance Handler
type GetRebalanceStatusFunc func(requester innerReq.RequesterInterface) ([]appsv1beta4.RebalanceState, error)
type StartRebalanceFunc func(requester innerReq.RequesterInterface, rebalance *appsv1beta4.Rebalance, emqx *appsv1beta4.EmqxEnterprise) error
type StopRebalanceFunc func(requester innerReq.RequesterInterface, rebalance *appsv1beta4.Rebalance) error

func rebalanceStatusHandler(rebalance *appsv1beta4.Rebalance, emqx *appsv1beta4.EmqxEnterprise,
	requester innerReq.RequesterInterface, startFun StartRebalanceFunc, getRebalanceStatusFun GetRebalanceStatusFunc,
) {
	switch rebalance.Status.Phase {
	case "":
		if err := startFun(requester, rebalance, emqx); err != nil {
			_ = rebalance.Status.SetFailed(appsv1beta4.RebalanceCondition{
				Type:    appsv1beta4.RebalanceConditionFailed,
				Status:  corev1.ConditionTrue,
				Message: fmt.Sprintf("Failed to start rebalance: %v", err.Error()),
			})
			rebalance.Status.RebalanceStates = nil
		}
		_ = rebalance.Status.SetProcessing(appsv1beta4.RebalanceCondition{
			Type:   appsv1beta4.RebalanceConditionProcessing,
			Status: corev1.ConditionTrue,
		})
	case appsv1beta4.RebalancePhaseProcessing:
		rebalanceStates, err := getRebalanceStatusFun(requester)
		if err != nil {
			_ = rebalance.Status.SetFailed(appsv1beta4.RebalanceCondition{
				Type:    appsv1beta4.RebalanceConditionFailed,
				Status:  corev1.ConditionTrue,
				Message: fmt.Sprintf("Failed to get rebalance status: %s", err.Error()),
			})
		}

		if len(rebalanceStates) == 0 {
			_ = rebalance.Status.SetCompleted(appsv1beta4.RebalanceCondition{
				Type:   appsv1beta4.RebalanceConditionCompleted,
				Status: corev1.ConditionTrue,
			})
			rebalance.Status.RebalanceStates = nil
		}

		_ = rebalance.Status.SetProcessing(appsv1beta4.RebalanceCondition{
			Type:   appsv1beta4.RebalanceConditionProcessing,
			Status: corev1.ConditionTrue,
		})
		rebalance.Status.RebalanceStates = rebalanceStates
	case appsv1beta4.RebalancePhaseFailed, appsv1beta4.RebalancePhaseCompleted:
		rebalance.Status.RebalanceStates = nil
	default:
		panic("unknown rebalance phase")
	}
}

func startRebalance(requester innerReq.RequesterInterface, rebalance *appsv1beta4.Rebalance, emqx *appsv1beta4.EmqxEnterprise) error {
	emqxNodeName := emqx.Status.EmqxNodes[0].Node

	bytes := getRequestBytes(rebalance, emqx)
	resp, respBody, err := requester.Request("POST", requester.GetURL("api/v4/load_rebalance/"+emqxNodeName+"/start"), bytes, nil)
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

func getRebalanceStatus(requester innerReq.RequesterInterface) ([]appsv1beta4.RebalanceState, error) {
	resp, body, err := requester.Request("GET", requester.GetURL("api/v4/load_rebalance/global_status"), nil, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, emperror.Errorf("request api failed: %s", resp.Status)
	}
	rebalanceStates := []appsv1beta4.RebalanceState{}
	data := gjson.GetBytes(body, "rebalances")
	if err := json.Unmarshal([]byte(data.Raw), &rebalanceStates); err != nil {
		return nil, emperror.Wrap(err, "failed to unmarshal rebalances")
	}
	return rebalanceStates, nil
}

func stopRebalance(requester innerReq.RequesterInterface, rebalance *appsv1beta4.Rebalance) error {
	// stop rebalance should use coordinatorNode as path parameter
	emqxNodeName := rebalance.Status.RebalanceStates[0].CoordinatorNode
	resp, respBody, err := requester.Request("POST", requester.GetURL("api/v4/load_rebalance/"+emqxNodeName+"/stop"), nil, nil)
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

func getRequestBytes(rebalance *appsv1beta4.Rebalance, emqx *appsv1beta4.EmqxEnterprise) []byte {
	nodes := []string{}
	for _, emqxNode := range emqx.Status.EmqxNodes {
		nodes = append(nodes, emqxNode.Node)
	}

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
