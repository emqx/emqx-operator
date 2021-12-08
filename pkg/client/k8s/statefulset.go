package k8s

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type StatefulSetManagers interface {
	GetStatefulSet(namespace, name string) (*appsv1.StatefulSet, error)
	CreateStatefulSet(object *appsv1.StatefulSet) error
	UpdateStatefulSet(object *appsv1.StatefulSet) error
	DeleteStatefulSet(namespace, name string) error
}

func (manager *Manager) GetStatefulSet(namespace, name string) (*appsv1.StatefulSet, error) {
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

func (manager *Manager) CreateStatefulSet(object *appsv1.StatefulSet) error {
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

func (manager *Manager) UpdateStatefulSet(object *appsv1.StatefulSet) error {
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

func (manager *Manager) DeleteStatefulSet(namespace, name string) error {
	object := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return manager.Client.Delete(context.TODO(), object)
}
