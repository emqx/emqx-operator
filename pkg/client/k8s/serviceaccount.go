package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type ServiceAccountManagers interface {
	GetServiceAccount(namespace, name string) (*corev1.ServiceAccount, error)
	CreateServiceAccount(object *corev1.ServiceAccount) error
	UpdateServiceAccount(object *corev1.ServiceAccount) error
	DeleteServiceAccount(namespace, name string) error
}

func (manager *Manager) GetServiceAccount(namespace, name string) (*corev1.ServiceAccount, error) {
	object := &corev1.ServiceAccount{}
	err := manager.Client.Get(
		context.TODO(),
		types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
		object,
	)

	if err != nil {
		return nil, err
	}
	return object, err
}

func (manager *Manager) CreateServiceAccount(object *corev1.ServiceAccount) error {
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

func (manager *Manager) UpdateServiceAccount(object *corev1.ServiceAccount) error {
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

func (manager *Manager) DeleteServiceAccount(namespace, name string) error {
	object := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return manager.Client.Delete(context.TODO(), object)
}
