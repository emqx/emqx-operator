package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
)

// NameSpaces the client that knows how to interact with kubernetes to manage them
type NameSpaces interface {
	// GetNameSpace get namespace info form kubernetes
	GetNameSpace(namespace string) (*corev1.Namespace, error)
}

// NameSpacesOption is the NameSpaces client implementation using API calls to kubernetes.
type NameSpacesOption struct {
	client client.Client
	logger logr.Logger
}

// NewNameSpaces returns a new NameSpaces client.
func NewNameSpaces(logger logr.Logger) NameSpaces {
	logger = logger.WithValues("service", "k8s.namespaces")
	cfg, err := config.GetConfig()
	if err != nil {
		panic(err)
	}
	kubeClient, err := client.New(cfg, client.Options{})
	if err != nil {
		panic(err)
	}
	return &NameSpacesOption{
		client: kubeClient,
		logger: logger,
	}
}

// GetNameSpace implement the NameSpaces.Interface
func (n *NameSpacesOption) GetNameSpace(namespace string) (*corev1.Namespace, error) {
	nm := &corev1.Namespace{}
	err := n.client.Get(context.TODO(), types.NamespacedName{
		Name:      namespace,
		Namespace: namespace,
	}, nm)
	if err != nil {
		return nil, err
	}
	return nm, err
}
