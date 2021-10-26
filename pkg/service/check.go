package service

import (
	"errors"

	"github.com/emqx/emqx-operator/api/v1alpha1"
	"github.com/emqx/emqx-operator/pkg/client/k8s"
	"github.com/emqx/emqx-operator/pkg/util"
	"github.com/go-logr/logr"
)

// EmqxClusterCheck defines the intercace able to check the correct status of a emq x cluster
type EmqxClusterCheck interface {
	CheckEmqxReadyReplicas(emqx *v1alpha1.Emqx) error
}

// EmqxClusterChecker is our implementation of EmqxClusterCheck intercace
type EmqxClusterChecker struct {
	k8sService k8s.Services
	// client     emqx.Client
	logger logr.Logger
}

// CheckEmqxReadyReplicas controls that the number of deployed emqx ready pod is the same than the requested on the spec
func (ec *EmqxClusterChecker) CheckEmqxReadyReplicas(e *v1alpha1.Emqx) error {
	d, err := ec.k8sService.GetStatefulSet(e.Namespace, util.GetEmqxName(e))
	if err != nil {
		return err
	}
	if *e.Spec.Replicas != d.Status.ReadyReplicas {
		return errors.New("waiting all of emqx pods become ready")
	}
	return nil
}
