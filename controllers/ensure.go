package controllers

import (
	"github.com/emqx/emqx-operator/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Ensure the EMQ X Cluster's components are correct.
func (ech *EmqxClusterHandler) Ensure(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	if err := ech.eService.EnsureEmqxSecret(emqx, labels, ownerRefs); err != nil {
		return err
	}

	if err := ech.eService.EnsureEmqxHeadlessService(emqx, labels, ownerRefs); err != nil {
		return err
	}

	if err := ech.eService.EnsureEmqxConfigMapForAcl(emqx, labels, ownerRefs); err != nil {
		return err
	}

	if err := ech.eService.EnsureEmqxConfigMapForLoadedModules(emqx, labels, ownerRefs); err != nil {
		return err
	}

	if err := ech.eService.EnsureEmqxConfigMapForLoadedPlugins(emqx, labels, ownerRefs); err != nil {
		return err
	}

	if err := ech.eService.EnsureEmqxStatefulSet(emqx, labels, ownerRefs); err != nil {
		return err
	}

	if err := ech.eService.EnsureEmqxListenerService(emqx, labels, ownerRefs); err != nil {
		return err
	}
	return nil
}
