package controllers

import (
	"github.com/emqx/emqx-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Ensure the EMQ X Cluster's components are correct.
func (ech *EmqxBrokerClusterHandler) Ensure(e *v1alpha1.EmqxBroker, labels map[string]string, or []metav1.OwnerReference) error {

	if err := ech.eService.EnsureEmqxBrokerSecret(e, labels, or); err != nil {
		return err
	}

	if err := ech.eService.EnsureEmqxBrokerHeadlessService(e, labels, or); err != nil {
		return err
	}

	if err := ech.eService.EnsureEmqxBrokerConfigMapForAcl(e, labels, or); err != nil {
		return err
	}

	if err := ech.eService.EnsureEmqxBrokerConfigMapForLoadedModules(e, labels, or); err != nil {
		return err
	}

	if err := ech.eService.EnsureEmqxBrokerConfigMapForLoadedPlugins(e, labels, or); err != nil {
		return err
	}

	if err := ech.eService.EnsureEmqxBrokerStatefulSet(e, labels, or); err != nil {
		return err
	}

	return nil
}
