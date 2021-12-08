package k8s

import (
	"context"

	"github.com/emqx/emqx-operator/api/v1beta1"
	"github.com/go-logr/logr"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Managers interface {
	ConfigMapManagers
	NamespacesManagers
	RoleManagers
	RoleBindingManagers
	SecretManagers
	ServiceManagers
	ServiceAccountManagers
	StatefulSetManagers
}

type Manager struct {
	Client client.Client
	Logger logr.Logger
}

func NewManager(client client.Client, logger logr.Logger) *Manager {
	return &Manager{
		Client: client,
		Logger: logger,
	}
}

func (manager *Manager) UpdateEmqxStatus(emqx v1beta1.Emqx) error {
	emqx.DescConditionsByTime()
	err := manager.Client.Status().Update(context.TODO(), emqx)
	if err != nil {
		manager.Logger.WithValues(
			"kind", emqx.GetKind(),
			"apiVersion", emqx.GetAPIVersion(),
			"namespace", emqx.GetNamespace(),
			"name", emqx.GetName(),
			"conditions", emqx.GetConditions(),
		).Error(err, "emqxStatus")
		return err
	}
	manager.Logger.WithValues(
		"kind", emqx.GetKind(),
		"apiVersion", emqx.GetAPIVersion(),
		"namespace", emqx.GetNamespace(),
		"name", emqx.GetName(),
		"conditions", emqx.GetConditions(),
	).V(3).Info("emqxStatus updated")
	return nil
}
