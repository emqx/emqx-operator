package v1beta4

import (
	"context"

	emperror "emperror.dev/errors"
	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addEmqxInitResources struct{}

func (a addEmqxInitResources) reconcile(ctx context.Context, r *EmqxReconciler, instance appsv1beta4.Emqx, _ ...any) subResult {
	resources, err := a.getInitResources(ctx, r, instance)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get init resources")}
	}

	for _, resource := range resources {
		if err := r.Client.Get(ctx, client.ObjectKeyFromObject(resource), resource); err != nil {
			if k8sErrors.IsNotFound(err) {
				if err := ctrl.SetControllerReference(instance, resource, r.Scheme); err != nil {
					r.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedSetControllerReference", err.Error())
					return subResult{err: emperror.Wrap(err, "failed to set controller reference")}
				}
				if err := r.Handler.Create(resource); err != nil {
					r.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedCreate", err.Error())
					return subResult{err: emperror.Wrap(err, "failed to create resource")}
				}
			}
			return subResult{err: emperror.Wrap(err, "failed to get resource")}
		}
	}

	return subResult{args: resources}
}

func (a addEmqxInitResources) getInitResources(ctx context.Context, r *EmqxReconciler, instance appsv1beta4.Emqx) ([]client.Object, error) {
	var resources []client.Object

	bootstrap_user := generateBootstrapUserSecret(instance)
	resources = append(resources, bootstrap_user)

	defaultPluginsConfig := generateDefaultPluginsConfig(instance)
	resources = append(resources, defaultPluginsConfig)

	plugins, err := a.getInitPluginList(ctx, r, instance)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get init plugin list")
	}
	resources = append(resources, plugins...)

	return resources, nil
}

func (a addEmqxInitResources) getInitPluginList(ctx context.Context, r *EmqxReconciler, instance appsv1beta4.Emqx) ([]client.Object, error) {
	pluginsList := &appsv1beta4.EmqxPluginList{}
	err := r.Client.List(ctx, pluginsList, client.InNamespace(instance.GetNamespace()))
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}
	initPluginsList := generateInitPluginList(instance, pluginsList)
	return initPluginsList, nil
}
