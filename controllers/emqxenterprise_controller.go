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

package controllers

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/emqx/emqx-operator/api/v1alpha2"
	appsv1alpha2 "github.com/emqx/emqx-operator/api/v1alpha2"
	"github.com/emqx/emqx-operator/pkg/cache"
	"github.com/emqx/emqx-operator/pkg/client/k8s"
	"github.com/emqx/emqx-operator/pkg/service"
)

// EmqxEnterpriseReconciler reconciles a EmqxEnterprise object
type EmqxEnterpriseReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme

	Handler *EmqxBrokerClusterHandler
}

func (*EmqxEnterpriseReconciler) New(mgr manager.Manager) *EmqxEnterpriseReconciler {
	// Create kubernetes service.
	k8sService := k8s.New(mgr.GetClient(), log)

	// Create the emqx clients
	// TODO

	// Create internal services.
	eService := service.NewEmqxBrokerClusterKubeClient(k8sService, log)
	// TODO eChecker

	// TODO eHealer

	handler := &EmqxBrokerClusterHandler{
		k8sServices: k8sService,
		eService:    eService,
		metaCache:   new(cache.MetaMap),
		eventsCli:   k8s.NewEvent(mgr.GetEventRecorderFor("emqx-operator"), log),
		logger:      log,
	}

	return &EmqxEnterpriseReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme(), Handler: handler}
}

//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxenterprises,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxenterprises/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxenterprises/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EmqxEnterprise object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *EmqxEnterpriseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling EMQ X Cluster")

	// Fetch the EMQ X Cluster instance
	instance := &v1alpha2.EmqxEnterprise{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("EMQ X Cluster delete")
			instance.Namespace = req.NamespacedName.Namespace
			instance.Name = req.NamespacedName.Name
			r.Handler.metaCache.Del(instance)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	reqLogger.V(5).Info(fmt.Sprintf("EMQ X Cluster Spec:\n %+v", instance))

	if err = r.Handler.Do(instance); err != nil {
		if err.Error() == needRequeueMsg {
			return reconcile.Result{RequeueAfter: 20 * time.Second}, nil
		}
		reqLogger.Error(err, "Reconcile handler")
		return reconcile.Result{}, err
	}

	// TODO
	// if err = r.handler.eChecker.CheckEmqxBrokerReadyReplicas(instance); err != nil {
	// 	reqLogger.Info(err.Error())
	// 	return reconcile.Result{RequeueAfter: 20 * time.Second}, nil
	// }

	return reconcile.Result{RequeueAfter: time.Duration(reconcileTime) * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EmqxEnterpriseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1alpha2.EmqxEnterprise{}).
		Complete(r)
}
