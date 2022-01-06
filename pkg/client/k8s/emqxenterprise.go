package k8s

import (
	"context"

	"github.com/emqx/emqx-operator/apis/apps/v1beta1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EmqxEnterpriseManagers interface {
	Get(emqx v1beta1.EmqxEnterprise) error
	UpdateStatus(emqx v1beta1.EmqxEnterprise) error
}

type EmqxEnterpriseManager struct {
	Client client.Client
	Logger logr.Logger
}

func NewEmqxEnterpeiseManager(client client.Client, logger logr.Logger) *EmqxEnterpriseManager {
	return &EmqxEnterpriseManager{
		Client: client,
		Logger: logger,
	}
}

func (manager *EmqxEnterpriseManager) Get(namespace, name string) (*v1beta1.EmqxEnterprise, error) {
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

func (manager *EmqxEnterpriseManager) UpdateStatus(emqx *v1beta1.EmqxEnterprise) error {
	emqx.DescConditionsByTime()
	err := manager.Client.Status().Update(context.TODO(), emqx)
	if err != nil {
		manager.Logger.WithValues(
			"kind", emqx.GetKind(),
			"apiVersion", emqx.GetAPIVersion(),
			"namespace", emqx.GetNamespace(),
			"name", emqx.GetName(),
			"conditions", emqx.GetConditions(),
		).Error(err, "Update emqx enterprise status unsuccessfully")
		return err
	}
	manager.Logger.WithValues(
		"kind", emqx.GetKind(),
		"apiVersion", emqx.GetAPIVersion(),
		"namespace", emqx.GetNamespace(),
		"name", emqx.GetName(),
		"conditions", emqx.GetConditions(),
	).V(3).Info("Update emqx enterprise status successfully")
	return nil
}
