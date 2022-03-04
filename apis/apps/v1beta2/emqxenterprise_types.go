/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta2

import (
	"reflect"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type EmqxEnterpriseTemplate struct {
	License  string                         `json:"license,omitempty"`
	Listener Listener                       `json:"listener,omitempty"`
	ACL      []v1beta3.ACL                  `json:"acl,omitempty"`
	Plugins  []v1beta3.Plugin               `json:"plugins,omitempty"`
	Modules  []v1beta3.EmqxEnterpriseModule `json:"modules,omitempty"`
}

// EmqxEnterpriseSpec defines the desired state of EmqxEnterprise
type EmqxEnterpriseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The fields of Broker.
	//The replicas of emqx broker
	//+kubebuilder:validation:Minimum=3
	Replicas *int32 `json:"replicas,omitempty"`

	//+kubebuilder:validation:Required
	Image            string                        `json:"image,omitempty"`
	ImagePullPolicy  corev1.PullPolicy             `json:"imagePullPolicy,omitempty"`
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// The service account name which is being bind with the service
	// account of the crd instance.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	Storage corev1.PersistentVolumeClaimSpec `json:"storage,omitempty"`

	// The labels configure must be specified.
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`

	Affinity     *corev1.Affinity    `json:"affinity,omitempty"`
	ToleRations  []corev1.Toleration `json:"toleRations,omitempty"`
	NodeName     string              `json:"nodeName,omitempty"`
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`

	ExtraVolumes      []corev1.Volume      `json:"extraVolumes,omitempty"`
	ExtraVolumeMounts []corev1.VolumeMount `json:"extraVolumeMounts,omitempty"`

	Env []corev1.EnvVar `json:"env,omitempty"`

	EmqxTemplate     EmqxEnterpriseTemplate    `json:"emqxTemplate,omitempty"`
	TelegrafTemplate *v1beta3.TelegrafTemplate `json:"telegrafTemplate,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:shortName=emqx-ee
//+kubebuilder:subresource:status
//+kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas

// EmqxEnterprise is the Schema for the emqxenterprises API
type EmqxEnterprise struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EmqxEnterpriseSpec `json:"spec,omitempty"`
	Status `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EmqxEnterpriseList contains a list of EmqxEnterprise
type EmqxEnterpriseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EmqxEnterprise `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EmqxEnterprise{}, &EmqxEnterpriseList{})
}

func (emqx *EmqxEnterprise) GetAPIVersion() string        { return emqx.APIVersion }
func (emqx *EmqxEnterprise) SetAPIVersion(version string) { emqx.APIVersion = version }
func (emqx *EmqxEnterprise) GetKind() string              { return emqx.Kind }
func (emqx *EmqxEnterprise) SetKind(kind string)          { emqx.Kind = kind }

func (emqx *EmqxEnterprise) GetReplicas() *int32 {
	if reflect.ValueOf(emqx.Spec.Replicas).IsZero() {
		defaultReplicas := int32(3)
		emqx.SetReplicas(&defaultReplicas)
	}
	return emqx.Spec.Replicas
}
func (emqx *EmqxEnterprise) SetReplicas(replicas *int32) { emqx.Spec.Replicas = replicas }

func (emqx *EmqxEnterprise) GetImage() string      { return emqx.Spec.Image }
func (emqx *EmqxEnterprise) SetImage(image string) { emqx.Spec.Image = image }

func (emqx *EmqxEnterprise) GetImagePullPolicy() corev1.PullPolicy { return emqx.Spec.ImagePullPolicy }
func (emqx *EmqxEnterprise) SetImagePullPolicy(pullPolicy corev1.PullPolicy) {
	emqx.Spec.ImagePullPolicy = pullPolicy
}

func (emqx *EmqxEnterprise) GetImagePullSecrets() []corev1.LocalObjectReference {
	return emqx.Spec.ImagePullSecrets
}
func (emqx *EmqxEnterprise) SetImagePullSecrets(imagePullSecrets []corev1.LocalObjectReference) {
	emqx.Spec.ImagePullSecrets = imagePullSecrets
}

func (emqx *EmqxEnterprise) GetServiceAccountName() string {
	if emqx.Spec.ServiceAccountName == "" {
		emqx.SetServiceAccountName(emqx.Name)
	}
	return emqx.Spec.ServiceAccountName
}
func (emqx *EmqxEnterprise) SetServiceAccountName(serviceAccountName string) {
	emqx.Spec.ServiceAccountName = serviceAccountName
}

func (emqx *EmqxEnterprise) GetResource() corev1.ResourceRequirements { return emqx.Spec.Resources }
func (emqx *EmqxEnterprise) SetResource(resource corev1.ResourceRequirements) {
	emqx.Spec.Resources = resource
}

func (emqx *EmqxEnterprise) GetLicense() string        { return emqx.Spec.EmqxTemplate.License }
func (emqx *EmqxEnterprise) SetLicense(license string) { emqx.Spec.EmqxTemplate.License = license }

func (emqx *EmqxEnterprise) GetStorage() corev1.PersistentVolumeClaimSpec { return emqx.Spec.Storage }
func (emqx *EmqxEnterprise) SetStorage(storage corev1.PersistentVolumeClaimSpec) {
	emqx.Spec.Storage = storage
}

func (emqx *EmqxEnterprise) GetNodeName() string { return emqx.Spec.NodeName }
func (emqx *EmqxEnterprise) SetNodeName(nodeName string) {
	emqx.Spec.NodeName = nodeName
}

func (emqx *EmqxEnterprise) GetNodeSelector() map[string]string { return emqx.Spec.NodeSelector }
func (emqx *EmqxEnterprise) SetNodeSelector(nodeSelector map[string]string) {
	emqx.Spec.NodeSelector = nodeSelector
}

func (emqx *EmqxEnterprise) GetAnnotations() map[string]string { return emqx.Spec.Annotations }
func (emqx *EmqxEnterprise) SetAnnotations(annotations map[string]string) {
	emqx.Spec.Annotations = annotations
}

func (emqx *EmqxEnterprise) GetListener() Listener { return emqx.Spec.EmqxTemplate.Listener }
func (emqx *EmqxEnterprise) SetListener(listener Listener) {
	emqx.Spec.EmqxTemplate.Listener = listener
}

func (emqx *EmqxEnterprise) GetAffinity() *corev1.Affinity         { return emqx.Spec.Affinity }
func (emqx *EmqxEnterprise) SetAffinity(affinity *corev1.Affinity) { emqx.Spec.Affinity = affinity }

func (emqx *EmqxEnterprise) GetToleRations() []corev1.Toleration { return emqx.Spec.ToleRations }
func (emqx *EmqxEnterprise) SetToleRations(tolerations []corev1.Toleration) {
	emqx.Spec.ToleRations = tolerations
}

func (emqx *EmqxEnterprise) GetExtraVolumes() []corev1.Volume { return emqx.Spec.ExtraVolumes }
func (emqx *EmqxEnterprise) GetExtraVolumeMounts() []corev1.VolumeMount {
	return emqx.Spec.ExtraVolumeMounts
}

func (emqx *EmqxEnterprise) GetACL() []v1beta3.ACL { return emqx.Spec.EmqxTemplate.ACL }
func (emqx *EmqxEnterprise) SetACL(acl []v1beta3.ACL) {
	emqx.Spec.EmqxTemplate.ACL = acl
}

func (emqx *EmqxEnterprise) GetEnv() []corev1.EnvVar { return emqx.Spec.Env }
func (emqx *EmqxEnterprise) SetEnv(env []corev1.EnvVar) {
	emqx.Spec.Env = env
}

func (emqx *EmqxEnterprise) GetPlugins() []v1beta3.Plugin { return emqx.Spec.EmqxTemplate.Plugins }
func (emqx *EmqxEnterprise) SetPlugins(plugins []v1beta3.Plugin) {
	emqx.Spec.EmqxTemplate.Plugins = plugins
}

func (emqx *EmqxEnterprise) GetModules() []v1beta3.EmqxEnterpriseModule {
	return emqx.Spec.EmqxTemplate.Modules
}
func (emqx *EmqxEnterprise) SetModules(modules []v1beta3.EmqxEnterpriseModule) {
	emqx.Spec.EmqxTemplate.Modules = modules
}

func (emqx *EmqxEnterprise) GetTelegrafTemplate() *v1beta3.TelegrafTemplate {
	return emqx.Spec.TelegrafTemplate
}
func (emqx *EmqxEnterprise) SetTelegrafTemplate(telegrafTemplate *v1beta3.TelegrafTemplate) {
	emqx.Spec.TelegrafTemplate = telegrafTemplate
}
