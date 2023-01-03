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

package v1beta4

import (
	"context"
	"time"

	emperror "emperror.dev/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	"github.com/emqx/emqx-operator/pkg/handler"
)

var _ reconcile.Reconciler = &EmqxBrokerReconciler{}

type EmqxReconciler struct {
	*handler.Handler
	Scheme *runtime.Scheme
	record.EventRecorder

	config    *rest.Config
	clientset *kubernetes.Clientset
}

func NewEmqxReconciler(mgr manager.Manager) *EmqxReconciler {
	return &EmqxReconciler{
		Handler:       handler.NewHandler(mgr),
		Scheme:        mgr.GetScheme(),
		EventRecorder: mgr.GetEventRecorderFor("emqx-controller"),
		config:        mgr.GetConfig(),
		clientset:     kubernetes.NewForConfigOrDie(mgr.GetConfig()),
	}
}

func (r *EmqxReconciler) Do(ctx context.Context, instance appsv1beta4.Emqx) (ctrl.Result, error) {
	emqxStatusMachine := newEmqxStatusMachine(instance, r, ctx)
	if requeue := emqxStatusMachine.currentStatus.nextStatus(instance); requeue != nil {
		return processRequeue(requeue)
	}

	// create resource start
	var resources []client.Object
	bootstrap_user := generateBootstrapUserSecret(instance)
	plugins, err := r.createInitPluginList(instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !instance.GetStatus().IsInitResourceReady() {
		resources = append(resources, bootstrap_user)
		resources = append(resources, plugins...)
		err := r.createInitResources(instance, resources)
		if err != nil {
			r.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedCreateOrUpdate", err.Error())
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	var license *corev1.Secret
	if enterprise, ok := instance.(*appsv1beta4.EmqxEnterprise); ok {
		if enterprise.Spec.License.SecretName != "" {
			license = &corev1.Secret{}
			if err := r.Client.Get(
				context.Background(),
				types.NamespacedName{
					Name:      enterprise.Spec.License.SecretName,
					Namespace: instance.GetNamespace(),
				},
				license,
			); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			license = generateLicense(instance)
		}
	}
	if license != nil {
		resources = append(resources, license)
	}

	var listenerPorts []corev1.ServicePort
	if instance.GetStatus().IsRunning() {
		listenerPorts, _ = r.getListenerPortsByAPI(instance)
	}
	if svc := generateService(instance, listenerPorts...); svc != nil {
		resources = append(resources, svc)
	}
	headlessSvc := generateHeadlessService(instance, listenerPorts...)
	acl := generateEmqxACL(instance)

	sts := generateStatefulSet(instance)
	sts = updateStatefulSetForACL(sts, acl)
	sts = updateStatefulSetForPluginsConfig(sts, generateDefaultPluginsConfig(instance))
	sts = updateStatefulSetForLicense(sts, license)
	sts = updateStatefulSetForBootstrapUser(sts, bootstrap_user)

	if enterprise, ok := instance.(*appsv1beta4.EmqxEnterprise); ok {
		if enterprise.Spec.EmqxBlueGreenUpdate != nil {
			var err error
			sts, err = r.getNewStatefulSet(instance, sts)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	resources = append(resources, acl, headlessSvc, sts)

	if err := r.CreateOrUpdateList(instance, r.Scheme, resources); err != nil {
		if k8sErrors.IsConflict(err) {
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		r.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedCreateOrUpdate", err.Error())
		return ctrl.Result{}, err
	}

	enterprise, ok := instance.(*appsv1beta4.EmqxEnterprise)
	if ok && enterprise.Status.EmqxBlueGreenUpdateStatus != nil {
		if err := r.syncStatefulSet(instance, enterprise.Status.EmqxBlueGreenUpdateStatus.EvacuationsStatus); err != nil {
			if k8sErrors.IsConflict(err) {
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			return ctrl.Result{}, emperror.Wrap(err, "sync statefulSet failed")
		}
	}

	if requeue := emqxStatusMachine.currentStatus.nextStatus(instance); requeue != nil {
		return processRequeue(requeue)
	}
	return ctrl.Result{RequeueAfter: 20 * time.Second}, nil
}

func (r *EmqxReconciler) createInitResources(instance appsv1beta4.Emqx, initResources []client.Object) error {
	if err := r.CreateOrUpdateList(instance, r.Scheme, initResources); err != nil {
		r.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedCreateOrUpdate", err.Error())
		return err
	}
	return nil
}

func (r *EmqxReconciler) createInitPluginList(instance appsv1beta4.Emqx) ([]client.Object, error) {
	pluginsList := &appsv1beta4.EmqxPluginList{}
	err := r.Client.List(context.Background(), pluginsList, client.InNamespace(instance.GetNamespace()))
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}
	initPluginsList := generateInitPluginList(instance, pluginsList)
	defaultPluginsConfig := generateDefaultPluginsConfig(instance)
	return append([]client.Object{defaultPluginsConfig}, initPluginsList...), nil
}
