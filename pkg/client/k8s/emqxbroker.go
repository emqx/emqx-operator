package k8s

import (
	"context"

	"github.com/emqx/emqx-operator/api/v1beta1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EmqxBrokerManagers interface {
	Get(emqx v1beta1.EmqxBroker) error
	UpdateStatus(emqx v1beta1.EmqxBroker) error
}

type EmqxBrokerManager struct {
	Client client.Client
	Logger logr.Logger
}

func NewEmqxBrokerManager(client client.Client, logger logr.Logger) *EmqxBrokerManager {
	return &EmqxBrokerManager{
		Client: client,
		Logger: logger,
	}
}

func (manager *EmqxBrokerManager) Get(namespace, name string) (*v1beta1.EmqxBroker, error) {
	object := &v1beta1.EmqxBroker{}
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

func (manager *EmqxBrokerManager) UpdateStatus(emqx *v1beta1.EmqxBroker) error {
	emqx.DescConditionsByTime()
	err := manager.Client.Status().Update(context.TODO(), emqx)
	if err != nil {
		manager.Logger.WithValues(
			"kind", emqx.GetKind(),
			"apiVersion", emqx.GetAPIVersion(),
			"namespace", emqx.GetNamespace(),
			"name", emqx.GetName(),
			"conditions", emqx.GetConditions(),
		).Error(err, "Update emqx broker status unsuccessfully")
		return err
	}
	manager.Logger.WithValues(
		"kind", emqx.GetKind(),
		"apiVersion", emqx.GetAPIVersion(),
		"namespace", emqx.GetNamespace(),
		"name", emqx.GetName(),
		"conditions", emqx.GetConditions(),
	).V(3).Info("Update emqx broker status successfully")
	return nil
}
