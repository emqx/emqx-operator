package controller

import (
	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v3beta1"
	config "github.com/emqx/emqx-operator/internal/controller/config"
)

type loadConfig struct {
	*EMQXReconciler
}

func (l *loadConfig) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	conf, err := config.EMQXConfigWithDefaults(applicableConfig(instance))
	if err != nil {
		return subResult{
			err: emperror.Wrap(err, "the .spec.config.data is not a valid HOCON config"),
		}
	}
	r.conf = conf
	return subResult{}
}
