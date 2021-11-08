package service

import (
	"errors"

	"github.com/emqx/emqx-operator/api/v1alpha2"
	"github.com/emqx/emqx-operator/pkg/client/k8s"
	"github.com/emqx/emqx-operator/pkg/util"
	"github.com/go-logr/logr"
)

// EmqxBrokerClusterCheck defines the intercace able to check the correct status of a emq x cluster
type EmqxBrokerClusterCheck interface {
	CheckEmqxBrokerReadyReplicas(emqx *v1alpha2.EmqxBroker) error
}

// EmqxBrokerClusterChecker is our implementation of EmqxBrokerClusterCheck intercace
type EmqxBrokerClusterChecker struct {
	k8sService k8s.Services
	// client     emqx.Client
	logger logr.Logger
}

// CheckEmqxBrokerReadyReplicas controls that the number of deployed emqx ready pod is the same than the requested on the spec
func (ec *EmqxBrokerClusterChecker) CheckEmqxBrokerReadyReplicas(e *v1alpha2.EmqxBroker) error {
	d, err := ec.k8sService.GetStatefulSet(e.Namespace, util.GetEmqxBrokerName(e))
	if err != nil {
		return err
	}
	if *e.Spec.Replicas != d.Status.ReadyReplicas {
		return errors.New("waiting all of emqx pods become ready")
	}
	return nil
}
