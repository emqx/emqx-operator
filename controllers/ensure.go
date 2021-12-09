package controllers

import (
	"github.com/emqx/emqx-operator/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (handler *Handler) Ensure(emqx v1beta1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	if err := handler.client.EnsureEmqxSecret(emqx, labels, ownerRefs); err != nil {
		return err
	}

	if err := handler.client.EnsureEmqxRBAC(emqx, labels, ownerRefs); err != nil {
		return err
	}

	if err := handler.client.EnsureEmqxHeadlessService(emqx, labels, ownerRefs); err != nil {
		return err
	}

	if err := handler.client.EnsureEmqxConfigMapForAcl(emqx, labels, ownerRefs); err != nil {
		return err
	}

	if err := handler.client.EnsureEmqxConfigMapForLoadedModules(emqx, labels, ownerRefs); err != nil {
		return err
	}

	if err := handler.client.EnsureEmqxConfigMapForLoadedPlugins(emqx, labels, ownerRefs); err != nil {
		return err
	}

	if err := handler.client.EnsureEmqxStatefulSet(emqx, labels, ownerRefs); err != nil {
		return err
	}

	if err := handler.client.EnsureEmqxListenerService(emqx, labels, ownerRefs); err != nil {
		return err
	}
	return nil
}
