package k8s

import (
	"context"

	"github.com/emqx/emqx-operator/api/v1beta1"
	"k8s.io/apimachinery/pkg/types"
)

type EmqxBrokerManagers interface {
	GetEmqxBroker(emqx v1beta1.Emqx) error
}

func (manager *Manager) GetEmqxBroker(namespace, name string) (*v1beta1.EmqxBroker, error) {
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
