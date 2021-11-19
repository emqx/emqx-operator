package service

import (
	"github.com/emqx/emqx-operator/api/v1alpha2"
	"github.com/emqx/emqx-operator/pkg/client/k8s"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EmqxClusterClient has the minimumm methods that a EMQ X Cluster controller needs to satisfy
// in order to talk with K8s
type EmqxClusterClient interface {
	EnsureEmqxSecret(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxHeadlessService(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxConfigMapForAcl(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxConfigMapForLoadedModules(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxConfigMapForLoadedPlugins(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxStatefulSet(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
}

// EmqxClusterKubeClient implements the required methods to talk with kubernetes
type EmqxClusterKubeClient struct {
	K8sService k8s.Services
	Logger     logr.Logger
}

// NewEmqxClusterKubeClient creates a New EmqxClusterKubeClient
func NewEmqxClusterKubeClient(k8sService k8s.Services, logger logr.Logger) *EmqxClusterKubeClient {
	return &EmqxClusterKubeClient{
		K8sService: k8sService,
		Logger:     logger,
	}
}

// EnsureEmqxSecret make sure the EMQ X secret exists
func (r *EmqxClusterKubeClient) EnsureEmqxSecret(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	secret := NewSecretForCR(emqx, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsSecret(emqx.GetNamespace(), secret)
}

// EnsureEmqxHeadlessService makes sure the EMQ X headless service exists
func (r *EmqxClusterKubeClient) EnsureEmqxHeadlessService(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	svc := NewHeadLessSvcForCR(emqx, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsService(emqx.GetNamespace(), svc)
}

func (r *EmqxClusterKubeClient) EnsureEmqxConfigMapForAcl(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	configmapForAcl := NewConfigMapForAcl(emqx, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsConfigMap(emqx.GetNamespace(), configmapForAcl)
}

// EnsureEmqxConfigMapForLoadedModules make sure the EMQ X configmap for loaded modules exists
func (r *EmqxClusterKubeClient) EnsureEmqxConfigMapForLoadedModules(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	configmapForLM := NewConfigMapForLoadedMoudles(emqx, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsConfigMap(emqx.GetNamespace(), configmapForLM)
}

// EnsureEmqxConfigMapForLoadedPlugins make sure the EMQ X configmap for loaded plugins exists
func (r *EmqxClusterKubeClient) EnsureEmqxConfigMapForLoadedPlugins(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	configmapForPG := NewConfigMapForLoadedPlugins(emqx, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsConfigMap(emqx.GetNamespace(), configmapForPG)
}

// EnsureEmqxStatefulSet makes sure the emqx statefulset exists in the desired state
func (r *EmqxClusterKubeClient) EnsureEmqxStatefulSet(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	// TODO PDB
	oldSts, err := r.K8sService.GetStatefulSet(emqx.GetNamespace(), emqx.GetName())
	if err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			sts := NewEmqxStatefulSet(emqx, labels, ownerRefs)
			return r.K8sService.CreateStatefulSet(emqx.GetNamespace(), sts)
		}
		return err
	}

	if shouldUpdateEmqx(emqx.GetResource(), oldSts.Spec.Template.Spec.Containers[0].Resources,
		emqx.GetReplicas(), oldSts.Spec.Replicas) {
		es := NewEmqxStatefulSet(emqx, labels, ownerRefs)
		return r.K8sService.UpdateStatefulSet(emqx.GetNamespace(), es)
	}

	return nil
}

func shouldUpdateEmqx(expectResource, containterResource corev1.ResourceRequirements, expectSize, replicas *int32) bool {
	if expectSize != replicas {
		return true
	}
	if result := containterResource.Requests.Cpu().Cmp(*expectResource.Requests.Cpu()); result != 0 {
		return true
	}
	if result := containterResource.Requests.Memory().Cmp(*expectResource.Requests.Memory()); result != 0 {
		return true
	}
	if result := containterResource.Limits.Cpu().Cmp(*expectResource.Limits.Cpu()); result != 0 {
		return true
	}
	if result := containterResource.Limits.Memory().Cmp(*expectResource.Limits.Memory()); result != 0 {
		return true
	}
	return false
}
