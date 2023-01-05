package v1beta4

import (
	"context"

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addEmqxResources struct {
	*EmqxReconciler
	*requestAPI
}

func (a addEmqxResources) reconcile(ctx context.Context, instance appsv1beta4.Emqx, args ...any) subResult {
	initResources, ok := args[0].([]client.Object)
	if !ok {
		panic("args[0] is not []client.Object")
	}

	// ignore error, because if statefulSet is not created, the listener port will be not found
	listenerPorts, _ := a.getListenerPortsByAPI(instance)

	var resources []client.Object

	acl := generateEmqxACL(instance)
	resources = append(resources, acl)

	headlessSvc := generateHeadlessService(instance, listenerPorts...)
	resources = append(resources, headlessSvc)

	svc := generateService(instance, listenerPorts...)
	if svc != nil {
		resources = append(resources, svc)
	}

	license := a.getLicense(ctx, instance)
	if license != nil {
		resources = append(resources, license)
	}

	if err := a.CreateOrUpdateList(instance, a.Scheme, resources); err != nil {
		a.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedCreateOrUpdate", err.Error())
		return subResult{err: err}
	}

	sts := generateStatefulSet(instance)
	sts = updateStatefulSetForACL(sts, acl)
	sts = updateStatefulSetForLicense(sts, license)

	names := appsv1beta4.Names{Object: instance}
	for _, initResource := range initResources {
		if initResource.GetName() == names.BootstrapUser() {
			bootstrapUser := initResource.(*corev1.Secret)
			sts = updateStatefulSetForBootstrapUser(sts, bootstrapUser)
		}
		if initResource.GetName() == names.PluginsConfig() {
			pluginsConfig := initResource.(*corev1.ConfigMap)
			sts = updateStatefulSetForPluginsConfig(sts, pluginsConfig)
		}
	}

	return subResult{args: sts}
}

func (a addEmqxResources) getLicense(ctx context.Context, instance appsv1beta4.Emqx) *corev1.Secret {
	enterprise, ok := instance.(*appsv1beta4.EmqxEnterprise)
	if !ok {
		return nil
	}

	if enterprise.Spec.License.SecretName != "" {
		license := &corev1.Secret{}
		if err := a.Client.Get(
			ctx,
			types.NamespacedName{
				Name:      enterprise.Spec.License.SecretName,
				Namespace: enterprise.GetNamespace(),
			},
			license,
		); err != nil {
			a.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedGetLicense", err.Error())
			return nil
		}
		return license
	}
	return generateLicense(instance)
}
