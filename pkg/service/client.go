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
	EnsureEmqxListenerService(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error
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
	// oldSecret, err := r.K8sService.GetSecret(emqx.GetNamespace(), emqx.GetName())
	oldSecret, err := r.K8sService.GetSecret(emqx.GetNamespace(), emqx.GetSecretName())

	if err != nil {
		// If no secret exists we need to create.
		if errors.IsNotFound(err) {
			secret := NewSecretForCR(emqx, labels, ownerRefs)
			return r.K8sService.CreateSecret(emqx.GetNamespace(), secret)
		}
		return err
	}

	if shouldUpdateSecret(emqx.GetLicense(), string(oldSecret.Data["emqx.lic"])) {
		secret := NewSecretForCR(emqx, labels, ownerRefs)
		return r.K8sService.UpdateSecret(emqx.GetNamespace(), secret)
	}

	return nil
}

// EnsureEmqxHeadlessService makes sure the EMQ X headless service exists
func (r *EmqxClusterKubeClient) EnsureEmqxHeadlessService(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	svc := NewHeadLessSvcForCR(emqx, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsService(emqx.GetNamespace(), svc)
}

//  EnsureEmqxConfigMapForACL make sure the EMQ X configmap for acl exists
func (r *EmqxClusterKubeClient) EnsureEmqxConfigMapForAcl(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	oldConfigMapForAcl, err := r.K8sService.GetConfigMap(emqx.GetNamespace(), emqx.GetName())

	if err != nil {
		// If no configmap for acl we need to create.
		if errors.IsNotFound(err) {
			cm := NewConfigMapForAcl(emqx, labels, ownerRefs)
			return r.K8sService.CreateConfigMap(emqx.GetNamespace(), cm)
		}
		return err
	}

	if shouldUpdateEmqxConfigMapForAcl(emqx.GetACL()["conf"], oldConfigMapForAcl.Data["acl.conf"]) {
		cm := NewConfigMapForAcl(emqx, labels, ownerRefs)
		return r.K8sService.UpdateConfigMap(emqx.GetNamespace(), cm)
	}

	return nil
}

// EnsureEmqxConfigMapForLoadedModules make sure the EMQ X configmap for loaded modules exists
func (r *EmqxClusterKubeClient) EnsureEmqxConfigMapForLoadedModules(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	oldConfigMapForLM, err := r.K8sService.GetConfigMap(emqx.GetNamespace(), emqx.GetLoadedModules()["name"])
	if err != nil {
		// If no configmap for acl we need to create.
		if errors.IsNotFound(err) {
			cm := NewConfigMapForLoadedModules(emqx, labels, ownerRefs)
			return r.K8sService.CreateConfigMap(emqx.GetNamespace(), cm)
		}
		return err
	}

	if shouldUpdateEmqxConfigMapForLM(emqx.GetLoadedModules()["conf"], oldConfigMapForLM.Data["loaded_modules"]) {
		cm := NewConfigMapForLoadedModules(emqx, labels, ownerRefs)
		return r.K8sService.UpdateConfigMap(emqx.GetNamespace(), cm)
	}

	return nil
}

// EnsureEmqxConfigMapForLoadedPlugins make sure the EMQ X configmap for loaded plugins exists
func (r *EmqxClusterKubeClient) EnsureEmqxConfigMapForLoadedPlugins(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	oldConfigMapForLP, err := r.K8sService.GetConfigMap(emqx.GetNamespace(), emqx.GetLoadedPlugins()["name"])
	if err != nil {
		// If no configmap for acl we need to create.
		if errors.IsNotFound(err) {
			cm := NewConfigMapForLoadedPlugins(emqx, labels, ownerRefs)
			return r.K8sService.CreateConfigMap(emqx.GetNamespace(), cm)
		}
		return err
	}

	if shouldUpdateEmqxConfigMapForLP(emqx.GetLoadedPlugins()["conf"], oldConfigMapForLP.Data["loaded_plugins"]) {
		cm := NewConfigMapForAcl(emqx, labels, ownerRefs)
		return r.K8sService.UpdateConfigMap(emqx.GetNamespace(), cm)
	}

	return nil
}

// EnsureEmqxStatefulSet makes sure the EMQ X statefulset exists in the desired state
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

// EnsureEmqxListenerService make sure the EMQ X service for ingress exists
func (r *EmqxClusterKubeClient) EnsureEmqxListenerService(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	listenerSvc := NewListenerSvcForCR(emqx, labels, ownerRefs)
	return r.K8sService.CreateIfNotExistsService(emqx.GetNamespace(), listenerSvc)
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

func shouldUpdateSecret(expectLiscence, emqxLiscence string) bool {
	return expectLiscence != emqxLiscence
}

func shouldUpdateEmqxConfigMapForAcl(expectEmqxACL, oldEmqxAcl string) bool {
	return expectEmqxACL != oldEmqxAcl
}

func shouldUpdateEmqxConfigMapForLM(expectEmqxLM, oldEmqxLM string) bool {
	return expectEmqxLM != oldEmqxLM
}

func shouldUpdateEmqxConfigMapForLP(expectEmqxLP, oldEmqxLP string) bool {
	return expectEmqxLP != oldEmqxLP
}
