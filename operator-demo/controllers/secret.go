package controllers

import (
	"Emqx/api/v1alpha1"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeSecretConfigFromSpec(instance *v1alpha1.Broker) *v1.Secret {
	config := instance.Spec.Secret
	configSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: emqxlicName,
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
