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
	"fmt"
	"strings"
	"time"

	emperror "emperror.dev/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	r.EventRecorder.Event(instance, corev1.EventTypeNormal, "test", "test")

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
	var resources []client.Object
	bootstrap_user := generateBootstrapUserSecret(instance)
	if instance.Status.IsCreating() {
		resources = append(resources, bootstrap_user)
	}

	dashboardSvc := generateDashboardService(instance)
	headlessSvc := generateHeadlessService(instance)
	sts := generateStatefulSet(instance)
	sts = updateStatefulSetForBootstrapUser(sts, bootstrap_user)
	resources = append(resources, dashboardSvc, headlessSvc, sts)

	if instance.Status.IsRunning() || instance.Status.IsCoreNodesReady() {
		deploy := generateDeployment(instance)
		resources = append(resources, deploy)

		username, password, err := r.getBootstrapUser(instance)
		if err != nil {
			r.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedToGetBootStrapUserSecret", err.Error())
		}

		listenerPorts, err := (&requestAPI{
			username: username,
			password: password,
			port:     dashboardPort,
			Handler:  r.Handler,
		}).getAllListenersByAPI(sts)
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
	var emqxNodes []appsv2alpha1.EMQXNode
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

	instance.Status.CoreNodeReplicas = *instance.Spec.CoreTemplate.Spec.Replicas
	if isExistReplicant(instance) {
		instance.Status.ReplicantNodeReplicas = *instance.Spec.ReplicantTemplate.Spec.Replicas
	} else {
		instance.Status.ReplicantNodeReplicas = int32(0)
	}
	instance.Status.CoreNodeReadyReplicas = int32(0)
	instance.Status.ReplicantNodeReadyReplicas = int32(0)

	err := r.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s-core", instance.Name), Namespace: instance.Namespace}, storeSts)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get store statefulSet")
	}

	username, password, err := r.getBootstrapUser(instance)
	if err != nil {
		r.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedToGetBootStrapUserSecret", err.Error())
	}
	emqxNodes, err = (&requestAPI{
		username: username,
		password: password,
		port:     dashboardPort,
		Handler:  r.Handler,
	}).getNodeStatuesByAPI(storeSts)
	if err != nil {
		r.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedToGetNodeStatuses", err.Error())
	}

	if emqxNodes != nil {
		instance.Status.EMQXNodes = emqxNodes

		for _, node := range emqxNodes {
			if strings.ToLower(node.NodeStatus) == "running" {
				if node.Role == "core" {
					instance.Status.CoreNodeReadyReplicas++
				}
				if node.Role == "replicant" {
					instance.Status.ReplicantNodeReadyReplicas++
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
			instance.Status.CoreNodeReplicas == instance.Status.CoreNodeReadyReplicas &&
			instance.Status.ReplicantNodeReplicas == instance.Status.ReplicantNodeReadyReplicas {
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
				if instance.Status.CoreNodeReadyReplicas == instance.Status.CoreNodeReplicas {
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

func (r *EMQXReconciler) getBootstrapUser(instance *appsv2alpha1.EMQX) (username, password string, err error) {
	secret := &corev1.Secret{}
	if err = r.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s-bootstrap-user", instance.Name), Namespace: instance.Namespace}, secret); err != nil {
		return "", "", err
	}

	data, ok := secret.Data["bootstrap_user"]
	if !ok {
		return "", "", fmt.Errorf("the secret does not contain the bootstrap_user")
	}

	str := string(data)
	index := strings.Index(str, ":")

	return str[:index], str[index+1:], nil
}

func isExistReplicant(instance *appsv2alpha1.EMQX) bool {
	return instance.Spec.ReplicantTemplate.Spec.Replicas != nil && *instance.Spec.ReplicantTemplate.Spec.Replicas > 0
}
