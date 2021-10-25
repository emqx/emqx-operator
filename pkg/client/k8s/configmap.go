package k8s

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ConfigMap the client that knows how to interact with kubernetes to manage them
type ConfigMap interface {
	// GetConfigMap get ConfigMap from kubernetes with namespace and name
	GetConfigMap(namespace string, name string) (*corev1.ConfigMap, error)
	// CreateConfigMap create the given ConfigMap
	CreateConfigMap(namespace string, configMap *corev1.ConfigMap) error
	// UpdateConfigMap update the given ConfigMap
	UpdateConfigMap(namespace string, configMap *corev1.ConfigMap) error
	// CreateOrUpdateConfigMap if the ConfigMap Already exists, create it, otherwise update it
	CreateOrUpdateConfigMap(namespace string, np *corev1.ConfigMap) error
	// DeleteConfigMap delete ConfigMap from kubernetes with namespace and name
	DeleteConfigMap(namespace string, name string) error
	// ListConfigMaps get set of ConfigMaps on a given namespace
	ListConfigMaps(namespace string) (*corev1.ConfigMapList, error)
	CreateIfNotExistsConfigMap(namespace string, configMap *corev1.ConfigMap) error
}

// ConfigMapOption is the configMap client interface implementation that using API calls to kubernetes.
type ConfigMapOption struct {
	client client.Client
	logger logr.Logger
}

// NewConfigMap returns a new ConfigMap client.
func NewConfigMap(kubeClient client.Client, logger logr.Logger) ConfigMap {
	logger = logger.WithValues("service", "k8s.configMap")
	return &ConfigMapOption{
		client: kubeClient,
		logger: logger,
	}
}

// GetConfigMap implement the  ConfigMap.Interface
func (p *ConfigMapOption) GetConfigMap(namespace string, name string) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	err := p.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, configMap)

	if err != nil {
		return nil, err
	}
	return configMap, err
}

// CreateConfigMap implement the  ConfigMap.Interface
func (p *ConfigMapOption) CreateConfigMap(namespace string, configMap *corev1.ConfigMap) error {
	err := p.client.Create(context.TODO(), configMap)
	if err != nil {
		return err
	}
	p.logger.WithValues("namespace", namespace, "configMap", configMap.Name).Info("configMap created")
	return nil
}

// UpdateConfigMap implement the  ConfigMap.Interface
func (p *ConfigMapOption) UpdateConfigMap(namespace string, configMap *corev1.ConfigMap) error {
	err := p.client.Update(context.TODO(), configMap)
	if err != nil {
		return err
	}
	p.logger.WithValues("namespace", namespace, "configMap", configMap.Name).Info("configMap updated")
	return nil
}

// CreateIfNotExistsConfigMap implement the ConfigMap.Interface
func (p *ConfigMapOption) CreateIfNotExistsConfigMap(namespace string, configMap *corev1.ConfigMap) error {
	if _, err := p.GetConfigMap(namespace, configMap.Name); err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			return p.CreateConfigMap(namespace, configMap)
		}
		return err
	}
	return nil
}

// CreateOrUpdateConfigMap implement the  ConfigMap.Interface
func (p *ConfigMapOption) CreateOrUpdateConfigMap(namespace string, configMap *corev1.ConfigMap) error {
	storedConfigMap, err := p.GetConfigMap(namespace, configMap.Name)
	if err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			return p.CreateConfigMap(namespace, configMap)
		}
		return err
	}

	// Already exists, need to Update.
	// Set the correct resource version to ensure we are on the latest version. This way the only valid
	// namespace is our spec(https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency),
	// we will replace the current namespace state.
	configMap.ResourceVersion = storedConfigMap.ResourceVersion
	return p.UpdateConfigMap(namespace, configMap)
}

// DeleteConfigMap implement the  ConfigMap.Interface
func (p *ConfigMapOption) DeleteConfigMap(namespace string, name string) error {
	configMap := &corev1.ConfigMap{}
	if err := p.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, configMap); err != nil {
		return err
	}
	return p.client.Delete(context.TODO(), configMap)
}

// ListConfigMaps implement the  ConfigMap.Interface
func (p *ConfigMapOption) ListConfigMaps(namespace string) (*corev1.ConfigMapList, error) {
	cms := &corev1.ConfigMapList{}
	listOps := &client.ListOptions{
		Namespace: namespace,
	}
	err := p.client.List(context.TODO(), cms, listOps)
	return cms, err
}
