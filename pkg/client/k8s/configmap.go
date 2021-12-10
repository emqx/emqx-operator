package k8s

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConfigMapManagers interface {
	Get(namespace, name string) (*corev1.ConfigMap, error)
	Create(configMap *corev1.ConfigMap) error
	Update(configMap *corev1.ConfigMap) error
	Delete(namespace, name string) error
}

type ConfigMapManager struct {
	Client client.Client
	Logger logr.Logger
}

func NewConfigMapManager(client client.Client, logger logr.Logger) *ConfigMapManager {
	return &ConfigMapManager{
		Client: client,
		Logger: logger,
	}
}

func (manager *ConfigMapManager) Get(namespace string, name string) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	err := manager.Client.Get(
		context.TODO(),
		types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, configMap,
	)

	if err != nil {
		return nil, err
	}
	return configMap, err
}

func (manager *ConfigMapManager) Create(object *corev1.ConfigMap) error {
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

func (manager *ConfigMapManager) Update(object *corev1.ConfigMap) error {
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

func (manager *ConfigMapManager) Delete(namespace, name string) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return manager.Client.Delete(context.TODO(), configMap)
}
