package k8s

import (
	"context"

	"github.com/emqx/emqx-operator/api/v1beta1"
	"k8s.io/apimachinery/pkg/types"
)

type EmqxEnterpriseManagers interface {
	GetEmqxEnterprise(emqx v1beta1.Emqx) error
}

func (manager *Manager) GetEmqxEnterprise(namespace, name string) (*v1beta1.EmqxEnterprise, error) {
	object := &v1beta1.EmqxEnterprise{}
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
