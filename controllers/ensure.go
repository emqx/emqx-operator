package controllers

import (
	"github.com/emqx/emqx-operator/api/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Ensure the EMQ X Cluster's components are correct.
func (ech *EmqxBrokerClusterHandler) Ensure(emqx v1alpha2.Emqx, labels map[string]string, or []metav1.OwnerReference) error {

	if err := ech.eService.EnsureEmqxBrokerSecret(emqx, labels, or); err != nil {
		return err
	}

	if err := ech.eService.EnsureEmqxBrokerHeadlessService(emqx, labels, or); err != nil {
		return err
	}

	if err := ech.eService.EnsureEmqxBrokerConfigMapForAcl(emqx, labels, or); err != nil {
		return err
	}

	if err := ech.eService.EnsureEmqxBrokerConfigMapForLoadedModules(emqx, labels, or); err != nil {
		return err
	}

	if err := ech.eService.EnsureEmqxBrokerConfigMapForLoadedPlugins(emqx, labels, or); err != nil {
		return err
	}

	if err := ech.eService.EnsureEmqxBrokerStatefulSet(emqx, labels, or); err != nil {
		return err
	}

	return nil
}
