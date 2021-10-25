package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
)

// Pod the client that knows how to interact with kubernetes to manage them
type Pod interface {
	// GetPod get pod from kubernetes with namespace and name
	GetPod(namespace string, name string) (*corev1.Pod, error)
	// CreatePod will create the given pod
	CreatePod(namespace string, pod *corev1.Pod) error
	// UpdatePod will update the given pod
	UpdatePod(namespace string, pod *corev1.Pod) error
	// CreateOrUpdatePod will update the given pod or create it if does not exist
	CreateOrUpdatePod(namespace string, pod *corev1.Pod) error
	// DeletePod will delete the given pod
	DeletePod(namespace string, name string) error
	// ListPods get set of pod on a given namespace
	ListPods(namespace string) (*corev1.PodList, error)
}

// PodOption is the pod client interface implementation using API calls to kubernetes.
type PodOption struct {
	client client.Client
	logger logr.Logger
}

// NewPod returns a new Pod client.
func NewPod(kubeClient client.Client, logger logr.Logger) Pod {
	logger = logger.WithValues("service", "k8s.pod")
	return &PodOption{
		client: kubeClient,
		logger: logger,
	}
}

// GetPod implement the Pod.Interface
func (p *PodOption) GetPod(namespace string, name string) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	err := p.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, pod)
	if err != nil {
		return nil, err
	}
	return pod, err
}

// CreatePod implement the Pod.Interface
func (p *PodOption) CreatePod(namespace string, pod *corev1.Pod) error {
	err := p.client.Create(context.TODO(), pod)
	if err != nil {
		return err
	}

	p.logger.WithValues("namespace", namespace, "pod", pod.Name).Info("pod created")
	return nil
}

// UpdatePod implement the Pod.Interface
func (p *PodOption) UpdatePod(namespace string, pod *corev1.Pod) error {
	err := p.client.Update(context.TODO(), pod)
	if err != nil {
		return err
	}
	p.logger.WithValues("namespace", namespace, "pod", pod.Name).Info("pod updated")
	return nil
}

// CreateOrUpdatePod implement the Pod.Interface
func (p *PodOption) CreateOrUpdatePod(namespace string, pod *corev1.Pod) error {
	storedPod, err := p.GetPod(namespace, pod.Name)
	if err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			return p.CreatePod(namespace, pod)
		}
		return err
	}

	// Already exists, need to Update.
	// Set the correct resource version to ensure we are on the latest version. This way the only valid
	// namespace is our spec(https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency),
	// we will replace the current namespace state.
	pod.ResourceVersion = storedPod.ResourceVersion
	return p.UpdatePod(namespace, pod)
}

// DeletePod implement the Pod.Interface
func (p *PodOption) DeletePod(namespace string, name string) error {
	pod := &corev1.Pod{}
	if err := p.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, pod); err != nil {
		return err
	}
	return p.client.Delete(context.TODO(), pod)
}

// ListPods implement the Pod.Interface
func (p *PodOption) ListPods(namespace string) (*corev1.PodList, error) {
	ps := &corev1.PodList{}
	listOps := &client.ListOptions{
		Namespace: namespace,
	}
	err := p.client.List(context.TODO(), ps, listOps)
	return ps, err
}
