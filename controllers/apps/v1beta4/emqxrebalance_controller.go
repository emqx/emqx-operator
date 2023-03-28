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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	"github.com/emqx/emqx-operator/internal/handler"
	innerPortFW "github.com/emqx/emqx-operator/internal/portforward"
	"github.com/tidwall/gjson"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EmqxRebalanceReconciler reconciles a EmqxRebalance object
type EmqxRebalanceReconciler struct {
	*handler.Handler
	Clientset *kubernetes.Clientset
	Config    *rest.Config
}

func NewEmqxRebalanceReconciler(mgr manager.Manager) *EmqxRebalanceReconciler {
	return &EmqxRebalanceReconciler{
		Handler:   handler.NewHandler(mgr),
		Clientset: kubernetes.NewForConfigOrDie(mgr.GetConfig()),
		Config:    mgr.GetConfig(),
	}
}

//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxrebalances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxrebalances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxrebalances/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EmqxRebalance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile

func (r *EmqxRebalanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("Reconcile emqx rebalance")

	emqxRebalance := &appsv1beta4.EmqxRebalance{}
	if err := r.Client.Get(ctx, req.NamespacedName, emqxRebalance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if emqxRebalance.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}
	emqxEnterprise := &appsv1beta4.EmqxEnterprise{}
	if err := r.Client.Get(ctx, types.NamespacedName{
		Namespace: emqxRebalance.Namespace, Name: emqxRebalance.Spec.InstanceName,
	}, emqxEnterprise); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	enable, err := r.canExecuteRebalance(emqxRebalance, emqxEnterprise)
	if !enable {
		emqxRebalance.Status.Phase = "Complete"
		r.udpateRebalanceCondition(emqxRebalance, appsv1beta4.RebalanceCondition{
			Type:    appsv1beta4.ConditionComplete,
			Status:  corev1.ConditionFalse,
			Reason:  "Complete",
			Message: err.Error(),
		})
		err = r.Client.Status().Update(ctx, emqxRebalance)
		if err != nil {
			return ctrl.Result{}, emperror.Wrap(err, "failed to update emqx rabalance status")
		}
		return ctrl.Result{}, nil

	}

	if len(emqxRebalance.Status.Rebalances) == 0 {
		err := r.startRebalance(emqxRebalance, emqxEnterprise)
		if err != nil {
			emqxRebalance.Status.Phase = "Complete"
			r.udpateRebalanceCondition(emqxRebalance, appsv1beta4.RebalanceCondition{
				Type:    appsv1beta4.ConditionComplete,
				Status:  corev1.ConditionFalse,
				Reason:  "Complete",
				Message: err.Error(),
			})
			if err := r.Client.Status().Update(ctx, emqxRebalance); err != nil {
				return ctrl.Result{}, emperror.Wrap(err, "failed to update emqx rebalance status")
			}
			return ctrl.Result{}, nil
		}
	}

	rebalances, err := r.getRebalanceStatus(emqxEnterprise)
	if err != nil {
		emqxRebalance.Status.Phase = "Complete"
		r.udpateRebalanceCondition(emqxRebalance, appsv1beta4.RebalanceCondition{
			Type:    appsv1beta4.ConditionComplete,
			Status:  corev1.ConditionFalse,
			Reason:  "Complete",
			Message: err.Error(),
		})
		if err := r.Client.Status().Update(ctx, emqxRebalance); err != nil {
			return ctrl.Result{}, emperror.Wrap(err, "failed to update emqx rebalance status")
		}
		return ctrl.Result{}, nil
	}

	requeueAfter := 0
	if len(rebalances) == 0 {
		emqxRebalance.Status.Phase = "Complete"
		emqxRebalance.Status.CompletionTime = metav1.Now()
		r.udpateRebalanceCondition(emqxRebalance, appsv1beta4.RebalanceCondition{
			Type:    appsv1beta4.ConditionComplete,
			Status:  corev1.ConditionTrue,
			Reason:  "Complete",
			Message: "rebalance has completed",
		})
	} else {
		if emqxRebalance.Status.StartTime.IsZero() {
			emqxRebalance.Status.StartTime = metav1.Now()
		}
		emqxRebalance.Status.Phase = "Process"
		r.udpateRebalanceCondition(emqxRebalance, appsv1beta4.RebalanceCondition{
			Type:    appsv1beta4.ConditionProcess,
			Status:  corev1.ConditionTrue,
			Reason:  "Process",
			Message: "rebalance is processing",
		})
		requeueAfter = 10
	}
	emqxRebalance.Status.Rebalances = rebalances
	if err := r.Client.Status().Update(ctx, emqxRebalance); err != nil {
		return ctrl.Result{}, emperror.Wrap(err, "failed to update emqx status")
	}

	return ctrl.Result{RequeueAfter: time.Duration(requeueAfter) * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EmqxRebalanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1beta4.EmqxRebalance{}).
		Complete(r)
}

