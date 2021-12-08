package k8s

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type RoleBindingManagers interface {
	GetRoleBinding(namespace, name string) (*rbacv1.RoleBinding, error)
	CreateRoleBinding(object *rbacv1.RoleBinding) error
	UpdateRoleBinding(object *rbacv1.RoleBinding) error
	DeleteRoleBinding(namespace, name string) error
}

func (manager *Manager) GetRoleBinding(namespace, name string) (*rbacv1.RoleBinding, error) {
	object := &rbacv1.RoleBinding{}
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

func (manager *Manager) CreateRoleBinding(object *rbacv1.RoleBinding) error {
	err := manager.Client.Create(context.TODO(), object)
	if err != nil {
		return err
	}
	manager.Logger.WithValues(
		"kind", object.Kind,
		"apiVersion", object.APIVersion,
		"namespace", object.Namespace,
		"name", object.Name,
	).Info("Create roleBinding successfully")
	return nil
}

func (manager *Manager) UpdateRoleBinding(object *rbacv1.RoleBinding) error {
	if err := manager.Client.Update(context.TODO(), object); err != nil {
		return err
	}
	manager.Logger.WithValues(
		"kind", object.Kind,
		"apiVersion", object.APIVersion,
		"namespace", object.Namespace,
		"serviceName", object.Name,
	).Info("Update roleBinding successfully")
	return nil
}

func (manager *Manager) DeleteRoleBinding(namespace, name string) error {
	object := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return manager.Client.Delete(context.TODO(), object)
}
