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

package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	emperror "emperror.dev/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	"github.com/emqx/emqx-operator/pkg/handler"
	appsv1 "k8s.io/api/apps/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
)

const EMQXContainerName string = "emqx"

var (
	username      string = "admin"
	password      string = "public"
	dashboardPort string = "18083"
)

// EMQXReconciler reconciles a EMQX object
type EMQXReconciler struct {
	handler.Handler
	Scheme *runtime.Scheme
	record.EventRecorder
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
	_ = log.FromContext(ctx)

	instance := &appsv2alpha1.EMQX{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Update EMQX Custom Resource's status
	instance, err := r.updateStatus(instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	// Create Resources
	resources, err := r.createResources(instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := r.CreateOrUpdateList(instance, r.Scheme, resources, func(client.Object) error { return nil }); err != nil {
		return ctrl.Result{}, err
	}

	if !instance.Status.IsRunning() {
		return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
	}
	return ctrl.Result{RequeueAfter: time.Duration(20) * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EMQXReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv2alpha1.EMQX{}).
		Complete(r)
}

func (r *EMQXReconciler) createResources(instance *appsv2alpha1.EMQX) ([]client.Object, error) {
	dashboardSvc := generateDashboardService(instance)
	headlessSvc := generateHeadlessService(instance)
	sts := generateStatefulSet(instance)

	if !reflect.ValueOf(instance.Spec.CoreTemplate.Spec.Persistent).IsZero() {
		pvcList := &corev1.PersistentVolumeClaimList{}
		_ = r.List(context.TODO(), pvcList, client.InNamespace(instance.GetNamespace()), client.MatchingLabels(instance.GetLabels()))
		if len(pvcList.Items) != 0 {
			sts.Spec.PodManagementPolicy = appsv1.ParallelPodManagement
		}
	}
	storeSts := &appsv1.StatefulSet{}
	if err := r.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s-core", instance.Name), Namespace: instance.Namespace}, storeSts); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return nil, err
		}
	}
	if storeSts.Spec.PodManagementPolicy != "" {
		sts.Spec.PodManagementPolicy = storeSts.Spec.PodManagementPolicy
	}

	resources := []client.Object{dashboardSvc, headlessSvc, sts}

	if instance.Status.IsRunning() || instance.Status.IsCoreNodesReady() {
		deploy := generateDeployment(instance)
		resources = append(resources, deploy)

		listenerPorts, err := r.getAllListenersByAPI(sts)
		if err != nil {
			r.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedToGetListenerPorts", err.Error())
		}
		if listenersSvc := generateListenerService(instance, listenerPorts); listenersSvc != nil {
			resources = append(resources, listenersSvc)
		}
	}
	return resources, nil
}

func (r *EMQXReconciler) updateStatus(instance *appsv2alpha1.EMQX) (*appsv2alpha1.EMQX, error) {
	var emqxNodes []appsv2alpha1.EmqxNode
	var storeSts *appsv1.StatefulSet = &appsv1.StatefulSet{}

	if instance.Status.Conditions == nil {
		instance.Status.CurrentImage = instance.Spec.Image
		condition := appsv2alpha1.NewCondition(
			appsv2alpha1.ClusterCreating,
			corev1.ConditionTrue,
			"ClusterCreating",
			"Creating EMQX cluster",
		)
		instance.Status.SetCondition(*condition)
		return instance, nil
	}

	instance.Status.CoreReplicas = *instance.Spec.CoreTemplate.Spec.Replicas
	if isExistReplicant(instance) {
		instance.Status.ReplicantReplicas = *instance.Spec.ReplicantTemplate.Spec.Replicas
	} else {
		instance.Status.ReplicantReplicas = int32(0)
	}
	instance.Status.ReadyCoreReplicas = int32(0)
	instance.Status.ReadyReplicantReplicas = int32(0)

	err := r.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s-core", instance.Name), Namespace: instance.Namespace}, storeSts)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get store statefulSet")
	}

	emqxNodes, err = r.getNodeStatuesByAPI(storeSts)
	if err != nil {
		condition := appsv2alpha1.NewCondition(
			appsv2alpha1.ClusterRunning,
			corev1.ConditionFalse,
			"FailureToGetNodeStatus",
			err.Error(),
		)
		instance.Status.SetCondition(*condition)
		return instance, err
	}

	if emqxNodes != nil {
		instance.Status.EmqxNodes = emqxNodes

		for _, node := range emqxNodes {
			if strings.ToLower(node.NodeStatus) == "running" {
				if node.Role == "core" {
					instance.Status.ReadyCoreReplicas++
				}
				if node.Role == "replicant" {
					instance.Status.ReadyReplicantReplicas++
				}
			}
		}
	}

	switch instance.Status.Conditions[0].Type {
	default:
		if instance.Status.CurrentImage != instance.Spec.Image {
			instance.Status.CurrentImage = instance.Spec.Image
			condition := appsv2alpha1.NewCondition(
				appsv2alpha1.ClusterCoreUpdating,
				corev1.ConditionTrue,
				"ClusterCoreUpdating",
				"Updating core nodes in cluster",
			)
			instance.Status.SetCondition(*condition)
		} else if storeSts.Status.UpdatedReplicas == storeSts.Status.Replicas &&
			storeSts.Status.UpdateRevision == storeSts.Status.CurrentRevision &&
			instance.Status.CoreReplicas == instance.Status.ReadyCoreReplicas &&
			instance.Status.ReplicantReplicas == instance.Status.ReadyReplicantReplicas {
			condition := appsv2alpha1.NewCondition(
				appsv2alpha1.ClusterRunning,
				corev1.ConditionTrue,
				"ClusterReady",
				"All nodes are ready",
			)
			instance.Status.SetCondition(*condition)
		}
	case appsv2alpha1.ClusterCreating:
		instance.Status.CurrentImage = instance.Spec.Image
		condition := appsv2alpha1.NewCondition(
			appsv2alpha1.ClusterCoreUpdating,
			corev1.ConditionTrue,
			"ClusterCoreUpdating",
			"Updating core nodes in cluster",
		)
		instance.Status.SetCondition(*condition)
	case appsv2alpha1.ClusterCoreUpdating:
		// statefulSet already updated
		if storeSts.Spec.Template.Spec.Containers[0].Image == instance.Spec.Image &&
			storeSts.Status.ObservedGeneration == storeSts.Generation {
			// statefulSet is ready
			if storeSts.Status.UpdateRevision == storeSts.Status.CurrentRevision &&
				storeSts.Status.ReadyReplicas == storeSts.Status.Replicas {
				// core nodes is ready
				if instance.Status.ReadyCoreReplicas == instance.Status.CoreReplicas {
					condition := appsv2alpha1.NewCondition(
						appsv2alpha1.ClusterCoreReady,
						corev1.ConditionTrue,
						"ClusterCoreReady",
						"Core nodes is ready",
					)
					instance.Status.SetCondition(*condition)
				}
			}
		}
	}
	return instance, nil
}

