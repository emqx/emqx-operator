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

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/emqx/emqx-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// EmqxReconciler reconciles a Emqx object
type EmqxReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
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
	log := r.Log.WithValues("emqx", req.NamespacedName)

	instance := &v1alpha1.Emqx{}

	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("the instance  is not found")
			return ctrl.Result{}, nil
		}

	}
	if err := createOrUpdatePvc(ctx, r, instance, req); err != nil {
		log.Error(err, "Create or update pvc error")
		return ctrl.Result{}, nil
	}

	if err := createOrUpdateSecret(ctx, r, instance, req); err != nil {
		log.Error(err, "Create or update secret error")
		return ctrl.Result{}, nil
	}

	if err := createOrUpdateService(ctx, r, instance, req); err != nil {
		log.Error(err, "Create or update service error")
		return ctrl.Result{}, nil
	}

	if err := createOrUpdateStatefulset(ctx, r, instance, req); err != nil {
		log.Error(err, "Create or update statefulset error")
		return ctrl.Result{}, nil
	}

	log.Info("Emqx :" + instance.String() + "reconciled successfully")

	return ctrl.Result{}, err

}

// SetupWithManager sets up the controller with the Manager.
func (r *EmqxReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Emqx{}).
		Complete(r)
}

func createOrUpdatePvc(ctx context.Context, r *EmqxReconciler, instance *v1alpha1.Emqx, req ctrl.Request) error {
	log := r.Log.WithValues("function", "reconcile pvc")

	storageList := []string{EMQX_LOG_NAME, EMQX_DATA_NAME}

	for _, item := range storageList {
		pvc := &v1.PersistentVolumeClaim{}

		pvcName := fmt.Sprintf("%s-%s", "pvc", item)
		pvcNamespacedName := resloveNameSpacedName(req, pvcName)

		err := r.Get(ctx, pvcNamespacedName, pvc)

		if err == nil || errors.IsNotFound(err) {
			log.Info("Set pvc reference")
			pvc.Namespace = instance.Namespace
			pvc.Name = pvcName
			if err := controllerutil.SetControllerReference(instance, pvc, r.Scheme); err != nil {
				log.Error(err, "Set pvc reference error")
				return err
			}
			op, err := controllerutil.CreateOrUpdate(ctx, r.Client, pvc, func() error {
				pvc.Spec = makePvcSpec(instance, item)
				return nil
			})
			if err != nil {
				log.Error(err, "Pvc reconcile failed")
				return err
			} else {
				log.Info("Pvc reconciled successfully", "operation", op)
			}
		}

		if err != nil && !errors.IsNotFound(err) {
			log.Error(err, "Query pvc error")
			return err
		}
	}
	return nil
}

func createOrUpdateSecret(ctx context.Context, r *EmqxReconciler, instance *v1alpha1.Emqx, req ctrl.Request) error {
	log := r.Log.WithValues("function", "reconcile secret")

	secret := &v1.Secret{}

	secretNamespacedName := resloveNameSpacedName(req, EMQX_LIC_NAME)

	err := r.Get(ctx, secretNamespacedName, secret)

	if err == nil || errors.IsNotFound(err) {
		log.Info("Set secret reference")
		secret.Namespace = instance.Namespace
		secret.Name = EMQX_LIC_NAME
		if err := controllerutil.SetControllerReference(instance, secret, r.Scheme); err != nil {
			log.Error(err, "Set secret reference error")
			return err
		}

		op, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {

			secret.Type = "Opaque"
			secret.StringData = makeSecretStringData(instance)
			return nil
		})
		if err != nil {
			log.Error(err, "Secret reconcile failed")
			return err
		} else {
			log.Info("Secret reconciled successfully", "operation", op)
		}
		return nil
	}

	if err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Query secret error")
		return err
	}
	return nil
}

func createOrUpdateService(ctx context.Context, r *EmqxReconciler, instance *v1alpha1.Emqx, req ctrl.Request) error {
	log := r.Log.WithValues("function", "reconcile service")

	svcName := []string{"svc", "headless"}
	svcSpecMap := makeServiceSpec(instance)
	for _, v := range svcName {
		svc := &v1.Service{}

		serviceNamespaced := resloveNameSpacedName(req, fmt.Sprintf("%s-%s", instance.Name, v))

		err := r.Get(ctx, serviceNamespaced, svc)

		// update svc phase:
		// 1. delete primary svc
		// 2. create the new svc
		if err == nil {
			log.Info("Delete the old service")
			err := r.Delete(ctx, svc)
			if err != nil {
				log.Error(err, "Delelet service error ")
				return err
			}
			// ensure the resourceVersion to be the correct version
			svc = &v1.Service{}
			log.Info("Set service reference")
			svc.Namespace = instance.Namespace
			svc.Name = fmt.Sprintf("%s-%s", instance.Name, v)
			if err := controllerutil.SetControllerReference(instance, svc, r.Scheme); err != nil {
				log.Error(err, "Set service reference error")
				return err
			}
			op, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
				svc.Spec = *svcSpecMap[v]
				return nil
			})
			if err != nil {
				log.Error(err, "Service reconcile failed")
				return err
			} else {
				log.Info("Service reconciled successfully", "operation", op)
			}
		} else if errors.IsNotFound(err) {
			log.Info("Set service reference")
			svc.Namespace = instance.Namespace
			svc.Name = fmt.Sprintf("%s-%s", instance.Name, v)
			if err := controllerutil.SetControllerReference(instance, svc, r.Scheme); err != nil {
				log.Error(err, "Set service reference error")
				return err
			}
			op, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
				svc.Spec = *svcSpecMap[v]
				return nil
			})
			if err != nil {
				log.Error(err, "Service reconcile failed")
				return err
			} else {
				log.Info("Service reconciled successfully", "operation", op)
			}
		}
		if err != nil && !errors.IsNotFound(err) {
			log.Error(err, "Query service error")
			return err
		}
	}
	return nil
}

func createOrUpdateStatefulset(ctx context.Context, r *EmqxReconciler, instance *v1alpha1.Emqx, req ctrl.Request) error {
	log := r.Log.WithValues("function", "reconcile statefulset")

	statefulset := &appsv1.StatefulSet{}

	err := r.Get(ctx, req.NamespacedName, statefulset)

	if err == nil || errors.IsNotFound(err) {
		statefulset.Name = instance.Name
		statefulset.Namespace = instance.Namespace
		log.Info("Set statefulset reference")
		if err := controllerutil.SetControllerReference(instance, statefulset, r.Scheme); err != nil {
			log.Error(err, "Set statefulset reference error")
			return err
		}
		op, err := controllerutil.CreateOrUpdate(ctx, r.Client, statefulset, func() error {
			statefulset.Spec = *makeStatefulSetSpec(instance)
			return nil
		})
		if err != nil {
			log.Error(err, "Statefulset reconcile failed")
			return err
		} else {
			log.Info("Statefulset reconciled successfully", "operation", op)
		}
		return nil
	}

	if err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Query statefulset error")
		return err
	}

	return nil
}

// reslove the namespacedname to correct scope
func resloveNameSpacedName(req ctrl.Request, s string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: req.Namespace,
		Name:      s,
	}
}
