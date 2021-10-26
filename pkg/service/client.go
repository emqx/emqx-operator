package service

import (
	"github.com/emqx/emqx-operator/api/v1alpha1"
	"github.com/emqx/emqx-operator/pkg/client/k8s"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EmqxClusterClient has the minimumm methods that a EMQ X Cluster controller needs to satisfy
// in order to talk with K8s
type EmqxClusterClient interface {
	// EnsureEmqxConfigMap(emqx *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxSecret(emqx *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxHeadlessService(emqx *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	// EnsureEmqxStatefulSet(emqx *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
}

// EmqxClusterKubeClient implements the required methods to talk with kubernetes
type EmqxClusterKubeClient struct {
	K8sService k8s.Services
	Logger     logr.Logger
}

// NewEmqxClusterKubeClient creates a new EmqxClusterKubeClient
func NewEmqxClusterKubeClient(k8sService k8s.Services, logger logr.Logger) *EmqxClusterKubeClient {
	return &EmqxClusterKubeClient{
		K8sService: k8sService,
		Logger:     logger,
	}
}

func generateSelectorLabels(component, name string) map[string]string {
	return map[string]string{}
}

// EnsureEmqxHeadlessService makes sure the EMQ X headless service exists
func (r *EmqxClusterKubeClient) EnsureEmqxHeadlessService(e *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	svc := newHeadLessSvcForCR(e, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsService(e.Namespace, svc)
}

// EnsureEmqxSecret make sure the EMQ X secret exists
func (r *EmqxClusterKubeClient) EnsureEmqxSecret(e *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	secret := newSecretForCR(e, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsSecret(e.Namespace, secret)
}
