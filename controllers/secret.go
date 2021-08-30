package controllers

import (
	"github.com/emqx/emqx-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeSecret(instance *v1alpha1.Emqx) *v1.Secret {
	configSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: EMQX_LIC_NAME,
		},
		Type:       "Opaque",
		StringData: map[string]string{"emqx.lic": instance.Spec.License},
	}
	configSecret.Namespace = instance.Namespace
	return configSecret
}
