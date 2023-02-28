package v2alpha1

import (
	"context"

	emperror "emperror.dev/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
)

type addBootstrap struct {
	*EMQXReconciler
}

func (a *addBootstrap) reconcile(ctx context.Context, instance *appsv2alpha1.EMQX) subResult {
	nodeCookie := generateNodeCookieSecret(instance)
	if err := ctrl.SetControllerReference(instance, nodeCookie, a.Scheme); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to set controller reference")}
	}
	if err := a.Create(nodeCookie); err != nil {
		if !k8sErrors.IsAlreadyExists(err) {
			return subResult{err: emperror.Wrap(err, "failed to create node cookie secret")}
		}
	}

	bootstrapUser := generateBootstrapUserSecret(instance)
	if err := ctrl.SetControllerReference(instance, bootstrapUser, a.Scheme); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to set controller reference")}
	}
	if err := a.Create(bootstrapUser); err != nil {
		if !k8sErrors.IsAlreadyExists(err) {
			return subResult{err: emperror.Wrap(err, "failed to create bootstrap user secret")}
		}
	}

	bootstrapConfig := generateBootstrapConfigMap(instance)
	if err := ctrl.SetControllerReference(instance, bootstrapConfig, a.Scheme); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to set controller reference")}
	}
	if err := a.Create(bootstrapConfig); err != nil {
		if !k8sErrors.IsAlreadyExists(err) {
			return subResult{err: emperror.Wrap(err, "failed to create bootstrap configMap")}
		}
	}

	return subResult{}
}