func (r *EMQXReconciler) getNodeStatuesByAPI(obj client.Object) ([]appsv2alpha1.EmqxNode, error) {
	resp, body, err := r.Handler.RequestAPI(obj, "GET", username, password, dashboardPort, "api/v5/nodes")
	if err != nil {
		return nil, fmt.Errorf("failed to get listeners: %v", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get listener, status : %s, body: %s", resp.Status, body)
	}

	nodeStatuses := []appsv2alpha1.EmqxNode{}
	if err := json.Unmarshal(body, &nodeStatuses); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node statuses: %v", err)
	}
	return nodeStatuses, nil
}

type emqxGateway struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type emqxListener struct {
	Enable bool   `json:"enable"`
	ID     string `json:"id"`
	Bind   string `json:"bind"`
	Type   string `json:"type"`
}

func (r *EMQXReconciler) getAllListenersByAPI(obj client.Object) ([]corev1.ServicePort, error) {
	ports, err := r.getListenerPortsByAPI(obj, "api/v5/listeners")
	if err != nil {
		return nil, err
	}

	gateways, err := r.getGatewaysByAPI(obj)
	if err != nil {
		return nil, err
	}

	for _, gateway := range gateways {
		if strings.ToLower(gateway.Status) == "running" {
			apiPath := fmt.Sprintf("api/v5/gateway/%s/listeners", gateway.Name)
			gatewayPorts, err := r.getListenerPortsByAPI(obj, apiPath)
			if err != nil {
				return nil, err
			}
			ports = append(ports, gatewayPorts...)
		}
	}

	return ports, nil
}

func (r *EMQXReconciler) getGatewaysByAPI(obj client.Object) ([]emqxGateway, error) {
	resp, body, err := r.Handler.RequestAPI(obj, "GET", username, password, dashboardPort, "api/v5/gateway")
	if err != nil {
		return nil, fmt.Errorf("failed to get API %s: %v", "api/v5/gateway", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get API %s, status : %s, body: %s", "api/v5/gateway", resp.Status, body)
	}
	gateway := []emqxGateway{}
	if err := json.Unmarshal(body, &gateway); err != nil {
		return nil, fmt.Errorf("failed to parse gateway: %v", err)
	}
	return gateway, nil
}

func (r *EMQXReconciler) getListenerPortsByAPI(obj client.Object, apiPath string) ([]corev1.ServicePort, error) {
	resp, body, err := r.Handler.RequestAPI(obj, "GET", username, password, dashboardPort, apiPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get API %s: %v", apiPath, err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get API %s, status : %s, body: %s", apiPath, resp.Status, body)
	}
	ports := []corev1.ServicePort{}
	listeners := []emqxListener{}
	if err := json.Unmarshal(body, &listeners); err != nil {
		return nil, fmt.Errorf("failed to parse listeners: %v", err)
	}
	for _, listener := range listeners {
		if !listener.Enable {
			continue
		}

		var protocol corev1.Protocol
		compile := regexp.MustCompile(".*(udp|dtls|quic).*")
		if compile.MatchString(listener.Type) {
			protocol = corev1.ProtocolUDP
		} else {
			protocol = corev1.ProtocolTCP
		}

		_, strPort, _ := net.SplitHostPort(listener.Bind)
		intPort, _ := strconv.Atoi(strPort)

		ports = append(ports, corev1.ServicePort{
			Name:       strings.ReplaceAll(listener.ID, ":", "-"),
			Protocol:   protocol,
			Port:       int32(intPort),
			TargetPort: intstr.FromInt(intPort),
		})
	}
	return ports, nil
}

func isExistReplicant(instance *appsv2alpha1.EMQX) bool {
	return instance.Spec.ReplicantTemplate.Spec.Replicas != nil && *instance.Spec.ReplicantTemplate.Spec.Replicas > 0
}
