/*
Copyright 2026.

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

package controller

import (
	"context"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
)

// EMQXAPIKeyReconciler reconciles an EMQXAPIKey object.
type EMQXAPIKeyReconciler struct {
	Client client.Client
}

func NewEMQXAPIKeyReconciler(mgr manager.Manager) *EMQXAPIKeyReconciler {
	return &EMQXAPIKeyReconciler{
		Client: mgr.GetClient(),
	}
}

// +kubebuilder:rbac:groups=apps.emqx.io,resources=emqxapikeys,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.emqx.io,resources=emqxapikeys/status,verbs=get;update;patch

// Reconcile is scaffolded for future EMQX API key synchronization.
func (r *EMQXAPIKeyReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("Reconcile EMQX API key")

	apiKey := &crd.EMQXAPIKey{}
	if err := r.Client.Get(ctx, request.NamespacedName, apiKey); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EMQXAPIKeyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&crd.EMQXAPIKey{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Named("emqxapikey").
		Complete(r)
}
