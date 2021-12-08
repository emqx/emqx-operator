package k8s

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type RoleManagers interface {
	GetRole(namespace, name string) (*rbacv1.Role, error)
	CreateRole(object *rbacv1.Role) error
	UpdateRole(object *rbacv1.Role) error
	DeleteRole(namespace, name string) error
}

func (manager *Manager) GetRole(namespace, name string) (*rbacv1.Role, error) {
	object := &rbacv1.Role{}
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

func (manager *Manager) CreateRole(object *rbacv1.Role) error {
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

func (manager *Manager) UpdateRole(object *rbacv1.Role) error {
	if err := manager.Client.Update(context.TODO(), object); err != nil {
		return err
	}
	manager.Logger.WithValues(
		"kind", object.Kind,
		"apiVersion", object.APIVersion,
		"namespace", object.Namespace,
		"serviceName", object.Name,
	).Info("Update successfully")
	return nil
}

func (manager *Manager) DeleteRole(namespace, name string) error {
	object := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return manager.Client.Delete(context.TODO(), object)
}
