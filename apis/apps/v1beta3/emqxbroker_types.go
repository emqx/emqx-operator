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

type EmqxBrokerTemplate struct {
	//+kubebuilder:validation:Required
	Image           string            `json:"image,omitempty"`
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	ExtraVolumes      []corev1.Volume      `json:"extraVolumes,omitempty"`
	ExtraVolumeMounts []corev1.VolumeMount `json:"extraVolumeMounts,omitempty"`

	Env  []corev1.EnvVar `json:"env,omitempty"`
	Args []string        `json:"args,omitempty"`

	SecurityContext *corev1.PodSecurityContext  `json:"securityContext,omitempty"`
	Resources       corev1.ResourceRequirements `json:"resources,omitempty"`

	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`
	LivenessProbe  *corev1.Probe `json:"livenessProbe,omitempty"`
	StartupProbe   *corev1.Probe `json:"startupProbe,omitempty"`

	Listener Listener           `json:"listener,omitempty"`
	ACL      []ACL              `json:"acl,omitempty"`
	Plugins  []Plugin           `json:"plugins,omitempty"`
	Modules  []EmqxBrokerModule `json:"modules,omitempty"`
}

// EmqxBrokerSpec defines the desired state of EmqxBroker
type EmqxBrokerSpec struct {
	//+kubebuilder:default:=3
	Replicas *int32 `json:"replicas,omitempty"`

	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	Persistent corev1.PersistentVolumeClaimSpec `json:"persistent,omitempty"`

	Affinity     *corev1.Affinity    `json:"affinity,omitempty"`
	ToleRations  []corev1.Toleration `json:"toleRations,omitempty"`
	NodeName     string              `json:"nodeName,omitempty"`
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`

	InitContainers []corev1.Container `json:"initContainers,omitempty"`
	EmqxTemplate   EmqxBrokerTemplate `json:"emqxTemplate,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=emqx
//+kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas
//+kubebuilder:storageversion

// EmqxBroker is the Schema for the emqxbrokers API
type EmqxBroker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EmqxBrokerSpec `json:"spec,omitempty"`
	Status `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EmqxBrokerList contains a list of EmqxBroker
type EmqxBrokerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EmqxBroker `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EmqxBroker{}, &EmqxBrokerList{})
}

func (emqx *EmqxBroker) GetAPIVersion() string        { return emqx.APIVersion }
func (emqx *EmqxBroker) SetAPIVersion(version string) { emqx.APIVersion = version }

func (emqx *EmqxBroker) GetKind() string     { return emqx.Kind }
func (emqx *EmqxBroker) SetKind(kind string) { emqx.Kind = kind }

func (emqx *EmqxBroker) GetReplicas() *int32 {
	return emqx.Spec.Replicas
}
func (emqx *EmqxBroker) SetReplicas(replicas *int32) { emqx.Spec.Replicas = replicas }

func (emqx *EmqxBroker) GetImagePullSecrets() []corev1.LocalObjectReference {
	return emqx.Spec.ImagePullSecrets
}
func (emqx *EmqxBroker) SetImagePullSecrets(imagePullSecrets []corev1.LocalObjectReference) {
	emqx.Spec.ImagePullSecrets = imagePullSecrets
}

func (emqx *EmqxBroker) GetPersistent() corev1.PersistentVolumeClaimSpec {
	return emqx.Spec.Persistent
}
func (emqx *EmqxBroker) SetPersistent(persistent corev1.PersistentVolumeClaimSpec) {
	emqx.Spec.Persistent = persistent
}

func (emqx *EmqxBroker) GetNodeName() string { return emqx.Spec.NodeName }
func (emqx *EmqxBroker) SetNodeName(nodeName string) {
	emqx.Spec.NodeName = nodeName
}

func (emqx *EmqxBroker) GetNodeSelector() map[string]string { return emqx.Spec.NodeSelector }
func (emqx *EmqxBroker) SetNodeSelector(nodeSelector map[string]string) {
	emqx.Spec.NodeSelector = nodeSelector
}

func (emqx *EmqxBroker) GetAffinity() *corev1.Affinity         { return emqx.Spec.Affinity }
func (emqx *EmqxBroker) SetAffinity(affinity *corev1.Affinity) { emqx.Spec.Affinity = affinity }

func (emqx *EmqxBroker) GetToleRations() []corev1.Toleration { return emqx.Spec.ToleRations }
func (emqx *EmqxBroker) SetToleRations(tolerations []corev1.Toleration) {
	emqx.Spec.ToleRations = tolerations
}

func (emqx *EmqxBroker) GetInitContainers() []corev1.Container {
	return emqx.Spec.InitContainers
}
func (emqx *EmqxBroker) SetInitContainers(containers []corev1.Container) {
	emqx.Spec.InitContainers = containers
}

func (emqx *EmqxBroker) GetImage() string      { return emqx.Spec.EmqxTemplate.Image }
func (emqx *EmqxBroker) SetImage(image string) { emqx.Spec.EmqxTemplate.Image = image }

func (emqx *EmqxBroker) GetImagePullPolicy() corev1.PullPolicy {
	return emqx.Spec.EmqxTemplate.ImagePullPolicy
}
func (emqx *EmqxBroker) SetImagePullPolicy(pullPolicy corev1.PullPolicy) {
	emqx.Spec.EmqxTemplate.ImagePullPolicy = pullPolicy
}

func (emqx *EmqxBroker) GetExtraVolumes() []corev1.Volume { return emqx.Spec.EmqxTemplate.ExtraVolumes }
func (emqx *EmqxBroker) GetExtraVolumeMounts() []corev1.VolumeMount {
	return emqx.Spec.EmqxTemplate.ExtraVolumeMounts
}

func (emqx *EmqxBroker) GetResource() corev1.ResourceRequirements {
	return emqx.Spec.EmqxTemplate.Resources
}
func (emqx *EmqxBroker) SetResource(resource corev1.ResourceRequirements) {
	emqx.Spec.EmqxTemplate.Resources = resource
}

func (emqx *EmqxBroker) GetSecurityContext() *corev1.PodSecurityContext {
	return emqx.Spec.EmqxTemplate.SecurityContext
}
func (emqx *EmqxBroker) SetSecurityContext(securityContext *corev1.PodSecurityContext) {
	emqx.Spec.EmqxTemplate.SecurityContext = securityContext
}

func (emqx *EmqxBroker) GetEnv() []corev1.EnvVar { return emqx.Spec.EmqxTemplate.Env }
func (emqx *EmqxBroker) SetEnv(env []corev1.EnvVar) {
	emqx.Spec.EmqxTemplate.Env = env
}

func (emqx *EmqxBroker) GetArgs() []string { return emqx.Spec.EmqxTemplate.Args }
func (emqx *EmqxBroker) SetArgs(args []string) {
	emqx.Spec.EmqxTemplate.Args = args
}

func (emqx *EmqxBroker) GetReadinessProbe() *corev1.Probe {
	return emqx.Spec.EmqxTemplate.ReadinessProbe
}
func (emqx *EmqxBroker) SetReadinessProbe(probe *corev1.Probe) {
	emqx.Spec.EmqxTemplate.ReadinessProbe = probe
}

func (emqx *EmqxBroker) GetLivenessProbe() *corev1.Probe {
	return emqx.Spec.EmqxTemplate.LivenessProbe
}
func (emqx *EmqxBroker) SetLivenessProbe(probe *corev1.Probe) {
	emqx.Spec.EmqxTemplate.LivenessProbe = probe
}

func (emqx *EmqxBroker) GetStartupProbe() *corev1.Probe {
	return emqx.Spec.EmqxTemplate.StartupProbe
}
func (emqx *EmqxBroker) SetStartupProbe(probe *corev1.Probe) {
	emqx.Spec.EmqxTemplate.StartupProbe = probe
}

func (emqx *EmqxBroker) GetListener() Listener { return emqx.Spec.EmqxTemplate.Listener }
func (emqx *EmqxBroker) SetListener(listener Listener) {
	emqx.Spec.EmqxTemplate.Listener = listener
}

func (emqx *EmqxBroker) GetACL() []ACL { return emqx.Spec.EmqxTemplate.ACL }
func (emqx *EmqxBroker) SetACL(acl []ACL) {
	emqx.Spec.EmqxTemplate.ACL = acl
}

func (emqx *EmqxBroker) GetPlugins() []Plugin { return emqx.Spec.EmqxTemplate.Plugins }
func (emqx *EmqxBroker) SetPlugins(plugins []Plugin) {
	emqx.Spec.EmqxTemplate.Plugins = plugins
}

func (emqx *EmqxBroker) GetModules() []EmqxBrokerModule { return emqx.Spec.EmqxTemplate.Modules }
func (emqx *EmqxBroker) SetModules(modules []EmqxBrokerModule) {
	emqx.Spec.EmqxTemplate.Modules = modules
}
