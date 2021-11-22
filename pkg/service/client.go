package service

import (
	"github.com/emqx/emqx-operator/api/v1alpha2"
	"github.com/emqx/emqx-operator/pkg/client/k8s"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EmqxBrokerClusterClient has the minimumm methods that a EMQ X Cluster controller needs to satisfy
// in order to talk with K8s
type EmqxBrokerClusterClient interface {
	EnsureEmqxBrokerSecret(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxBrokerHeadlessService(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxBrokerConfigMapForAcl(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxBrokerConfigMapForLoadedModules(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxBrokerConfigMapForLoadedPlugins(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxBrokerStatefulSet(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
}

// EmqxBrokerClusterKubeClient implements the required methods to talk with kubernetes
type EmqxBrokerClusterKubeClient struct {
	K8sService k8s.Services
	Logger     logr.Logger
}

// NewEmqxBrokerClusterKubeClient creates a new EmqxBrokerClusterKubeClient
func NewEmqxBrokerClusterKubeClient(k8sService k8s.Services, logger logr.Logger) *EmqxBrokerClusterKubeClient {
	return &EmqxBrokerClusterKubeClient{
		K8sService: k8sService,
		Logger:     logger,
	}
}

// EnsureEmqxBrokerSecret make sure the EMQ X secret exists
func (r *EmqxBrokerClusterKubeClient) EnsureEmqxBrokerSecret(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	secret := NewSecretForCR(emqx, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsSecret(emqx.GetNamespace(), secret)
}

// EnsureEmqxBrokerHeadlessService makes sure the EMQ X headless service exists
func (r *EmqxBrokerClusterKubeClient) EnsureEmqxBrokerHeadlessService(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	svc := NewHeadLessSvcForCR(emqx, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsService(emqx.GetNamespace(), svc)
}

func (r *EmqxBrokerClusterKubeClient) EnsureEmqxBrokerConfigMapForAcl(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	configmapForAcl := NewConfigMapForAcl(emqx, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsConfigMap(emqx.GetNamespace(), configmapForAcl)
}

// EnsureEmqxBrokerConfigMapForLoadedModules make sure the EMQ X configmap for loaded modules exists
func (r *EmqxBrokerClusterKubeClient) EnsureEmqxBrokerConfigMapForLoadedModules(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	configmapForLM := NewConfigMapForLoadedMoudles(emqx, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsConfigMap(emqx.GetNamespace(), configmapForLM)
}

// EnsureEmqxBrokerConfigMapForLoadedPlugins make sure the EMQ X configmap for loaded plugins exists
func (r *EmqxBrokerClusterKubeClient) EnsureEmqxBrokerConfigMapForLoadedPlugins(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	configmapForPG := NewConfigMapForLoadedPlugins(emqx, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsConfigMap(emqx.GetNamespace(), configmapForPG)
}

// EnsureEmqxBrokerStatefulSet makes sure the emqx statefulset exists in the desired state
func (r *EmqxBrokerClusterKubeClient) EnsureEmqxBrokerStatefulSet(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	// TODO PDB
	oldSts, err := r.K8sService.GetStatefulSet(emqx.GetNamespace(), emqx.GetName())
	if err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			sts := NewEmqxBrokerStatefulSet(emqx, labels, ownerRefs)
			return r.K8sService.CreateStatefulSet(emqx.GetNamespace(), sts)
		}
		return err
	}

	if shouldUpdateEmqxBroker(emqx.GetResource(), oldSts.Spec.Template.Spec.Containers[0].Resources,
		emqx.GetReplicas(), oldSts.Spec.Replicas) {
		es := NewEmqxBrokerStatefulSet(emqx, labels, ownerRefs)
		return r.K8sService.UpdateStatefulSet(emqx.GetNamespace(), es)
	}

	return nil
}

func shouldUpdateEmqxBroker(expectResource, containterResource corev1.ResourceRequirements, expectSize, replicas *int32) bool {
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
