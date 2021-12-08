package controllers

import (
	"github.com/emqx/emqx-operator/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Ensure the EMQ X Cluster's components are correct.
func (handler *EmqxClusterHandler) Ensure(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	if err := handler.eService.EnsureEmqxSecret(emqx, labels, ownerRefs); err != nil {
		return err
	}

	if err := handler.eService.EnsureEmqxHeadlessService(emqx, labels, ownerRefs); err != nil {
		return err
	}

	if err := handler.eService.EnsureEmqxConfigMapForAcl(emqx, labels, ownerRefs); err != nil {
		return err
	}

	if err := handler.eService.EnsureEmqxConfigMapForLoadedModules(emqx, labels, ownerRefs); err != nil {
		return err
	}

	if err := handler.eService.EnsureEmqxConfigMapForLoadedPlugins(emqx, labels, ownerRefs); err != nil {
		return err
	}

	if err := handler.eService.EnsureEmqxStatefulSet(emqx, labels, ownerRefs); err != nil {
		return err
	}

	if err := handler.eService.EnsureEmqxListenerService(emqx, labels, ownerRefs); err != nil {
		return err
	}
	return nil
}
