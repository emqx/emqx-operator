package k8s

import (
	"context"

	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RoleBindingManagers interface {
	List(namespace string) (*rbacv1.RoleBindingList, error)
	Get(namespace, name string) (*rbacv1.RoleBinding, error)
	Create(object *rbacv1.RoleBinding) error
	Update(object *rbacv1.RoleBinding) error
	Delete(namespace, name string) error
}

type RoleBindingManager struct {
	Client client.Client
	Logger logr.Logger
}

func NewRoleBindingManager(client client.Client, logger logr.Logger) *RoleBindingManager {
	return &RoleBindingManager{
		Client: client,
		Logger: logger,
	}
}

func (manager *RoleBindingManager) List(namespace string) (*rbacv1.RoleBindingList, error) {
	list := &rbacv1.RoleBindingList{}
	err := manager.Client.List(
		context.TODO(),
		list,
		&client.ListOptions{
			Namespace: namespace,
		},
	)
	return list, err
}

func (manager *RoleBindingManager) Get(namespace, name string) (*rbacv1.RoleBinding, error) {
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

func (manager *RoleBindingManager) Create(object *rbacv1.RoleBinding) error {
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

func (manager *RoleBindingManager) Update(object *rbacv1.RoleBinding) error {
	if err := manager.Client.Update(context.TODO(), object); err != nil {
		return err
	}
	manager.Logger.WithValues(
		"kind", object.Kind,
		"apiVersion", object.APIVersion,
		"namespace", object.Namespace,
		"name", object.Name,
	).Info("Update roleBinding successfully")
	return nil
}

func (manager *RoleBindingManager) Delete(namespace, name string) error {
	object := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return manager.Client.Delete(context.TODO(), object)
}