func (r *EmqxRebalanceReconciler) canExecuteRebalance(emqxRabalance *appsv1beta4.EmqxRebalance, emqxEnterPrise *appsv1beta4.EmqxEnterprise) (bool, error) {
	conditions := emqxEnterPrise.Status.Conditions
	if len(conditions) == 0 {
		return false, emperror.New("emqx cluster is not ready")
	}
	if conditions[0].Type != appsv1beta4.ConditionRunning || conditions[0].Status != corev1.ConditionTrue {
		return false, emperror.New("emqx cluster is not ready")
	}

	emqxRebalanceList := &appsv1beta4.EmqxRebalanceList{}
	if err := r.Client.List(context.Background(), emqxRebalanceList, client.InNamespace(emqxRabalance.Namespace)); err != nil {
		if k8sErrors.IsNotFound(err) {
			return false, emperror.New("there has no rebalance job")
		}
		return false, emperror.New("failed to list rebalance job")
	}

	for _, item := range emqxRebalanceList.Items {
		if item.Status.Phase == "Process" && item.Name != emqxRabalance.Name {
			return false, emperror.New("there is already a running rebalance job")
		}
	}

	rebalanceConditions := emqxRabalance.Status.Conditions
	if len(rebalanceConditions) > 0 && rebalanceConditions[0].Type == appsv1beta4.ConditionComplete {
		return false, emperror.New(rebalanceConditions[0].Message)
	}

	return true, nil
}

func (r *EmqxRebalanceReconciler) startRebalance(emqxRebalance *appsv1beta4.EmqxRebalance, emqxEnterprise *appsv1beta4.EmqxEnterprise) error {

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
		"conn_evict_rate":    emqxRebalance.Spec.RebalanceStrategy.ConnEvictRate,
		"sess_evict_rate":    emqxRebalance.Spec.RebalanceStrategy.SessEvictRate,
		"wait_takeover":      emqxRebalance.Spec.RebalanceStrategy.WaitTakeover,
		"wait_health_check":  emqxRebalance.Spec.RebalanceStrategy.WaitHealthCheck,
		"abs_conn_threshold": emqxRebalance.Spec.RebalanceStrategy.AbsConnThreshold,
		"abs_sess_threshold": emqxRebalance.Spec.RebalanceStrategy.AbsSessThreshold,
		"nodes":              nodes,
	}

	if len(emqxRebalance.Spec.RebalanceStrategy.RelConnThreshold) > 0 {
		relConnThreshold, _ := strconv.ParseFloat(emqxRebalance.Spec.RebalanceStrategy.RelConnThreshold, 32)
		body["rel_conn_threshold"] = relConnThreshold
	}

	if len(emqxRebalance.Spec.RebalanceStrategy.RelConnThreshold) > 0 {
		relSessThreshold, _ := strconv.ParseFloat(emqxRebalance.Spec.RebalanceStrategy.RelSessThreshold, 32)
		body["rel_sess_threshold"] = relSessThreshold
	}

	bytes, err := json.Marshal(body)
	if err != nil {
		return emperror.Wrap(err, "marshal body failed")
	}

	emqxNodeName := getEmqxNodeName(emqxEnterprise, pod)
	_, respBody, err := p.requestAPI("POST", "api/v4/load_rebalance/"+emqxNodeName+"/start", bytes)
	if err != nil {
		fmt.Println("err:", err.Error())
		return err
	}
	code := gjson.GetBytes(respBody, "code")
	if code.String() == "400" {
		message := gjson.GetBytes(respBody, "message")
		return emperror.New(message.String())
	}
	return nil
}

func (r *EmqxRebalanceReconciler) udpateRebalanceCondition(emqxRebalance *appsv1beta4.EmqxRebalance, condition appsv1beta4.RebalanceCondition) {
	if len(emqxRebalance.Status.Conditions) == 0 {
		condition.LastTransitionTime = metav1.Now()
		emqxRebalance.Status.Conditions = []appsv1beta4.RebalanceCondition{
			condition,
		}
		return
	}
	currCondition := emqxRebalance.Status.Conditions[0]
	if currCondition.Type != condition.Type {
		condition.LastTransitionTime = metav1.Now()
	}
	emqxRebalance.Status.Conditions[0] = condition
}

func (r *EmqxRebalanceReconciler) getRebalanceStatus(emqxEnterprise *appsv1beta4.EmqxEnterprise) ([]appsv1beta4.Rebalance, error) {
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

	rebalances := []appsv1beta4.Rebalance{}
	data := gjson.GetBytes(body, "rebalances")
	if err := json.Unmarshal([]byte(data.Raw), &rebalances); err != nil {
		return nil, emperror.Wrap(err, "failed to unmarshal rebalances")
	}
	return rebalances, nil
}

func (r *EmqxRebalanceReconciler) getPodInCluster(emqxEnterprise *appsv1beta4.EmqxEnterprise) *corev1.Pod {
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

func (r *EmqxRebalanceReconciler) getPortForwardAPI(instance appsv1beta4.Emqx, pod *corev1.Pod) (*portForwardAPI, error) {
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
