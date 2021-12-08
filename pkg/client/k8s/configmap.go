package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type ConfigMapManagers interface {
	GetConfigMap(namespace, name string) (*corev1.ConfigMap, error)
	CreateConfigMap(configMap *corev1.ConfigMap) error
	UpdateConfigMap(configMap *corev1.ConfigMap) error
	DeleteConfigMap(namespace, name string) error
}

func (manager *Manager) GetConfigMap(namespace string, name string) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	err := manager.Client.Get(
		context.TODO(),
		types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, configMap,
	)

	if err != nil {
		return nil, err
	}
	return configMap, err
}

func (manager *Manager) CreateConfigMap(object *corev1.ConfigMap) error {
	err := manager.Client.Create(context.TODO(), object)
	if err != nil {
		return err
	}
	manager.Logger.WithValues(
		"kind", object.Kind,
		"apiVersion", object.APIVersion,
		"namespace", object.Namespace,
		"name", object.Name,
	).Info("Create successfully")
	return nil
}

func (manager *Manager) UpdateConfigMap(object *corev1.ConfigMap) error {
	if err := manager.Client.Update(context.TODO(), object); err != nil {
		return err
	}
	manager.Logger.WithValues(
		"kind", object.Kind,
		"apiVersion", object.APIVersion,
		"namespace", object.Namespace,
		"name", object.Name,
	).Info("Update successfully")
	return nil
}

func (manager *Manager) DeleteConfigMap(namespace, name string) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return manager.Client.Delete(context.TODO(), configMap)
}
