package v2alpha1

import (
	"context"

	emperror "emperror.dev/errors"
	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addCore struct {
	*EMQXReconciler
}

func (a *addCore) reconcile(ctx context.Context, instance *appsv2alpha1.EMQX) subResult {
	nodeCookie := &corev1.Secret{}
	if err := a.Client.Get(ctx, instance.NodeCookieNamespacedName(), nodeCookie); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get node cookie secret")}
	}
	bootstrapUser := &corev1.Secret{}
	if err := a.Client.Get(ctx, instance.BootstrapUserNamespacedName(), bootstrapUser); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get bootstrap user secret")}
	}
	bootstrapConf := &corev1.ConfigMap{}
	if err := a.Client.Get(ctx, instance.BootstrapConfigNamespacedName(), bootstrapConf); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get bootstrap configMap")}
	}

	sts := generateStatefulSet(instance)
	sts = updateStatefulSetForNodeCookie(sts, nodeCookie)
	sts = updateStatefulSetForBootstrapUser(sts, bootstrapUser)
	sts = updateStatefulSetForBootstrapConf(sts, bootstrapConf)

	if err := a.CreateOrUpdateList(instance, a.Scheme, []client.Object{sts}); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to create or update statefulSet")}
	}
	return subResult{}
}
