package service

import (
	"errors"

	"github.com/emqx/emqx-operator/api/v1alpha2"
	"github.com/emqx/emqx-operator/pkg/client/k8s"
	"github.com/go-logr/logr"
)

// EmqxClusterCheck defines the intercace able to check the correct status of a emq x cluster
type EmqxClusterCheck interface {
	CheckEmqxReadyReplicas(emqx v1alpha2.Emqx) error
}

// EmqxClusterChecker is our implementation of EmqxClusterCheck intercace
type EmqxClusterChecker struct {
	k8sService k8s.Services
	// TODO httpClient
	// EmqxClient broker.Client
	logger logr.Logger
}

// NewEmqxClusterChecker creates an object of the EmqxClusterChecker struct
// func NewEmqxClusterChecker(k8sService k8s.Services, EmqxClient broker.Client, logger logr.Logger) *EmqxClusterChecker {
func NewEmqxClusterChecker(k8sService k8s.Services, logger logr.Logger) *EmqxClusterChecker {
	return &EmqxClusterChecker{
		k8sService: k8sService,
		// TODO
		// EmqxClient: EmqxClient,
		logger: logger,
	}
}

// CheckEmqxReadyReplicas controls that the number of deployed emqx ready pod is the same than the requested on the spec
func (ec *EmqxClusterChecker) CheckEmqxReadyReplicas(emqx v1alpha2.Emqx) error {
	d, err := ec.k8sService.GetStatefulSet(emqx.GetNamespace(), emqx.GetName())
	if err != nil {
		return err
	}
	if *emqx.GetReplicas() != d.Status.ReadyReplicas {
		return errors.New("waiting all of emqx pods become ready")
	}
	return nil
}

// TODO
// GetEmqxClusterIPs return the IPS of brokers
// func (ec *EmqxClusterChecker) GetEmqxClusterIPs(e *v1alpha2.Emqx) ([]string, error) {
// 	ips := []string{}
// 	stsps, err := ec.k8sService.GetStatefulSetPods(e.Namespace, util.GetEmqxName(e))
// 	if err != nil {
// 		return nil, err
// 	}
// 	for _, stsp := range stsps.Items {
// 		if stsp.Status.Phase == corev1.PodRunning {
// 			ips = append(ips, stsp.Status.PodIP)
// 		}
// 	}
// 	return ips, nil
// }
