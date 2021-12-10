package k8s

import (
	"context"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatefulSetManagers interface {
	Get(namespace, name string) (*appsv1.StatefulSet, error)
	Create(object *appsv1.StatefulSet) error
	Update(object *appsv1.StatefulSet) error
	Delete(namespace, name string) error
}

type StatefulSetManager struct {
	Client client.Client
	Logger logr.Logger
}

func NewStatefulSetManager(client client.Client, logger logr.Logger) *StatefulSetManager {
	return &StatefulSetManager{
		Client: client,
		Logger: logger,
	}
}

func (manager *StatefulSetManager) Get(namespace, name string) (*appsv1.StatefulSet, error) {
	object := &appsv1.StatefulSet{}
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

func (manager *StatefulSetManager) Create(object *appsv1.StatefulSet) error {
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
	return err
}

func (manager *StatefulSetManager) Update(object *appsv1.StatefulSet) error {
	err := manager.Client.Update(context.TODO(), object)
	if err != nil {
		return err
	}
	manager.Logger.WithValues(
		"kind", object.Kind,
		"apiVersion", object.APIVersion,
		"namespace", object.Namespace,
		"serviceName", object.Name,
	).Info("Update successfully")
	return err
}

func (manager *StatefulSetManager) Delete(namespace, name string) error {
	object := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return manager.Client.Delete(context.TODO(), object)
}
