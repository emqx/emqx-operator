package k8s

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SecretManagers interface {
	Get(namespace, name string) (*corev1.Secret, error)
	Create(object *corev1.Secret) error
	Update(object *corev1.Secret) error
	Delete(namespace, name string) error
}

type SecretManager struct {
	Client client.Client
	Logger logr.Logger
}

func NewSecretManager(client client.Client, logger logr.Logger) *SecretManager {
	return &SecretManager{
		Client: client,
		Logger: logger,
	}
}

// GetSecret implement the Service.Interface
func (manager *SecretManager) Get(namespace, name string) (*corev1.Secret, error) {
	object := &corev1.Secret{}
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

func (manager *SecretManager) Create(object *corev1.Secret) error {
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

func (manager *SecretManager) Update(object *corev1.Secret) error {
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

func (manager *SecretManager) Delete(namespace string, name string) error {
	object := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return manager.Client.Delete(context.TODO(), object)
}
