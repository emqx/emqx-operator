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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1beta3 "github.com/emqx/emqx-operator/apis/apps/v1beta3"
)

var _ reconcile.Reconciler = &EmqxBrokerReconciler{}

// EmqxEnterpriseReconciler reconciles a EmqxEnterprise object
type EmqxEnterpriseReconciler struct {
	Handler
}

func NewEmqxEnterpriseReconciler(mgr manager.Manager) *EmqxEnterpriseReconciler {
	return &EmqxEnterpriseReconciler{*NewHandler(mgr)}
}

// SetupWithManager sets up the controller with the Manager.
func (r *EmqxEnterpriseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1beta3.EmqxEnterprise{}).
		Complete(r)
}
