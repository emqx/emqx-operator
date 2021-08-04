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
	"reflect"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"Emqx/api/v1alpha1"
	customv1alpha1 "Emqx/api/v1alpha1"

	pkgerr "github.com/pkg/errors"
)

const (
	emqxName               = "emqx"
	emqxlicName            = "emqx-lic"
	emqxlogName            = "emqx-log-dir"
	emqxdataName           = "emqx-data-dir"
	emqxloadmodulesName    = "emqx-loaded-modules"
	emqxenvName            = "cloud-env"
	emqxlicDir             = "/opt/emqx/etc/emqx.lic"
	emqxlicSubPath         = "emqx.lic"
	emqxdataDir            = "/opt/emqx/data/mnesia"
	emqxlogDir             = "/opt/emqx/log"
	emqxloadmodulesDir     = "/opt/emqx/data/loaded_modules"
	emqxloadmodulesSubpath = "loaded_modules"

	serviceTcpName = "tcp"
	serviceTcpPort = 1883

	serviceTcpsName = "tcps"
	serviceTcpsPort = 8883

	serviceWsName = "ws"
	serviceWsPort = 8083

	serviceWssName = "wss"
	serviceWssPort = 8084
)

type StorageList map[StorageKey]StorageValue

type StorageKey string

type StorageValue string

// BrokerReconciler reconciles a Broker object
type BrokerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=custom.emqx.io,resources=brokers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=custom.emqx.io,resources=brokers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=custom.emqx.io,resources=brokers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Broker object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *BrokerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	log.Log.Info("Emqx Reconcile start")

	instance := &v1alpha1.Broker{}
	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
	}

	// create pv and pvc
	storageList := map[StorageKey]StorageValue{
		emqxlogName:  "/opt/emqx-log",
		emqxdataName: "/opt/emqx-data",
	}
	for k, v := range storageList {
		pv := makePvFromSpec(instance, k, v)
		err = r.createOrUpdate(pv.Name, pv.Namespace, pv)
		if err != nil && errors.IsAlreadyExists(err) == false {
			return reconcile.Result{}, pkgerr.Wrap(err, "creating pv failed")
		}

		pvc := makePvcFromSpec(instance, k)
		err = r.createOrUpdate(pvc.Name, pvc.Namespace, pvc)
		if err != nil && errors.IsAlreadyExists(err) == false {
			return reconcile.Result{}, pkgerr.Wrap(err, "creating pvc failed")
		}
	}
	// create secret
	log.Log.Info("Start create license config")
	configSecret := makeSecretConfigFromSpec(instance)

	err = r.createOrUpdate(configSecret.Name, configSecret.Namespace, configSecret)
	if err != nil && errors.IsAlreadyExists(err) == false {
		return reconcile.Result{}, pkgerr.Wrap(err, "creating  config Secret failed")
	}

	// log.Log.Info("Strart create configmap")
	// // create configmap
	// modules := makeModulesFromSpec(instance)
	// err = r.createOrUpdate(modules.Name, modules.Namespace, modules)
	// if err != nil && errors.IsAlreadyExists(err) == false {
	// 	return reconcile.Result{}, pkgerr.Wrap(err, "creating  modules configmap failed")
	// }

	log.Log.Info("start create env config")
	env := makeEnvFromSpec(instance)
	err = r.createOrUpdate(env.Name, env.Namespace, env)
	if err != nil && errors.IsAlreadyExists(err) == false {
		return reconcile.Result{}, pkgerr.Wrap(err, "creating  env configmap failed")
	}

	// create service
	log.Log.Info("start create service")
	service := makeServiceFromSpec(instance)
	err = r.createOrUpdate(service.Name, service.Namespace, service)
	if err != nil && errors.IsAlreadyExists(err) == false {
		return reconcile.Result{}, pkgerr.Wrap(err, "creating  service failed")
	}

	// create  statefulset
	log.Log.Info("start create statefulset")
	statefulset, _ := makeStatefulSet(instance)
	err = r.createOrUpdate(statefulset.Name, statefulset.Namespace, statefulset)
	if err != nil && errors.IsAlreadyExists(err) == false {
		return reconcile.Result{}, pkgerr.Wrap(err, "creating  statefulset failed")
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BrokerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&customv1alpha1.Broker{}).
		Complete(r)
}

func (r *BrokerReconciler) createOrUpdate(name string, namespace string, object client.Object) error {
	found := object.DeepCopyObject()
	key := types.NamespacedName{Name: name, Namespace: namespace}
	err := r.Client.Get(context.TODO(), key, object)
	if err != nil && errors.IsNotFound(err) {
		// define a new resource
		err = r.Client.Create(context.TODO(), object)
		if err != nil {
			return pkgerr.Wrap(err, "failed to create object")
		}
		log.Log.Info("created", "object", reflect.TypeOf(object))
		return nil
	} else if err != nil {
		return pkgerr.Wrap(err, "failed to retrieve object")
	} else {
		a := meta.NewAccessor()
		resourceVersion, err := a.ResourceVersion(found)
		if err != nil {
			return pkgerr.Wrap(err, "coudln't extract resource version of object")
		}
		err = a.SetResourceVersion(object, resourceVersion)
		if err != nil {
			return pkgerr.Wrap(err, "coudln't set resource version on object")
		}
		err = r.Client.Update(context.TODO(), object)
		if err != nil {
			return pkgerr.Wrap(err, "failed to update object")
		}
		log.Log.Info("updated", "object", reflect.TypeOf(object))
		return nil
	}
}
