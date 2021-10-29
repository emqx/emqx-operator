package controllers

import (
	"github.com/emqx/emqx-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Ensure the EMQ X Cluster's components are correct.
func (ech *EmqxClusterHandler) Ensure(e *v1alpha1.Emqx, labels map[string]string, or []metav1.OwnerReference) error {

	if err := ech.eService.EnsureEmqxSecret(e, labels, or); err != nil {
		return err
	}

	if err := ech.eService.EnsureEmqxHeadlessService(e, labels, or); err != nil {
		return err
	}

	if err := ech.eService.EnsureEmqxConfigMapForAcl(e, labels, or); err != nil {
		return err
	}

	if err := ech.eService.EnsureEmqxConfigMapForLoadedModules(e, labels, or); err != nil {
		return err
	}

	if err := ech.eService.EnsureEmqxConfigMapForLoadedPlugins(e, labels, or); err != nil {
		return err
	}

	if err := ech.eService.EnsureEmqxStatefulSet(e, labels, or); err != nil {
		return err
	}

	return nil
}
