package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type NamespacesManagers interface {
	GetNamespace(namespace string) (*corev1.Namespace, error)
	CreateNamespace(object *corev1.Namespace) error
	UpdateNamespace(object *corev1.Namespace) error
	DeleteNamespace(namespace string) error
}

func (manager *Manager) GetNamespace(namespace string) (*corev1.Namespace, error) {
	object := &corev1.Namespace{}
	err := manager.Client.Get(
		context.TODO(),
		types.NamespacedName{
			Namespace: namespace,
		}, object,
	)
	if err != nil {
		return nil, err
	}
	return object, err
}

func (manager *Manager) CreateNamespace(object *corev1.Namespace) error {
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

func (manager *Manager) UpdateNamespace(object *corev1.Namespace) error {
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

func (manager *Manager) DeleteNamespace(namespace string) error {
	object := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
	}
	return manager.Client.Delete(context.TODO(), object)
}
