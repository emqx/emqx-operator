package v1beta4

import (
	"context"

	emperror "emperror.dev/errors"
	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addEmqxResources struct {
	*EmqxReconciler
	PortForwardAPI
}

func (a addEmqxResources) reconcile(ctx context.Context, instance appsv1beta4.Emqx, args ...any) subResult {
	initResources, ok := args[0].([]client.Object)
	if !ok {
		panic("args[0] is not []client.Object")
	}

	var resources []client.Object

	license, err := a.getLicense(ctx, instance)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get license")}
	}
	if license != nil {
		resources = append(resources, license)
	}

	acl := generateEmqxACL(instance)
	resources = append(resources, acl)

	headlessSvc := generateHeadlessService(instance)
	resources = append(resources, headlessSvc)

	if err := a.CreateOrUpdateList(instance, a.Scheme, resources); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to create or update resource")}
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

func (a addEmqxResources) getLicense(ctx context.Context, instance appsv1beta4.Emqx) (*corev1.Secret, error) {
	enterprise, ok := instance.(*appsv1beta4.EmqxEnterprise)
	if !ok {
		return nil, nil
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
			return nil, err
		}
		return license, nil
	}
	return generateLicense(instance), nil
}
