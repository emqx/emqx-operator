package v2alpha1

import (
	"context"

	emperror "emperror.dev/errors"
	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addRepl struct {
	*EMQXReconciler
}

func (a *addRepl) reconcile(ctx context.Context, instance *appsv2alpha1.EMQX) subResult {
	if instance.Status.IsRunning() || instance.Status.IsCoreNodesReady() {
		nodeCookie := &corev1.Secret{}
		if err := a.Client.Get(ctx, instance.NodeCookieNamespacedName(), nodeCookie); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to get node cookie secret")}
		}
		bootstrapConf := &corev1.ConfigMap{}
		if err := a.Client.Get(ctx, instance.BootstrapConfigNamespacedName(), bootstrapConf); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to get bootstrap configMap")}
		}

		deploy := generateDeployment(instance)
		deploy = updateDeploymentForNodeCookie(deploy, nodeCookie)
		deploy = updateDeploymentForBootstrapConfig(deploy, bootstrapConf)

		if err := a.CreateOrUpdateList(instance, a.Scheme, []client.Object{deploy}); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to create or update deployment")}
		}
	}

	return subResult{}
}
