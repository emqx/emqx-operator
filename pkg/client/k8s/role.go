package k8s

import (
	"context"

	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RoleManagers interface {
	Get(namespace, name string) (*rbacv1.Role, error)
	Create(object *rbacv1.Role) error
	Update(object *rbacv1.Role) error
	Delete(namespace, name string) error
}

type RoleManager struct {
	Client client.Client
	Logger logr.Logger
}

func NewRoleManager(client client.Client, logger logr.Logger) *RoleManager {
	return &RoleManager{
		Client: client,
		Logger: logger,
	}
}

func (manager *RoleManager) Get(namespace, name string) (*rbacv1.Role, error) {
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

func (manager *RoleManager) Create(object *rbacv1.Role) error {
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

func (manager *RoleManager) Update(object *rbacv1.Role) error {
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

func (manager *RoleManager) Delete(namespace, name string) error {
	object := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return manager.Client.Delete(context.TODO(), object)
}
