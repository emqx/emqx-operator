package controller

import (
	"crypto/rand"

	emperror "emperror.dev/errors"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	resources "github.com/emqx/emqx-operator/internal/controller/resources"
)

type addBootstrap struct {
	*EMQXReconciler
}

func (a *addBootstrap) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	for _, resource := range []client.Object{
		generateNodeCookieSecret(instance),
		generateBootstrapAPIKeySecret(instance),
	} {
		if err := ctrl.SetControllerReference(instance, resource, a.Scheme); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to set controller reference")}
		}
		if err := a.Create(r.ctx, resource); err != nil {
			if !k8sErrors.IsAlreadyExists(err) {
				return subResult{err: emperror.Wrap(err, "failed to create bootstrap secret")}
			}
		}
	}

	return subResult{}
}

func generateBootstrapAPIKeySecret(instance *crd.EMQX) *corev1.Secret {
	content := resources.DefaultBootstrapAPIKey + ":" + rand.Text()
	return resources.BootstrapAPIKey(instance).Secret(content)
}

func generateNodeCookieSecret(instance *crd.EMQX) *corev1.Secret {
	return resources.Cookie(instance).Secret(rand.Text())
}
