package k8s

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Secret the client that knows how to interact with kubernetes to manage them
type Secret interface {
	// GetService get secret from kubernetes with namespace and name
	GetSecret(namespace string, name string) (*corev1.Secret, error)
	// CreateService will create the given secret
	CreateSecret(namespace string, secret *corev1.Secret) error
	//CreateIfNotExistsSecret create secret if it does not exist
	CreateIfNotExistsSecret(namespace string, secret *corev1.Secret) error
	// UpdateSecret will update the given secret
	UpdateSecret(namespace string, secret *corev1.Secret) error
	// CreateOrUpdateSecret will update the given secret or create it if does not exist
	CreateOrUpdateSecret(namespace string, secret *corev1.Secret) error
	// DeleteSecret will delete the given secret
	DeleteSecret(namespace string, name string) error
	// ListSecret get set of secret on a given namespace
	ListSecrets(namespace string) (*corev1.SecretList, error)
}

// SecretOption is the secret client implementation using API calls to kubernetes.
type SecretOption struct {
	client client.Client
	logger logr.Logger
}

// NewSecret returns a new Secret client.
func NewSecret(kubeClient client.Client, logger logr.Logger) Secret {
	logger = logger.WithValues("secret", "k8s.secret")
	return &SecretOption{
		client: kubeClient,
		logger: logger,
	}
}

// GetSecret implement the Service.Interface
func (s *SecretOption) GetSecret(namespace string, name string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := s.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, secret)

	if err != nil {
		return nil, err
	}
	return secret, err
}

// CreateSecret implement the Secret.Interface
func (s *SecretOption) CreateSecret(namespace string, secret *corev1.Secret) error {
	err := s.client.Create(context.TODO(), secret)
	if err != nil {
		return err
	}
	s.logger.WithValues("namespace", namespace, "secretName", secret.Name).Info("secret created")
	return nil
}

// CreateIfNotExistsSecret implement the Secret.Interface
func (s *SecretOption) CreateIfNotExistsSecret(namespace string, secret *corev1.Secret) error {
	if _, err := s.GetSecret(namespace, secret.Name); err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			return s.CreateSecret(namespace, secret)
		}
		return err
	}
	return nil
}

// UpdateSecret implement the Secret.Interface
func (s *SecretOption) UpdateSecret(namespace string, secret *corev1.Secret) error {
	err := s.client.Update(context.TODO(), secret)
	if err != nil {
		return err
	}
	s.logger.WithValues("namespace", namespace, "serviceName", secret.Name).Info("secret updated")
	return nil
}

// CreateOrUpdateSecret implement the Secret.Interface
func (s *SecretOption) CreateOrUpdateSecret(namespace string, secret *corev1.Secret) error {
	storedSecret, err := s.GetSecret(namespace, secret.Name)
	if err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			return s.CreateSecret(namespace, secret)
		}
		return err
	}

	// Already exists, need to Update.
	// Set the correct resource version to ensure we are on the latest version. This way the only valid
	// namespace is our spec(https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency),
	// we will replace the current namespace state.
	secret.ResourceVersion = storedSecret.ResourceVersion
	return s.UpdateSecret(namespace, secret)
}

// DeleteSecret implement the Secret.Interface
func (s *SecretOption) DeleteSecret(namespace string, name string) error {
	secret := &corev1.Secret{}
	if err := s.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, secret); err != nil {
		return err
	}
	return s.client.Delete(context.TODO(), secret)
}

// ListSecrets implement the Secret.Interface
func (s *SecretOption) ListSecrets(namespace string) (*corev1.SecretList, error) {
	secrets := &corev1.SecretList{}
	listOps := &client.ListOptions{
		Namespace: namespace,
	}
	err := s.client.List(context.TODO(), secrets, listOps)
	return secrets, err
}
