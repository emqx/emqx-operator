package controller

import (
	"context"

	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v3alpha1"
	corev1 "k8s.io/api/core/v1"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

type setupAPIRequester struct {
	*EMQXReconciler
}

func (s *setupAPIRequester) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	bootstrapAPIKey, err := getBootstrapAPIKey(r.ctx, s.Client, instance)
	if err != nil {
		return subResult{err: err}
	}
	reqBuilder, err := newAPIRequesterBuilder(r.conf, bootstrapAPIKey)
	if err != nil {
		return subResult{err: err}
	}
	r.requester = reqBuilder
	return subResult{}
}

func getBootstrapAPIKey(
	ctx context.Context,
	client k8s.Client,
	instance *crd.EMQX,
) (*corev1.Secret, error) {
	bootstrapAPIKey := &corev1.Secret{}
	err := client.Get(ctx, instance.BootstrapAPIKeyNamespacedName(), bootstrapAPIKey)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get secret")
	}
	return bootstrapAPIKey, nil
}
