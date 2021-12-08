package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type ServiceManagers interface {
	GetService(namespace string, name string) (*corev1.Service, error)
	CreateService(object *corev1.Service) error
	UpdateService(object *corev1.Service) error
	DeleteService(namespace, name string) error
}

func (manager *Manager) GetService(namespace string, name string) (*corev1.Service, error) {
	object := &corev1.Service{}
	err := manager.Client.Get(
		context.TODO(),
		types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, object,
	)

	if err != nil {
		return nil, err
	}
	return object, err
}

func (manager *Manager) CreateService(object *corev1.Service) error {
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

func (manager *Manager) UpdateService(object *corev1.Service) error {
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

func (manager *Manager) DeleteService(namespace string, name string) error {
	object := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return manager.Client.Delete(context.TODO(), object)
}
