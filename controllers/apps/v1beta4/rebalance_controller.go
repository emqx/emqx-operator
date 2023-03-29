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
	"strconv"
	"time"

	emperror "emperror.dev/errors"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	"github.com/emqx/emqx-operator/internal/handler"
	innerPortFW "github.com/emqx/emqx-operator/internal/portforward"
	"github.com/tidwall/gjson"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RebalanceReconciler reconciles a Rebalance object
type RebalanceReconciler struct {
	*handler.Handler
	Clientset     *kubernetes.Clientset
	Config        *rest.Config
	EventRecorder record.EventRecorder
}

func NewRebalanceReconciler(mgr manager.Manager) *RebalanceReconciler {
	return &RebalanceReconciler{
		Handler:       handler.NewHandler(mgr),
		Clientset:     kubernetes.NewForConfigOrDie(mgr.GetConfig()),
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
	logger := log.FromContext(ctx)
	logger.V(1).Info("Reconcile rebalance")

	rebalance := &appsv1beta4.Rebalance{}
	if err := r.Client.Get(ctx, req.NamespacedName, rebalance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	finalizer := "apps.emqx.io/finalizer"
	if !controllerutil.ContainsFinalizer(rebalance, finalizer) {
		controllerutil.AddFinalizer(rebalance, finalizer)
		err := r.Client.Update(ctx, rebalance)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	emqxEnterprise := &appsv1beta4.EmqxEnterprise{}
	if err := r.Client.Get(ctx, types.NamespacedName{
		Namespace: rebalance.Namespace, Name: rebalance.Spec.InstanceName,
	}, emqxEnterprise); err != nil {
		rebalance.Status.Phase = "Failed"
		rebalance.Status.RebalanceStates = []appsv1beta4.RebalanceState{}
		r.udpateRebalanceCondition(rebalance, appsv1beta4.RebalanceCondition{
			Type:           appsv1beta4.RebalanceFailed,
			Status:         corev1.ConditionFalse,
			Reason:         "Failed",
			Message:        err.Error(),
			LastUpdateTime: metav1.Now(),
		})
		err = r.Client.Status().Update(ctx, rebalance)
		return ctrl.Result{}, err
	}

	if rebalance.DeletionTimestamp != nil {
		err := r.stopRebalance(rebalance, emqxEnterprise, finalizer)
		if err != nil {
			return ctrl.Result{}, err
		}
		controllerutil.RemoveFinalizer(rebalance, finalizer)
		err = r.Client.Update(ctx, rebalance)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	enable, err := r.canExecuteRebalance(rebalance)
	if !enable {
		if rebalance.Status.Phase != "Complete" {
			rebalance.Status.Phase = "Failed"
			rebalance.Status.RebalanceStates = []appsv1beta4.RebalanceState{}
			r.udpateRebalanceCondition(rebalance, appsv1beta4.RebalanceCondition{
				Type:           appsv1beta4.RebalanceFailed,
				Status:         corev1.ConditionFalse,
				Reason:         "Failed",
				Message:        err.Error(),
				LastUpdateTime: metav1.Now(),
			})
		} else {
			rebalance.Status.Conditions[0].LastUpdateTime = metav1.Now()
		}
		err = r.Client.Status().Update(ctx, rebalance)
		if err != nil {
			return ctrl.Result{}, emperror.Wrap(err, "failed to update emqx rabalance status")
		}
		return ctrl.Result{}, nil
	}

	if len(rebalance.Status.RebalanceStates) == 0 {
		err := r.startRebalance(ctx, rebalance, emqxEnterprise)
		if err != nil {
			rebalance.Status.Phase = "Failed"
			rebalance.Status.RebalanceStates = []appsv1beta4.RebalanceState{}
			r.udpateRebalanceCondition(rebalance, appsv1beta4.RebalanceCondition{
				Type:           appsv1beta4.RebalanceFailed,
				Status:         corev1.ConditionFalse,
				Reason:         "Failed",
				Message:        err.Error(),
				LastUpdateTime: metav1.Now(),
			})
			if err := r.Client.Status().Update(ctx, rebalance); err != nil {
				return ctrl.Result{}, emperror.Wrap(err, "failed to update emqx rebalance status")
			}
			return ctrl.Result{}, nil
		}
	}

	rebalanceStates, err := r.getRebalanceStatus(emqxEnterprise)
	if err != nil {
		rebalance.Status.Phase = "Failed"
		r.udpateRebalanceCondition(rebalance, appsv1beta4.RebalanceCondition{
			Type:           appsv1beta4.RebalanceFailed,
			Status:         corev1.ConditionFalse,
			Reason:         "Failed",
			Message:        err.Error(),
			LastUpdateTime: metav1.Now(),
		})
		if err := r.Client.Status().Update(ctx, rebalance); err != nil {
			return ctrl.Result{}, emperror.Wrap(err, "failed to update emqx rebalance status")
		}
		return ctrl.Result{}, nil
	}

	requeueAfter := 0
	if len(rebalanceStates) == 0 {
		rebalance.Status.Phase = "Complete"
		rebalance.Status.CompletionTime = metav1.Now()
		r.udpateRebalanceCondition(rebalance, appsv1beta4.RebalanceCondition{
			Type:           appsv1beta4.RebalanceComplete,
			Status:         corev1.ConditionTrue,
			Reason:         "Complete",
			Message:        "rebalance has completed",
			LastUpdateTime: metav1.Now(),
		})
		r.EventRecorder.Event(rebalance, corev1.EventTypeNormal, "Rebalance", "complete rebalance successfully")
	} else {
		if rebalance.Status.StartTime.IsZero() {
			rebalance.Status.StartTime = metav1.Now()
		}
		rebalance.Status.Phase = "Processing"
		r.udpateRebalanceCondition(rebalance, appsv1beta4.RebalanceCondition{
			Type:           appsv1beta4.RebalanceProcessing,
			Status:         corev1.ConditionTrue,
			Reason:         "Processing",
			Message:        "rebalance is processing",
			LastUpdateTime: metav1.Now(),
		})
		requeueAfter = 10
	}
	rebalance.Status.RebalanceStates = rebalanceStates
	if err := r.Client.Status().Update(ctx, rebalance); err != nil {
		return ctrl.Result{}, emperror.Wrap(err, "failed to update emqx status")
	}
	return ctrl.Result{RequeueAfter: time.Duration(requeueAfter) * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RebalanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1beta4.Rebalance{}).
		Complete(r)
}

func (r *RebalanceReconciler) canExecuteRebalance(rebalance *appsv1beta4.Rebalance) (bool, error) {
	rebalanceList := &appsv1beta4.RebalanceList{}
	if err := r.Client.List(context.Background(), rebalanceList, client.InNamespace(rebalance.Namespace)); err != nil {
		return false, emperror.Wrap(err, "failed to list rebalance job")
	}
	for _, item := range rebalanceList.Items {
		if rebalance.Status.Phase == "Processing" && rebalance.Name != item.Name {
			return false, emperror.New("there is already a running rebalance job")
		}
	}
	conditions := rebalance.Status.Conditions
	if len(conditions) > 0 {
		if conditions[0].Type == appsv1beta4.RebalanceComplete || conditions[0].Type == appsv1beta4.RebalanceFailed {
			return false, emperror.New(conditions[0].Message)
		}
	}
	return true, nil
}

func (r *RebalanceReconciler) startRebalance(ctx context.Context, rebalance *appsv1beta4.Rebalance, emqxEnterprise *appsv1beta4.EmqxEnterprise) error {
	pod := r.getPodInCluster(emqxEnterprise)
	if pod == nil {
		return emperror.New("failed to get in-cluster pod")
	}
	p, err := r.getPortForwardAPI(emqxEnterprise, pod)
	if err != nil {
		return err
	}
	defer close(p.Options.StopChannel)
	if err := p.Options.ForwardPorts(); err != nil {
		return emperror.Wrap(err, "failed to forward ports")
	}

	nodes := []string{}
	for _, emqxNode := range emqxEnterprise.Status.EmqxNodes {
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
		relConnThreshold, _ := strconv.ParseFloat(rebalance.Spec.RebalanceStrategy.RelConnThreshold, 32)
		body["rel_conn_threshold"] = relConnThreshold
	}

	if len(rebalance.Spec.RebalanceStrategy.RelConnThreshold) > 0 {
		relSessThreshold, _ := strconv.ParseFloat(rebalance.Spec.RebalanceStrategy.RelSessThreshold, 32)
		body["rel_sess_threshold"] = relSessThreshold
	}

	bytes, err := json.Marshal(body)
	if err != nil {
		return emperror.Wrap(err, "marshal body failed")
	}

	emqxNodeName := getEmqxNodeName(emqxEnterprise, pod)
	_, respBody, err := p.requestAPI("POST", "api/v4/load_rebalance/"+emqxNodeName+"/start", bytes)
	if err != nil {
		return err
	}
	code := gjson.GetBytes(respBody, "code")
	if code.String() == "400" {
		message := gjson.GetBytes(respBody, "message")
		return emperror.New(message.String())
	}
	r.EventRecorder.Event(rebalance, corev1.EventTypeNormal, "Rebalance", "start rebalance successfully")
	return nil
}

func (r *RebalanceReconciler) udpateRebalanceCondition(rebalance *appsv1beta4.Rebalance, condition appsv1beta4.RebalanceCondition) {
	if len(rebalance.Status.Conditions) == 0 {
		condition.LastTransitionTime = metav1.Now()
		rebalance.Status.Conditions = []appsv1beta4.RebalanceCondition{
			condition,
		}
		return
	}
	currCondition := rebalance.Status.Conditions[0]
	if currCondition.Type != condition.Type {
		condition.LastTransitionTime = metav1.Now()
	} else {
		condition.LastTransitionTime = currCondition.LastTransitionTime
	}
	rebalance.Status.Conditions[0] = condition
}

func (r *RebalanceReconciler) getRebalanceStatus(emqxEnterprise *appsv1beta4.EmqxEnterprise) ([]appsv1beta4.RebalanceState, error) {
	p, _ := newPortForwardAPI(context.Background(), r.Client, r.Clientset, r.Config, emqxEnterprise)
	if p == nil {
		return nil, emperror.New("fail to get portforward")
	}
	defer close(p.Options.StopChannel)
	if err := p.Options.ForwardPorts(); err != nil {
		return nil, emperror.Wrap(err, "failed to forward ports")
	}

	_, body, err := p.requestAPI("GET", "api/v4/load_rebalance/global_status", nil)
	if err != nil {
		return nil, err
	}

	rebalanceStates := []appsv1beta4.RebalanceState{}
	data := gjson.GetBytes(body, "rebalances")
	if err := json.Unmarshal([]byte(data.Raw), &rebalanceStates); err != nil {
		return nil, emperror.Wrap(err, "failed to unmarshal rebalances")
	}
	return rebalanceStates, nil
}

func (r *RebalanceReconciler) getPodInCluster(emqxEnterprise *appsv1beta4.EmqxEnterprise) *corev1.Pod {
	podList := &corev1.PodList{}
	_ = r.Client.List(context.Background(), podList,
		client.InNamespace(emqxEnterprise.GetNamespace()),
		client.MatchingLabels(emqxEnterprise.GetSpec().GetTemplate().Labels),
	)
	clusterPods := make(map[string]struct{})
	for _, emqx := range emqxEnterprise.GetStatus().GetEmqxNodes() {
		podName := extractPodName(emqx.Node)
		clusterPods[podName] = struct{}{}
	}
	for _, pod := range podList.Items {
		if _, ok := clusterPods[pod.Name]; !ok {
			continue
		}
		for _, c := range pod.Status.Conditions {
			if c.Type == corev1.ContainersReady && c.Status == corev1.ConditionTrue {
				return &pod
			}
		}
	}
	return nil
}

func (r *RebalanceReconciler) getPortForwardAPI(instance appsv1beta4.Emqx, pod *corev1.Pod) (*portForwardAPI, error) {
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

func (r *RebalanceReconciler) stopRebalance(rebalance *appsv1beta4.Rebalance, emqxEnterprise *appsv1beta4.EmqxEnterprise, finalizer string) error {
	if len(rebalance.Status.RebalanceStates) == 0 {
		return nil
	}

	pod := r.getPodInCluster(emqxEnterprise)
	if pod == nil {
		return emperror.New("failed to get in-cluster pod")
	}

	p, _ := newPortForwardAPI(context.Background(), r.Client, r.Clientset, r.Config, emqxEnterprise)
	if p == nil {
		return emperror.New("fail to get portforward")
	}
	defer close(p.Options.StopChannel)
	if err := p.Options.ForwardPorts(); err != nil {
		return emperror.Wrap(err, "failed to forward ports")
	}

	emqxNodeName := getEmqxNodeName(emqxEnterprise, pod)
	_, respBody, err := p.requestAPI("POST", "api/v4/load_rebalance/"+emqxNodeName+"/stop", nil)
	if err != nil {
		return err
	}
	code := gjson.GetBytes(respBody, "code")
	if code.String() == "400" {
		message := gjson.GetBytes(respBody, "message")
		return emperror.New(message.String())
	}
	return nil
}
