package controllers

import (
	"Emqx/api/v1alpha1"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeModulesFromSpec(instance *v1alpha1.Broker) *v1.ConfigMap {
	config := createStringData(instance.Spec.Modules)
	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: emqxloadmodulesName,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: instance.APIVersion,
					Kind:       instance.Kind,
					Name:       instance.Name,
					UID:        instance.UID,
				},
			},
		},
		Data: map[string]string{"loaded-modules": config},
	}
	configMap.Namespace = instance.Namespace
	return configMap
}

func makeEnvFromSpec(instance *v1alpha1.Broker) *v1.ConfigMap {
	env := createStringData(instance.Spec.Env)
	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: emqxenvName,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: instance.APIVersion,
					Kind:       instance.Kind,
					Name:       instance.Name,
					UID:        instance.UID,
				},
			},
		},
		Data: map[string]string{"data": env},
	}
	configMap.Namespace = instance.Namespace
	return configMap
}
