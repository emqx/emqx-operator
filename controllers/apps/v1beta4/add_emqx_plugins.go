package v1beta4

import (
	"context"

	emperror "emperror.dev/errors"
	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	"github.com/go-logr/logr"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addEmqxPlugins struct {
	*EmqxReconciler
}

func (a addEmqxPlugins) reconcile(ctx context.Context, logger logr.Logger, instance appsv1beta4.Emqx, args ...any) subResult {
	other, ok := args[0].(client.Object)
	if !ok {
		panic("args[0] is not client.Object")
	}

	var resources []client.Object

	defaultPluginsConfig := generateDefaultPluginsConfig(instance)
	resources = append(resources, defaultPluginsConfig)

	plugins, err := a.getInitPluginList(ctx, instance)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get init plugin list")}
	}
	resources = append(resources, plugins...)
	for _, resource := range resources {
		if err := a.Client.Get(ctx, client.ObjectKeyFromObject(resource), resource); err != nil {
			if k8sErrors.IsNotFound(err) {
				if err := ctrl.SetControllerReference(instance, resource, a.Scheme); err != nil {
					return subResult{err: emperror.Wrap(err, "failed to set controller reference")}
				}
				if err := a.Handler.Create(ctx, resource); err != nil {
					return subResult{err: emperror.Wrap(err, "failed to create resource")}
				}
			}
			return subResult{err: emperror.Wrap(err, "failed to get resource")}
		}
	}

	return subResult{args: append(resources, other)}
}

func (a addEmqxPlugins) getInitPluginList(ctx context.Context, instance appsv1beta4.Emqx) ([]client.Object, error) {
	pluginsList := &appsv1beta4.EmqxPluginList{}
	err := a.Client.List(ctx, pluginsList, client.InNamespace(instance.GetNamespace()))
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}
	initPluginsList := generateInitPluginList(instance, pluginsList)
	return initPluginsList, nil
}
