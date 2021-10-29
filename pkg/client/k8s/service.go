package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Service the client that knows how to interact with kubernetes to manage them
type Service interface {
	// GetService get service from kubernetes with namespace and name
	GetService(namespace string, name string) (*corev1.Service, error)
	// CreateService will create the given service
	CreateService(namespace string, service *corev1.Service) error
	//CreateIfNotExistsService create service if it does not exist
	CreateIfNotExistsService(namespace string, service *corev1.Service) error
	// UpdateService will update the given service
	UpdateService(namespace string, service *corev1.Service) error
	// CreateOrUpdateService will update the given service or create it if does not exist
	CreateOrUpdateService(namespace string, service *corev1.Service) error
	// DeleteService will delete the given service
	DeleteService(namespace string, name string) error
	// ListServices get set of service on a given namespace
	ListServices(namespace string) (*corev1.ServiceList, error)
}

// ServiceOption is the service client implementation using API calls to kubernetes.
type ServiceOption struct {
	client client.Client
	logger logr.Logger
}

// NewService returns a new Service client.
func NewService(kubeClient client.Client, logger logr.Logger) Service {
	logger = logger.WithValues("service", "k8s.service")
	return &ServiceOption{
		client: kubeClient,
		logger: logger,
	}
}

// GetService implement the Service.Interface
func (s *ServiceOption) GetService(namespace string, name string) (*corev1.Service, error) {
	service := &corev1.Service{}
	err := s.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, service)

	if err != nil {
		return nil, err
	}
	return service, err
}

// CreateService implement the Service.Interface
func (s *ServiceOption) CreateService(namespace string, service *corev1.Service) error {
	err := s.client.Create(context.TODO(), service)
	if err != nil {
		return err
	}
	s.logger.WithValues("namespace", namespace, "serviceName", service.Name).Info("service created")
	return nil
}

// CreateIfNotExistsService implement the Service.Interface
func (s *ServiceOption) CreateIfNotExistsService(namespace string, service *corev1.Service) error {
	if _, err := s.GetService(namespace, service.Name); err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			return s.CreateService(namespace, service)
		}
		return err
	}
	return nil
}

// UpdateService implement the Service.Interface
func (s *ServiceOption) UpdateService(namespace string, service *corev1.Service) error {
	err := s.client.Update(context.TODO(), service)
	if err != nil {
		return err
	}
	s.logger.WithValues("namespace", namespace, "serviceName", service.Name).Info("service updated")
	return nil
}

// CreateOrUpdateService implement the Service.Interface
func (s *ServiceOption) CreateOrUpdateService(namespace string, service *corev1.Service) error {
	storedService, err := s.GetService(namespace, service.Name)
	if err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			return s.CreateService(namespace, service)
		}
		return err
	}

	// Already exists, need to Update.
	// Set the correct resource version to ensure we are on the latest version. This way the only valid
	// namespace is our spec(https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency),
	// we will replace the current namespace state.
	service.ResourceVersion = storedService.ResourceVersion
	return s.UpdateService(namespace, service)
}

// DeleteService implement the Service.Interface
func (s *ServiceOption) DeleteService(namespace string, name string) error {
	service := &corev1.Service{}
	if err := s.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, service); err != nil {
		return err
	}
	return s.client.Delete(context.TODO(), service)
}

// ListServices implement the Service.Interface
func (s *ServiceOption) ListServices(namespace string) (*corev1.ServiceList, error) {
	services := &corev1.ServiceList{}
	listOps := &client.ListOptions{
		Namespace: namespace,
	}
	err := s.client.List(context.TODO(), services, listOps)
	return services, err
}
