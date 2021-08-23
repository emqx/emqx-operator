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

	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/emqx/emqx-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// EmqxReconciler reconciles a Emqx object
type EmqxReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.emqx.io,resources=emqxes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Emqx object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *EmqxReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("Reconcile start")

	instance := &v1alpha1.Emqx{}

	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
	}

	// create pvc
	storageList := []string{EMQX_LOG_NAME, EMQX_DATA_NAME}

	for _, item := range storageList {
		pvc := makePvcOwnerReference(instance, item)
		op, err := controllerutil.CreateOrUpdate(ctx, r.Client, pvc, func() error {
			pvc.Spec = makePvcSpec(instance, item)
			return nil
		})
		if err != nil {
			log.Error(err, "Pvc reconcile failed")
		} else {
			log.Info("Pvc successfully reconciled", "operation", op)
		}
	}

	// create secret
	log.Info("Start create license config")
	secret := *makeSecretOwnerReference(instance)
	secretOp, err := controllerutil.CreateOrUpdate(ctx, r.Client, &secret, func() error {
		secret.StringData = makeSecretSpec(instance)
		return nil
	})
	if err != nil {
		log.Error(err, "Secret reconcile failed")
	} else {
		log.Info("Secret successfully reconciled", "operation", secretOp)
	}

	// create service
	log.Info("start create service")
	service := *makeServiceOwnerReference(instance)
	serviceOp, err := controllerutil.CreateOrUpdate(ctx, r.Client, &service, func() error {
		service.Spec = makeServiceSpec(instance)
		return nil
	})
	if err != nil {
		log.Error(err, "ServiceOp reconcile failed")
	} else {
		log.Info("ServiceOp successfully reconciled", "operation", serviceOp)
	}

	// create  statefulset
	log.Info("start create statefulset")
	statefulset, _ := makeStatefulOwnerReference(instance)
	statefulsetOp, err := controllerutil.CreateOrUpdate(ctx, r.Client, statefulset, func() error {
		statefulset.Spec = *makeStatefulSetSpec(instance)
		return nil
	})
	if err != nil {
		log.Error(err, "ServiceOp reconcile failed")
	} else {
		log.Info("ServiceOp successfully reconciled", "operation", statefulsetOp)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EmqxReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Emqx{}).
		Complete(r)
}
