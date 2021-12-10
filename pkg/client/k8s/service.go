package k8s

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ServiceManagers interface {
	Get(namespace string, name string) (*corev1.Service, error)
	Create(object *corev1.Service) error
	Update(object *corev1.Service) error
	Delete(namespace, name string) error
}

type ServiceManager struct {
	Client client.Client
	Logger logr.Logger
}

func NewServiceManager(client client.Client, logger logr.Logger) *ServiceManager {
	return &ServiceManager{
		Client: client,
		Logger: logger,
	}
}

func (manager *ServiceManager) Get(namespace string, name string) (*corev1.Service, error) {
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

func (manager *ServiceManager) Create(object *corev1.Service) error {
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

func (manager *ServiceManager) Update(object *corev1.Service) error {
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

func (manager *ServiceManager) Delete(namespace string, name string) error {
	object := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return manager.Client.Delete(context.TODO(), object)
}
