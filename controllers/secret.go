package controllers

import (
	"github.com/emqx/emqx-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeSecretConfigFromSpec(instance *v1alpha1.Emqx) *v1.Secret {
	config := instance.Spec.License
	configSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: EMQX_LIC_NAME,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: instance.APIVersion,
					Kind:       instance.Kind,
					Name:       instance.Name,
					UID:        instance.UID,
				},
			},
		},
		Type:       "Opaque",
		StringData: map[string]string{"emqx.lic": config},
	}
	configSecret.Namespace = instance.Namespace
	return configSecret
}
