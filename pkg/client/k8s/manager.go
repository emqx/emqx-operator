package k8s

import (
	"github.com/go-logr/logr"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Manager struct {
	EmqxBroker     EmqxBrokerManager
	EmqxEnterprise EmqxEnterpriseManager
	ConfigMap      ConfigMapManager
	Role           RoleManager
	RoleBinding    RoleBindingManager
	Secret         SecretManager
	Service        ServiceManager
	ServiceAccount ServiceAccountManager
	StatefulSet    StatefulSetManager
}

func NewManager(client client.Client, logger logr.Logger) *Manager {
	return &Manager{
		EmqxBroker:     *NewEmqxBrokerManager(client, logger),
		EmqxEnterprise: *NewEmqxEnterpeiseManager(client, logger),
		ConfigMap:      *NewConfigMapManager(client, logger),
		Role:           *NewRoleManager(client, logger),
		RoleBinding:    *NewRoleBindingManager(client, logger),
		Secret:         *NewSecretManager(client, logger),
		Service:        *NewServiceManager(client, logger),
		ServiceAccount: *NewServiceAccountManager(client, logger),
		StatefulSet:    *NewStatefulSetManager(client, logger),
	}
}
