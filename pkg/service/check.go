package service

import (
	"errors"

	"github.com/emqx/emqx-operator/api/v1beta1"
	"github.com/emqx/emqx-operator/pkg/client/k8s"
)

type EmqxChecker interface {
	CheckReadyReplicas(emqx v1beta1.Emqx) error
}

type Checker struct {
	k8s.Manager
}

func NewChecker(manager k8s.Manager) *Checker {
	return &Checker{manager}
}

func (checker *Checker) CheckReadyReplicas(emqx v1beta1.Emqx) error {
	sts, err := checker.StatefulSet.Get(emqx.GetNamespace(), emqx.GetName())
	if err != nil {
		return err
	}
	if *emqx.GetReplicas() != sts.Status.ReadyReplicas {
		return errors.New("waiting all of emqx pods become ready")
	}
	return nil
}
