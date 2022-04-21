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

package v1beta3

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type License struct {
	Data       []byte `json:"data,omitempty"`
	StringData string `json:"stringData,omitempty"`
}

type EmqxEnterpriseTemplate struct {
	//+kubebuilder:validation:Required
	Image           string            `json:"image,omitempty"`
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	ExtraVolumes      []corev1.Volume      `json:"extraVolumes,omitempty"`
	ExtraVolumeMounts []corev1.VolumeMount `json:"extraVolumeMounts,omitempty"`

	Env []corev1.EnvVar `json:"env,omitempty"`

	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty"`

	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	Listener Listener               `json:"listener,omitempty"`
	ACL      []ACL                  `json:"acl,omitempty"`
	Plugins  []Plugin               `json:"plugins,omitempty"`
	Modules  []EmqxEnterpriseModule `json:"modules,omitempty"`
	License  License                `json:"license,omitempty"`
}

// EmqxEnterpriseSpec defines the desired state of EmqxEnterprise
type EmqxEnterpriseSpec struct {
	//+kubebuilder:default:=3
	Replicas *int32 `json:"replicas,omitempty"`

	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	Persistent corev1.PersistentVolumeClaimSpec `json:"persistent,omitempty"`

	Affinity     *corev1.Affinity    `json:"affinity,omitempty"`
	ToleRations  []corev1.Toleration `json:"toleRations,omitempty"`
	NodeName     string              `json:"nodeName,omitempty"`
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`

	EmqxTemplate EmqxEnterpriseTemplate `json:"emqxTemplate,omitempty"`

	TelegrafTemplate *TelegrafTemplate `json:"telegrafTemplate,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=emqx-ee
//+kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas
//+kubebuilder:storageversion

// EmqxEnterprise is the Schema for the emqxEnterprises API
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

func (emqx *EmqxEnterprise) GetKind() string     { return emqx.Kind }
func (emqx *EmqxEnterprise) SetKind(kind string) { emqx.Kind = kind }

func (emqx *EmqxEnterprise) GetReplicas() *int32 {
	return emqx.Spec.Replicas
}
func (emqx *EmqxEnterprise) SetReplicas(replicas *int32) { emqx.Spec.Replicas = replicas }

func (emqx *EmqxEnterprise) GetImage() string      { return emqx.Spec.EmqxTemplate.Image }
func (emqx *EmqxEnterprise) SetImage(image string) { emqx.Spec.EmqxTemplate.Image = image }

func (emqx *EmqxEnterprise) GetImagePullPolicy() corev1.PullPolicy {
	return emqx.Spec.EmqxTemplate.ImagePullPolicy
}
func (emqx *EmqxEnterprise) SetImagePullPolicy(pullPolicy corev1.PullPolicy) {
	emqx.Spec.EmqxTemplate.ImagePullPolicy = pullPolicy
}

func (emqx *EmqxEnterprise) GetImagePullSecrets() []corev1.LocalObjectReference {
	return emqx.Spec.ImagePullSecrets
}
func (emqx *EmqxEnterprise) SetImagePullSecrets(imagePullSecrets []corev1.LocalObjectReference) {
	emqx.Spec.ImagePullSecrets = imagePullSecrets
}

func (emqx *EmqxEnterprise) GetResource() corev1.ResourceRequirements {
	return emqx.Spec.EmqxTemplate.Resources
}
func (emqx *EmqxEnterprise) SetResource(resource corev1.ResourceRequirements) {
	emqx.Spec.EmqxTemplate.Resources = resource
}

func (emqx *EmqxEnterprise) GetPersistent() corev1.PersistentVolumeClaimSpec {
	return emqx.Spec.Persistent
}
func (emqx *EmqxEnterprise) SetPersistent(persistent corev1.PersistentVolumeClaimSpec) {
	emqx.Spec.Persistent = persistent
}

func (emqx *EmqxEnterprise) GetNodeName() string { return emqx.Spec.NodeName }
func (emqx *EmqxEnterprise) SetNodeName(nodeName string) {
	emqx.Spec.NodeName = nodeName
}

func (emqx *EmqxEnterprise) GetNodeSelector() map[string]string { return emqx.Spec.NodeSelector }
func (emqx *EmqxEnterprise) SetNodeSelector(nodeSelector map[string]string) {
	emqx.Spec.NodeSelector = nodeSelector
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

func (emqx *EmqxEnterprise) GetExtraVolumes() []corev1.Volume {
	return emqx.Spec.EmqxTemplate.ExtraVolumes
}
func (emqx *EmqxEnterprise) GetExtraVolumeMounts() []corev1.VolumeMount {
	return emqx.Spec.EmqxTemplate.ExtraVolumeMounts
}

func (emqx *EmqxEnterprise) GetACL() []ACL { return emqx.Spec.EmqxTemplate.ACL }
func (emqx *EmqxEnterprise) SetACL(acl []ACL) {
	emqx.Spec.EmqxTemplate.ACL = acl
}

func (emqx *EmqxEnterprise) GetEnv() []corev1.EnvVar { return emqx.Spec.EmqxTemplate.Env }
func (emqx *EmqxEnterprise) SetEnv(env []corev1.EnvVar) {
	emqx.Spec.EmqxTemplate.Env = env
}

func (emqx *EmqxEnterprise) GetPlugins() []Plugin { return emqx.Spec.EmqxTemplate.Plugins }
func (emqx *EmqxEnterprise) SetPlugins(plugins []Plugin) {
	emqx.Spec.EmqxTemplate.Plugins = plugins
}

func (emqx *EmqxEnterprise) GetModules() []EmqxEnterpriseModule {
	return emqx.Spec.EmqxTemplate.Modules
}
func (emqx *EmqxEnterprise) SetModules(modules []EmqxEnterpriseModule) {
	emqx.Spec.EmqxTemplate.Modules = modules
}

func (emqx *EmqxEnterprise) GetSecurityContext() *corev1.PodSecurityContext {
	return emqx.Spec.EmqxTemplate.SecurityContext
}
func (emqx *EmqxEnterprise) SetSecurityContext(securityContext *corev1.PodSecurityContext) {
	emqx.Spec.EmqxTemplate.SecurityContext = securityContext
}

func (emqx *EmqxEnterprise) GetLicense() License {
	return emqx.Spec.EmqxTemplate.License
}
func (emqx *EmqxEnterprise) SetLicense(license License) {
	emqx.Spec.EmqxTemplate.License = license
}

func (emqx *EmqxEnterprise) GetTelegrafTemplate() *TelegrafTemplate {
	return emqx.Spec.TelegrafTemplate
}
func (emqx *EmqxEnterprise) SetTelegrafTemplate(telegrafTemplate *TelegrafTemplate) {
	emqx.Spec.TelegrafTemplate = telegrafTemplate
}
