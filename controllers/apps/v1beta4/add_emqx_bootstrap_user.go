package v1beta4

import (
	"context"

	emperror "emperror.dev/errors"
	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addEmqxBootstrapUser struct {
	*EmqxReconciler
}

func (a addEmqxBootstrapUser) reconcile(ctx context.Context, instance appsv1beta4.Emqx, _ ...any) subResult {
	bootstrapUser := generateBootstrapUserSecret(instance)

	if err := a.Client.Get(ctx, client.ObjectKeyFromObject(bootstrapUser), bootstrapUser); err != nil {
		if k8sErrors.IsNotFound(err) {
			if err := ctrl.SetControllerReference(instance, bootstrapUser, a.Scheme); err != nil {
				return subResult{err: emperror.Wrap(err, "failed to set controller reference")}
			}
			if err := a.Handler.Create(bootstrapUser); err != nil {
				return subResult{err: emperror.Wrap(err, "failed to create resource")}
			}
		}
		return subResult{err: emperror.Wrap(err, "failed to get resource")}
	}

	return subResult{args: bootstrapUser}
}
