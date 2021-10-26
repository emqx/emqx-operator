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
	EnsureEmqxSecret(emqx *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxHeadlessService(emqx *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxConfigMapForLoadedModules(emqx *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxConfigMapForLoadedPlugins(emqx *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
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

// EnsureEmqxSecret make sure the EMQ X secret exists
func (r *EmqxClusterKubeClient) EnsureEmqxSecret(e *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	secret := newSecretForCR(e, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsSecret(e.Namespace, secret)
}

// EnsureEmqxHeadlessService makes sure the EMQ X headless service exists
func (r *EmqxClusterKubeClient) EnsureEmqxHeadlessService(e *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	svc := newHeadLessSvcForCR(e, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsService(e.Namespace, svc)
}

// EnsureEmqxConfigMapForLoadedModules make sure the EMQ X configmap for loaded modules exists
func (r *EmqxClusterKubeClient) EnsureEmqxConfigMapForLoadedModules(e *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	configmapForLM := newConfigMapForLoadedMoudles(e, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsConfigMap(e.Namespace, configmapForLM)
}

// EnsureEmqxConfigMapForLoadedPlugins make sure the EMQ X configmap for loaded plugins exists
func (r *EmqxClusterKubeClient) EnsureEmqxConfigMapForLoadedPlugins(e *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	configmapForPG := newConfigMapForLoadedPlugins(e, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsConfigMap(e.Namespace, configmapForPG)
}
