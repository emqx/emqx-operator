package service

import (
	"errors"

	"github.com/emqx/emqx-operator/api/v1alpha2"
	"github.com/emqx/emqx-operator/pkg/client/broker"
	"github.com/emqx/emqx-operator/pkg/client/k8s"
	"github.com/emqx/emqx-operator/pkg/util"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

// EmqxBrokerClusterCheck defines the intercace able to check the correct status of a emq x cluster
type EmqxBrokerClusterCheck interface {
	CheckEmqxBrokerReadyReplicas(emqx *v1alpha2.EmqxBroker) error
}

// EmqxBrokerClusterChecker is our implementation of EmqxBrokerClusterCheck intercace
type EmqxBrokerClusterChecker struct {
	k8sService       k8s.Services
	emqxBrokerClient broker.Client
	logger           logr.Logger
}

// NewEmqxBrokerClusterChecker creates an object of the emqxBrokerClusterChecker struct
func NewEmqxBrokerClusterChecker(k8sService k8s.Services, emqxBrokerClient broker.Client, logger logr.Logger) *EmqxBrokerClusterChecker {
	return &EmqxBrokerClusterChecker{
		k8sService:       k8sService,
		emqxBrokerClient: emqxBrokerClient,
		logger:           logger,
	}
}

// CheckEmqxBrokerReadyReplicas controls that the number of deployed emqx ready pod is the same than the requested on the spec
func (ec *EmqxBrokerClusterChecker) CheckEmqxBrokerReadyReplicas(e *v1alpha2.EmqxBroker) error {
	d, err := ec.k8sService.GetStatefulSet(e.Namespace, util.GetEmqxBrokerName(e))
	if err != nil {
		return err
	}
	if *e.Spec.Replicas != d.Status.ReadyReplicas {
		return errors.New("waiting all of emqx broker pods become ready")
	}
	return nil
}

// GetEmqxBrokerClusterIPs return the IPS of brokers
func (ec *EmqxBrokerClusterChecker) GetEmqxBrokerClusterIPs(e *v1alpha2.EmqxBroker) ([]string, error) {
	ips := []string{}
	stsps, err := ec.k8sService.GetStatefulSetPods(e.Namespace, util.GetEmqxBrokerName(e))
	if err != nil {
		return nil, err
	}
	for _, stsp := range stsps.Items {
		if stsp.Status.Phase == corev1.PodRunning {
			ips = append(ips, stsp.Status.PodIP)
		}
	}
	return ips, nil
}
