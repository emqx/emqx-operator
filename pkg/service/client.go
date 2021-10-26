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

// EnsureEmqxStatfulSet makes sure the emqx statefulset exists in the desired state
func (r *EmqxClusterKubeClient) EnsureEmqxStatefulSet(e *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	// TODO PDB
	oldSts, err := r.K8sService.GetStatefulSet(e.Namespace, util.GetEmqxName(e))
	if err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			sts := newEmqxStatefulSet(e, labels, ownerRefs)
			return r.K8sService.CreateStatefulSet(e.Namespace, sts)
		}
		return err
	}

	if shouldUpdateEmqx(e.Spec.Resources, oldSts.Spec.Template.Spec.Containers[0].Resources,
		e.Spec.Replicas, oldSts.Spec.Replicas) {
		es := newEmqxStatefulSet(e, labels, ownerRefs)
		return r.K8sService.UpdateStatefulSet(e.Namespace, es)
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
