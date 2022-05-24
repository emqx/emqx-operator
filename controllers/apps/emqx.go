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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1beta3 "github.com/emqx/emqx-operator/apis/apps/v1beta3"
	"github.com/emqx/emqx-operator/pkg/service"
)

var _ reconcile.Reconciler = &EmqxBrokerReconciler{}

type EmqxReconciler struct {
	Handler
}

func (r *EmqxReconciler) Do(ctx context.Context, instance appsv1beta3.Emqx) error {
	resourceList, sts := service.Generate(instance)
	// First reconcile
	if len(instance.GetStatus().Conditions) == 0 {
		pluginResourceList := generateDefaultPluginList(instance)
		resourceList = append(resourceList, pluginResourceList...)
	}

	ownerRef := metav1.NewControllerRef(instance, instance.GetObjectKind().GroupVersionKind())

	// StateFulSet should be created last
	for _, resource := range append(resourceList, sts) {
		addOwnerRefToObject(resource, *ownerRef)

		var err error
		names := appsv1beta3.Names{Object: instance}
		switch resource.GetName() {
		case names.PluginsConfig(), names.LoadedPlugins():
			// Only create plugins config and loaded plugins, do not update
			configMap := &corev1.ConfigMap{}
			err = r.Get(context.TODO(), client.ObjectKeyFromObject(resource), configMap)
			if k8sErrors.IsNotFound(err) {
				nothing := func(client.Object) error { return nil }
				err = r.Handler.doCreate(resource, nothing)
			} else {
				err = nil
			}
		case names.MQTTSCertificate():
			postFun := func(instance client.Object) error {
				return r.Handler.ExecToPods(instance, "emqx", "emqx_ctl listeners restart mqtt:wss:external")
			}
			err = r.Handler.CreateOrUpdate(resource, postFun)
		case names.WSSCertificate():
			postFun := func(instance client.Object) error {
				return r.Handler.ExecToPods(instance, "emqx", "emqx_ctl listeners restart mqtt:wss:external")
			}
			err = r.Handler.CreateOrUpdate(resource, postFun)

		default:
			nothing := func(client.Object) error { return nil }
			err = r.Handler.CreateOrUpdate(resource, nothing)
		}

		if err != nil {
			r.EventRecorder.Event(instance, corev1.EventTypeWarning, "Reconciled", err.Error())
			instance.SetFailedCondition(err.Error())
			instance.DescConditionsByTime()
			_ = r.Status().Update(ctx, instance)
			return err
		}
	}

	instance.SetRunningCondition("Reconciled")
	instance.DescConditionsByTime()
	_ = r.Status().Update(ctx, instance)
	return nil
}

func generateDefaultPluginList(instance appsv1beta3.Emqx) []client.Object {
	pluginList := []client.Object{}

	// Default plugins
	emqxManagement := &appsv1beta3.EmqxPlugin{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps.emqx.io/v1beta3",
			Kind:       "EmqxPlugin",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-management", instance.GetName()),
			Namespace: instance.GetNamespace(),
			Labels:    instance.GetLabels(),
		},
		Spec: appsv1beta3.EmqxPluginSpec{
			PluginName: "emqx_management",
			Selector:   instance.GetLabels(),
			Config: map[string]string{
				"management.listener.http":              "8081",
				"management.default_application.id":     "admin",
				"management.default_application.secret": "public",
			},
		},
	}

	emqxDashboard := &appsv1beta3.EmqxPlugin{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps.emqx.io/v1beta3",
			Kind:       "EmqxPlugin",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-dashboard", instance.GetName()),
			Namespace: instance.GetNamespace(),
			Labels:    instance.GetLabels(),
		},
		Spec: appsv1beta3.EmqxPluginSpec{
			PluginName: "emqx_dashboard",
			Selector:   instance.GetLabels(),
			Config: map[string]string{
				"dashboard.listener.http":         "18083",
				"dashboard.default_user.login":    "admin",
				"dashboard.default_user.password": "public",
			},
		},
	}

	emqxRuleEngine := &appsv1beta3.EmqxPlugin{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps.emqx.io/v1beta3",
			Kind:       "EmqxPlugin",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-rule-engine", instance.GetName()),
			Namespace: instance.GetNamespace(),
			Labels:    instance.GetLabels(),
		},
		Spec: appsv1beta3.EmqxPluginSpec{
			PluginName: "emqx_rule_engine",
			Selector:   instance.GetLabels(),
			Config:     map[string]string{},
		},
	}

	emqxRetainer := &appsv1beta3.EmqxPlugin{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps.emqx.io/v1beta3",
			Kind:       "EmqxPlugin",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-retainer", instance.GetName()),
			Namespace: instance.GetNamespace(),
			Labels:    instance.GetLabels(),
		},
		Spec: appsv1beta3.EmqxPluginSpec{
			PluginName: "emqx_retainer",
			Selector:   instance.GetLabels(),
			Config:     map[string]string{},
		},
	}

	pluginList = append(pluginList, emqxManagement, emqxDashboard, emqxRuleEngine, emqxRetainer)

	return pluginList
}

func addOwnerRefToObject(obj metav1.Object, ownerRef metav1.OwnerReference) {
	obj.SetOwnerReferences(append(obj.GetOwnerReferences(), ownerRef))
}
