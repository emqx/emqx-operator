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
	"reflect"
	"time"

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
	if !instance.IsPluginInitialized() {
		condition, err := r.initializedPluginList(instance)
		if condition != nil {
			instance.SetCondition(*condition)
			_ = r.Client.Status().Update(ctx, instance)
		}
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	var resources []client.Object
	var license *corev1.Secret
	if instance.GetTemplate().Spec.EmqxContainer.EmqxLicense.SecretName != "" {
		if err := r.Client.Get(
			context.Background(),
			types.NamespacedName{
				Name:      instance.GetTemplate().Spec.EmqxContainer.EmqxLicense.SecretName,
				Namespace: instance.GetNamespace(),
			},
			license,
		); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		license = generateLicense(instance)
	}

	if license != nil {
		resources = append(resources, license)
	}

	var listenerPorts []corev1.ServicePort
	if instance.IsRunning() {
		listenerPorts, _ = r.getListenerPortsByAPI(instance)
	}
	headlessSvc := generateHeadlessService(instance, listenerPorts...)
	acl := generateEmqxACL(instance)

	sts := generateStatefulSet(instance)
	sts = updateStatefulSetForACL(sts, acl)
	sts = updateStatefulSetForPluginsConfig(sts, generateDefaultPluginsConfig(instance))
	sts = updateStatefulSetForLicense(sts, license)

	if enterprise, ok := instance.(*appsv1beta4.EmqxEnterprise); ok {
		if !reflect.ValueOf(enterprise.Spec.EmqxBlueGreenUpdate).IsZero() {
			var err error
			sts, err = r.getNewStatefulSet(instance, sts)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	resources = append(resources, acl, headlessSvc, sts)

	if err := r.CreateOrUpdateList(instance, r.Scheme, resources); err != nil {
		r.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedCreateOrUpdate", err.Error())
		condition := appsv1beta4.NewCondition(
			appsv1beta4.ConditionRunning,
			corev1.ConditionFalse,
			"FailedCreateOrUpdate",
			err.Error(),
		)
		instance.SetCondition(*condition)
		_ = r.Client.Status().Update(ctx, instance)
		return ctrl.Result{}, err
	}

	status, err := r.updateEmqxStatus(instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	instance.SetStatus(status)
	_ = r.Client.Status().Update(ctx, instance)

	if _, ok := instance.(*appsv1beta4.EmqxEnterprise); ok {
		if err := r.syncStatefulSet(instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	svc := generateService(instance, listenerPorts...)
	latestReadySts, err := r.getLatestReadyStatefulSet(instance, true)
	if err != nil {
		return ctrl.Result{}, err
	}
	selector := svc.Spec.Selector
	selector["controller-revision-hash"] = latestReadySts.Status.CurrentRevision
	svc.Spec.Selector = selector

	if err := r.CreateOrUpdateList(instance, r.Scheme, []client.Object{svc}); err != nil {
		r.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedCreateOrUpdate", err.Error())
		condition := appsv1beta4.NewCondition(
			appsv1beta4.ConditionRunning,
			corev1.ConditionFalse,
			"FailedCreateOrUpdate",
			err.Error(),
		)
		instance.SetCondition(*condition)
		_ = r.Client.Status().Update(ctx, instance)
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: time.Duration(20) * time.Second}, nil
}

func (r *EmqxReconciler) initializedPluginList(instance appsv1beta4.Emqx) (*appsv1beta4.Condition, error) {
	plugins, err := r.createInitPluginList(instance)
	if err != nil {
		return nil, err
	}

	if err := r.CreateOrUpdateList(instance, r.Scheme, plugins); err != nil {
		if err != nil {
			r.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedCreateOrUpdate", err.Error())
			condition := appsv1beta4.NewCondition(
				appsv1beta4.ConditionPluginInitialized,
				corev1.ConditionFalse,
				"PluginInitializeFailed",
				err.Error(),
			)
			return condition, err
		}
	}
	condition := appsv1beta4.NewCondition(
		appsv1beta4.ConditionPluginInitialized,
		corev1.ConditionTrue,
		"PluginInitializeSuccessfully",
		"All default plugins initialized",
	)
	return condition, nil
}

func (r *EmqxReconciler) updateEmqxStatus(instance appsv1beta4.Emqx) (appsv1beta4.Status, error) {
	var condition *appsv1beta4.Condition

	status := instance.GetStatus()
	status.Replicas = *instance.GetReplicas()

	emqxNodes, err := r.getNodeStatusesByAPI(instance)
	if err != nil {
		r.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedToGetNodeStatues", err.Error())
		condition = appsv1beta4.NewCondition(
			appsv1beta4.ConditionRunning,
			corev1.ConditionFalse,
			"FailedToGetNodeStatues",
			err.Error(),
		)
		status.SetCondition(*condition)
		return status, err
	}

	if emqxNodes != nil {
		readyReplicas := int32(0)
		for _, node := range emqxNodes {
			if node.NodeStatus == "Running" {
				readyReplicas++
			}
		}
		status.ReadyReplicas = readyReplicas
		status.EmqxNodes = emqxNodes
	}

	if status.ReadyReplicas >= status.Replicas {
		condition = appsv1beta4.NewCondition(
			appsv1beta4.ConditionRunning,
			corev1.ConditionTrue,
			"ClusterReady",
			"All resources are ready",
		)
	} else {
		condition = appsv1beta4.NewCondition(
			appsv1beta4.ConditionRunning,
			corev1.ConditionFalse,
			"ClusterNotReady",
			"Some nodes are not ready",
		)
	}
	status.SetCondition(*condition)
	return status, nil
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
