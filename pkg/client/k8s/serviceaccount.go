package k8s

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ServiceAccountManagers interface {
	Get(namespace, name string) (*corev1.ServiceAccount, error)
	Create(object *corev1.ServiceAccount) error
	Update(object *corev1.ServiceAccount) error
	Delete(namespace, name string) error
}

type ServiceAccountManager struct {
	Client client.Client
	Logger logr.Logger
}

func NewServiceAccountManager(client client.Client, logger logr.Logger) *ServiceAccountManager {
	return &ServiceAccountManager{
		Client: client,
		Logger: logger,
	}
}

func (manager *ServiceAccountManager) Get(namespace, name string) (*corev1.ServiceAccount, error) {
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

func (manager *ServiceAccountManager) Create(object *corev1.ServiceAccount) error {
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

func (manager *ServiceAccountManager) Update(object *corev1.ServiceAccount) error {
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

func (manager *ServiceAccountManager) Delete(namespace, name string) error {
	object := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return manager.Client.Delete(context.TODO(), object)
}
