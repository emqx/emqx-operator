package controllers

import (
	"github.com/emqx/emqx-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeSecretOwnerReference(instance *v1alpha1.Emqx) *v1.Secret {
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
		Type: "Opaque",
	}
	configSecret.Namespace = instance.Namespace
	return configSecret
}

func makeSecretSpec(instance *v1alpha1.Emqx) map[string]string {
	license := instance.Spec.License
	stringData := map[string]string{"emqx.lic": license}
	return stringData
}
