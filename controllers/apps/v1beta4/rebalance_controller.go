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
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	innerPortFW "github.com/emqx/emqx-operator/internal/portforward"
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
			if rebalance.Status.Phase == appsv1beta4.RebalancePhaseProcessing {
				controllerutil.RemoveFinalizer(rebalance, finalizer)
				return ctrl.Result{}, r.Client.Update(ctx, rebalance)
			}
		}
		_ = rebalance.Status.SetFailed(appsv1beta4.RebalanceCondition{
			Type:    appsv1beta4.RebalanceConditionFailed,
			Status:  corev1.ConditionTrue,
			Message: fmt.Sprintf("EMQX Enterprise %s is not found", rebalance.Spec.InstanceName),
		})
		return ctrl.Result{}, r.Client.Status().Update(ctx, rebalance)
	}

	pod := r.getReadyPod(emqx)
	if pod == nil {
		return ctrl.Result{}, emperror.New("failed to get in-cluster pod")
	}

	portForward, err := r.getPortForwardAPI(emqx, pod)
	if err != nil {
		return ctrl.Result{}, err
	}
	defer close(portForward.GetOptions().StopChannel)
	if err := portForward.GetOptions().ForwardPorts(); err != nil {
		return ctrl.Result{}, err
	}

	if !rebalance.DeletionTimestamp.IsZero() {
		if rebalance.Status.Phase == appsv1beta4.RebalancePhaseProcessing {
			_ = stopRebalance(portForward, rebalance)
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

	if err := rebalanceStatusHandler(rebalance, emqx, pod, portForward,
		startRebalance, getRebalanceStatus); err != nil {
		_ = r.Client.Status().Update(ctx, rebalance)
		return ctrl.Result{}, err
	}
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
		// panic("unknown rebalance phase")
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *RebalanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1beta4.Rebalance{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				return e.ObjectNew.GetGeneration() != e.ObjectOld.GetGeneration()
			},
		}).
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
				return &pod
			}
		}
	}
	return nil
}

func (r *RebalanceReconciler) getPortForwardAPI(instance appsv1beta4.Emqx, pod *corev1.Pod) (PortForwardAPI, error) {
	o, err := innerPortFW.NewPortForwardOptions(r.Clientset, r.Config, pod, "8081")
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create port forward options")
	}

	username, password, err := getBootstrapUser(context.Background(), r.Client, instance)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get bootstrap user")
	}
	return &portForwardAPI{
		Username: username,
		Password: password,
		Options:  o,
	}, nil
}

// Rebalance Handler
type GetRebalanceStatusFunc func(PortForwardAPI) ([]appsv1beta4.RebalanceState, error)
type StartRebalanceFunc func(p PortForwardAPI, rebalance *appsv1beta4.Rebalance, emqx *appsv1beta4.EmqxEnterprise, emqxNodeName string) error
type StopRebalanceFunc func(p PortForwardAPI, rebalance *appsv1beta4.Rebalance) error

func rebalanceStatusHandler(rebalance *appsv1beta4.Rebalance, emqx *appsv1beta4.EmqxEnterprise, pod *corev1.Pod,
	portForward PortForwardAPI, startFun StartRebalanceFunc, getRebalanceStatusFun GetRebalanceStatusFunc,
) error {
	if rebalance.Status.Phase == "" {
		if err := startFun(portForward, rebalance, emqx, getEmqxNodeName(emqx, pod)); err != nil {
			_ = rebalance.Status.SetFailed(appsv1beta4.RebalanceCondition{
				Type:    appsv1beta4.RebalanceConditionFailed,
				Status:  corev1.ConditionTrue,
				Message: fmt.Sprintf("Failed to start rebalance: %v", err.Error()),
			})
			return emperror.Wrap(err, "failed to start rebalance")
		}
		_ = rebalance.Status.SetProcessing(appsv1beta4.RebalanceCondition{
			Type:   appsv1beta4.RebalanceConditionProcessing,
			Status: corev1.ConditionTrue,
		})
		return nil
	}

	rebalanceStates, err := getRebalanceStatusFun(portForward)
	if err != nil {
		_ = rebalance.Status.SetFailed(appsv1beta4.RebalanceCondition{
			Type:    appsv1beta4.RebalanceConditionFailed,
			Status:  corev1.ConditionTrue,
			Message: fmt.Sprintf("Failed to get rebalance status: %s", err.Error()),
		})
		return emperror.Wrap(err, "failed to get rebalance status")
	}
	rebalance.Status.RebalanceStates = rebalanceStates

	if len(rebalanceStates) == 0 {
		if rebalance.Status.Phase == "Processing" {
			_ = rebalance.Status.SetCompleted(appsv1beta4.RebalanceCondition{
				Type:   appsv1beta4.RebalanceConditionCompleted,
				Status: corev1.ConditionTrue,
			})
			return nil
		}
		message := "Can not get rebalance status"
		_ = rebalance.Status.SetFailed(appsv1beta4.RebalanceCondition{
			Type:    appsv1beta4.RebalanceConditionFailed,
			Status:  corev1.ConditionTrue,
			Message: message,
		})
		return nil
	}
	_ = rebalance.Status.SetProcessing(appsv1beta4.RebalanceCondition{
		Type:   appsv1beta4.RebalanceConditionProcessing,
		Status: corev1.ConditionTrue,
	})
	return nil
}

func startRebalance(p PortForwardAPI, rebalance *appsv1beta4.Rebalance, emqx *appsv1beta4.EmqxEnterprise, emqxNodeName string) error {
	nodes := []string{}
	for _, emqxNode := range emqx.Status.EmqxNodes {
		nodes = append(nodes, emqxNode.Node)
	}

	bytes := getRequestBytes(rebalance, nodes)
	resp, respBody, err := p.RequestAPI("POST", "api/v4/load_rebalance/"+emqxNodeName+"/start", bytes)
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

func getRebalanceStatus(p PortForwardAPI) ([]appsv1beta4.RebalanceState, error) {
	resp, body, err := p.RequestAPI("GET", "api/v4/load_rebalance/global_status", nil)
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

func stopRebalance(p PortForwardAPI, rebalance *appsv1beta4.Rebalance) error {
	// stop rebalance should use coordinatorNode as path parameter
	emqxNodeName := rebalance.Status.RebalanceStates[0].CoordinatorNode
	resp, respBody, err := p.RequestAPI("POST", "api/v4/load_rebalance/"+emqxNodeName+"/stop", nil)
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

func getRequestBytes(rebalance *appsv1beta4.Rebalance, nodes []string) []byte {
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
