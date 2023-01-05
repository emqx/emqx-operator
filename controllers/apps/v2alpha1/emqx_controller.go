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

package v2alpha1

import (
	"context"
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
	"sigs.k8s.io/controller-runtime/pkg/manager"

	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	"github.com/emqx/emqx-operator/pkg/apiclient"
	"github.com/emqx/emqx-operator/pkg/handler"
	appsv1 "k8s.io/api/apps/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
)

const EMQXContainerName string = "emqx"

// EMQXReconciler reconciles a EMQX object
type EMQXReconciler struct {
	*handler.Handler
	APIClient     *apiclient.APIClient
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
}

func NewEMQXReconciler(mgr manager.Manager) *EMQXReconciler {
	return &EMQXReconciler{
		Handler:       handler.NewHandler(mgr),
		APIClient:     apiclient.NewAPIClient(mgr),
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
	_ = log.FromContext(ctx)

	instance := &appsv2alpha1.EMQX{}
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Create Resources
	resources, err := r.createResources(instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := r.CreateOrUpdateList(instance, r.Scheme, resources); err != nil {
		if k8sErrors.IsConflict(err) {
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	// Update EMQX Custom Resource's status
	instance, err = r.updateStatus(instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Client.Status().Update(ctx, instance); err != nil {
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
	nodeCookie := generateNodeCookieSecret(instance)
	bootstrapUser := generateBootstrapUserSecret(instance)
	bootstrapConfig := generateBootstrapConfigMap(instance)
	if instance.Status.IsCreating() {
		resources = append(resources, nodeCookie, bootstrapUser, bootstrapConfig)
	}

	dashboardSvc := generateDashboardService(instance)
	headlessSvc := generateHeadlessService(instance)
	sts := generateStatefulSet(instance)
	sts = updateStatefulSetForNodeCookie(sts, nodeCookie)
	sts = updateStatefulSetForBootstrapUser(sts, bootstrapUser)
	sts = updateStatefulSetForBootstrapConfig(sts, bootstrapConfig)
	resources = append(resources, dashboardSvc, headlessSvc, sts)

	if instance.Status.IsRunning() || instance.Status.IsCoreNodesReady() {
		deploy := generateDeployment(instance)
		deploy = updateDeploymentForNodeCookie(deploy, nodeCookie)
		deploy = updateDeploymentForBootstrapConfig(deploy, bootstrapConfig)
		resources = append(resources, deploy)

		listenerPorts, err := newRequestAPI(r, instance).getAllListenersByAPI(sts)
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
	var existedSts *appsv1.StatefulSet = &appsv1.StatefulSet{}
	var existedDeploy *appsv1.Deployment = &appsv1.Deployment{}
	var err error

	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.CoreTemplate.Name, Namespace: instance.Namespace}, existedSts)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return instance, nil
		}
		return nil, emperror.Wrap(err, "failed to get existed statefulSet")
	}

	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.ReplicantTemplate.Name, Namespace: instance.Namespace}, existedDeploy)
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, emperror.Wrap(err, "failed to get existed deployment")
	}

	emqxNodes, err = newRequestAPI(r, instance).getNodeStatuesByAPI(existedSts)
	if err != nil {
		r.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedToGetNodeStatuses", err.Error())
	}

	emqxStatusMachine := newEMQXStatusMachine(instance)
	emqxStatusMachine.CheckNodeCount(emqxNodes)
	emqxStatusMachine.NextStatus(existedSts, existedDeploy)
	return emqxStatusMachine.GetEMQX(), nil
}

func (r *EMQXReconciler) getBootstrapUser(instance *appsv2alpha1.EMQX) (username, password string, err error) {
	secret := &corev1.Secret{}
	if err = r.Client.Get(context.TODO(), types.NamespacedName{Name: instance.NameOfBootStrapUser(), Namespace: instance.Namespace}, secret); err != nil {
		return "", "", err
	}

	data, ok := secret.Data["bootstrap_user"]
	if !ok {
		return "", "", emperror.Errorf("the secret does not contain the bootstrap_user")
	}

	str := string(data)
	index := strings.Index(str, ":")

	return str[:index], str[index+1:], nil
}
