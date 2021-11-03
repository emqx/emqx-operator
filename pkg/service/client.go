package service

import (
	"github.com/emqx/emqx-operator/api/v1alpha1"
	"github.com/emqx/emqx-operator/pkg/client/k8s"
	"github.com/emqx/emqx-operator/pkg/util"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EmqxBrokerClusterClient has the minimumm methods that a EMQ X Cluster controller needs to satisfy
// in order to talk with K8s
type EmqxBrokerClusterClient interface {
	EnsureEmqxBrokerSecret(emqx *v1alpha1.EmqxBroker, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxBrokerHeadlessService(emqx *v1alpha1.EmqxBroker, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxBrokerConfigMapForAcl(emqx *v1alpha1.EmqxBroker, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxBrokerConfigMapForLoadedModules(emqx *v1alpha1.EmqxBroker, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxBrokerConfigMapForLoadedPlugins(emqx *v1alpha1.EmqxBroker, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureEmqxBrokerStatefulSet(emqx *v1alpha1.EmqxBroker, labels map[string]string, ownerRefs []metav1.OwnerReference) error
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

func generateSelectorLabels(component, name string) map[string]string {
	return map[string]string{}
}

// EnsureEmqxBrokerSecret make sure the EMQ X secret exists
func (r *EmqxBrokerClusterKubeClient) EnsureEmqxBrokerSecret(e *v1alpha1.EmqxBroker, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	secret := newSecretForCR(e, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsSecret(e.Namespace, secret)
}

// EnsureEmqxBrokerHeadlessService makes sure the EMQ X headless service exists
func (r *EmqxBrokerClusterKubeClient) EnsureEmqxBrokerHeadlessService(e *v1alpha1.EmqxBroker, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	svc := newHeadLessSvcForCR(e, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsService(e.Namespace, svc)
}

func (r *EmqxBrokerClusterKubeClient) EnsureEmqxBrokerConfigMapForAcl(e *v1alpha1.EmqxBroker, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	configmapForAcl := newConfigMapForAcl(e, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsConfigMap(e.Namespace, configmapForAcl)
}

// EnsureEmqxBrokerConfigMapForLoadedModules make sure the EMQ X configmap for loaded modules exists
func (r *EmqxBrokerClusterKubeClient) EnsureEmqxBrokerConfigMapForLoadedModules(e *v1alpha1.EmqxBroker, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	configmapForLM := newConfigMapForLoadedMoudles(e, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsConfigMap(e.Namespace, configmapForLM)
}

// EnsureEmqxBrokerConfigMapForLoadedPlugins make sure the EMQ X configmap for loaded plugins exists
func (r *EmqxBrokerClusterKubeClient) EnsureEmqxBrokerConfigMapForLoadedPlugins(e *v1alpha1.EmqxBroker, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	configmapForPG := newConfigMapForLoadedPlugins(e, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsConfigMap(e.Namespace, configmapForPG)
}

// EnsureEmqxBrokerStatefulSet makes sure the emqx statefulset exists in the desired state
func (r *EmqxBrokerClusterKubeClient) EnsureEmqxBrokerStatefulSet(e *v1alpha1.EmqxBroker, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	// TODO PDB
	oldSts, err := r.K8sService.GetStatefulSet(e.Namespace, util.GetEmqxBrokerName(e))
	if err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			sts := newEmqxBrokerStatefulSet(e, labels, ownerRefs)
			return r.K8sService.CreateStatefulSet(e.Namespace, sts)
		}
		return err
	}

	if shouldUpdateEmqxBroker(e.Spec.Resources, oldSts.Spec.Template.Spec.Containers[0].Resources,
		e.Spec.Replicas, oldSts.Spec.Replicas) {
		es := newEmqxBrokerStatefulSet(e, labels, ownerRefs)
		return r.K8sService.UpdateStatefulSet(e.Namespace, es)
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
