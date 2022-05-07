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
	"time"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
)

var (
	log = logf.Log.WithName("emqx-controller")
	// reconcileTime is the delay between reconciliations. Defaults to 60s.
	reconcileTime = time.Duration(30) * time.Second
)

//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods/exec,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=impersonate;get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxbrokers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxbrokers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxbrokers/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxenterprises,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxenterprises/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxenterprises/finalizers,verbs=update
//+kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EmqxBroker object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (handler *Handler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the EMQX Cluster instance
	emqx, err := handler.getEmqx(req.Namespace, req.Name)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if err := handler.Ensure(emqx); err != nil {
		if err.Error() == "need requeue" {
			return reconcile.Result{RequeueAfter: 20 * time.Second}, nil
		}
		handler.eventsCli.Event(emqx, corev1.EventTypeWarning, "Reconcile", err.Error())
		emqx.SetFailedCondition(err.Error())
		_ = handler.updateEmqxStatus(emqx)
		return reconcile.Result{}, err
	}

	emqx.SetRunningCondition("Cluster ok")
	return reconcile.Result{RequeueAfter: reconcileTime}, nil
}

func (handler *Handler) getEmqx(Namespace, Name string) (v1beta3.Emqx, error) {
	broker := &v1beta3.EmqxBroker{}
	err := handler.client.Get(
		context.TODO(),
		types.NamespacedName{
			Name:      Name,
			Namespace: Namespace,
		},
		broker,
	)
	if err != nil && k8sErrors.IsNotFound(err) {
		enterprise := &v1beta3.EmqxEnterprise{}
		err := handler.client.Get(
			context.TODO(),
			types.NamespacedName{
				Name:      Name,
				Namespace: Namespace,
			},
			enterprise,
		)
		return enterprise, err
	}
	return broker, err
}

func (handler *Handler) updateEmqxStatus(emqx v1beta3.Emqx) error {
	emqx.DescConditionsByTime()
	err := handler.client.Status().Update(context.TODO(), emqx)
	if err != nil {
		handler.logger.WithValues(
			"kind", emqx.GetKind(),
			"apiVersion", emqx.GetAPIVersion(),
			"namespace", emqx.GetNamespace(),
			"name", emqx.GetName(),
			"conditions", emqx.GetConditions(),
		).Error(err, "Update emqx broker status unsuccessfully")
		return err
	}
	handler.logger.WithValues(
		"kind", emqx.GetKind(),
		"apiVersion", emqx.GetAPIVersion(),
		"namespace", emqx.GetNamespace(),
		"name", emqx.GetName(),
		"conditions", emqx.GetConditions(),
	).V(3).Info("Update emqx broker status successfully")
	return nil
}
