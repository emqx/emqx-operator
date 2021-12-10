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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/emqx/emqx-operator/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ reconcile.Reconciler = &EmqxBrokerReconciler{}

// EmqxBrokerReconciler reconciles a EmqxBroker object
type EmqxBrokerReconciler struct {
	Handler
}

func NewEmqxBrokerReconciler(mgr manager.Manager) *EmqxBrokerReconciler {
	return &EmqxBrokerReconciler{*NewHandler(mgr)}
}

// SetupWithManager sets up the controller with the Manager.
func (r *EmqxBrokerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.EmqxBroker{}).
		Complete(r)
}
