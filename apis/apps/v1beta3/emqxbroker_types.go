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
	Image string `json:"image,omitempty"`
	// Image pull policy.
	// One of Always, Never, IfNotPresent.
	// Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/containers/images#updating-images
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// Username for EMQX Dashboard and API
	//+kubebuilder:default:="admin"
	Username string `json:"username,omitempty"`
	// Password for EMQX Dashboard and API
	//+kubebuilder:default:="public"
	Password string `json:"password,omitempty"`

	// See https://github.com/emqx/emqx-operator/pull/72
	ExtraVolumes []corev1.Volume `json:"extraVolumes,omitempty"`
	// See https://github.com/emqx/emqx-operator/pull/72
	ExtraVolumeMounts []corev1.VolumeMount `json:"extraVolumeMounts,omitempty"`

	// Config represents the configurations of EMQX
	// More info: https://www.emqx.io/docs/en/v4.4/configuration/configuration.html
	EmqxConfig EmqxConfig `json:"config,omitempty"`
	// Arguments to the entrypoint. The container image's CMD is used if this is not provided.
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	Args []string `json:"args,omitempty"`

	// SecurityContext defines the security options the container should be run with.
	// If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty"`
	// Compute Resources required by EMQX container.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Periodic probe of container service readiness.
	// Container will be removed from service endpoints if the probe fails.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`
	// Periodic probe of container liveness.
	// Container will be restarted if the probe fails.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`
	// StartupProbe indicates that the Pod has successfully initialized.
	// If specified, no other probes are executed until this completes successfully.
	// If this probe fails, the Pod will be restarted, just as if the livenessProbe failed.
	// This can be used to provide different probe parameters at the beginning of a Pod's lifecycle,
	// when it might take a long time to load data or warm a cache, than during steady-state operation.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	StartupProbe *corev1.Probe `json:"startupProbe,omitempty"`

	// ServiceTemplate defines a logical set of ports and a policy by which to access them
	ServiceTemplate ServiceTemplate `json:"serviceTemplate,omitempty"`
	// ACL defines ACL rules
	// More info: https://www.emqx.io/docs/en/v4.4/advanced/acl.html
	ACL []string `json:"acl,omitempty"`
	// Modules define functional modules for EMQX broker
	Modules []EmqxBrokerModule `json:"modules,omitempty"`
}

// EmqxBrokerSpec defines the desired state of EmqxBroker
type EmqxBrokerSpec struct {
	//+kubebuilder:default:=3
	Replicas *int32 `json:"replicas,omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use.
	// More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Persistent describes the common attributes of storage devices
	Persistent corev1.PersistentVolumeClaimSpec `json:"persistent,omitempty"`
	// List of environment variables to set in the container.
	Env []corev1.EnvVar `json:"env,omitempty"`

	// If specified, the pod's scheduling constraints
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// If specified, the pod's tolerations.
	ToleRations []corev1.Toleration `json:"toleRations,omitempty"`
	NodeName    string              `json:"nodeName,omitempty"`
	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// List of initialization containers belonging to the pod.
	// Init containers are executed in order prior to containers being started. If any
	// init container fails, the pod is considered to have failed and is handled according
	// to its restartPolicy. The name for an init container or normal container must be
	// unique among all containers.
	// Init containers may not have Lifecycle actions, Readiness probes, Liveness probes, or Startup probes.
	// The resourceRequirements of an init container are taken into account during scheduling
	// by finding the highest request/limit for each resource type, and then using the max of
	// of that value or the sum of the normal containers. Limits are applied to init containers
	// in a similar fashion.
	// Init containers cannot currently be added or removed.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/
	InitContainers []corev1.Container `json:"initContainers,omitempty"`
	// ExtraContainers represents extra containers to be added to the pod.
	// See https://github.com/emqx/emqx-operator/issues/252
	ExtraContainers []corev1.Container `json:"extraContainers,omitempty"`
	EmqxTemplate    EmqxBrokerTemplate `json:"emqxTemplate,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
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

func (emqx *EmqxBroker) GetAPIVersion() string {
	return emqx.APIVersion
}
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

func (emqx *EmqxBroker) GetNodeSelector() map[string]string {
	return emqx.Spec.NodeSelector
}
func (emqx *EmqxBroker) SetNodeSelector(nodeSelector map[string]string) {
	emqx.Spec.NodeSelector = nodeSelector
}

func (emqx *EmqxBroker) GetAffinity() *corev1.Affinity {
	return emqx.Spec.Affinity
}
func (emqx *EmqxBroker) SetAffinity(affinity *corev1.Affinity) {
	emqx.Spec.Affinity = affinity
}

func (emqx *EmqxBroker) GetToleRations() []corev1.Toleration { return emqx.Spec.ToleRations }
func (emqx *EmqxBroker) SetToleRations(tolerations []corev1.Toleration) {
	emqx.Spec.ToleRations = tolerations
}

func (emqx *EmqxBroker) GetExtraContainers() []corev1.Container { return emqx.Spec.ExtraContainers }
func (emqx *EmqxBroker) SetExtraContainers(containers []corev1.Container) {
	emqx.Spec.ExtraContainers = containers
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

func (emqx *EmqxBroker) GetEmqxConfig() EmqxConfig       { return emqx.Spec.EmqxTemplate.EmqxConfig }
func (emqx *EmqxBroker) SetEmqxConfig(config EmqxConfig) { emqx.Spec.EmqxTemplate.EmqxConfig = config }

func (emqx *EmqxBroker) GetEnv() []corev1.EnvVar { return emqx.Spec.Env }
func (emqx *EmqxBroker) SetEnv(env []corev1.EnvVar) {
	emqx.Spec.Env = env
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

func (emqx *EmqxBroker) GetServiceTemplate() ServiceTemplate {
	return emqx.Spec.EmqxTemplate.ServiceTemplate
}
func (emqx *EmqxBroker) SetServiceTemplate(serviceTemplate ServiceTemplate) {
	emqx.Spec.EmqxTemplate.ServiceTemplate = serviceTemplate
}

func (emqx *EmqxBroker) GetACL() []string { return emqx.Spec.EmqxTemplate.ACL }
func (emqx *EmqxBroker) SetACL(acl []string) {
	emqx.Spec.EmqxTemplate.ACL = acl
}

func (emqx *EmqxBroker) GetModules() []EmqxBrokerModule { return emqx.Spec.EmqxTemplate.Modules }
func (emqx *EmqxBroker) SetModules(modules []EmqxBrokerModule) {
	emqx.Spec.EmqxTemplate.Modules = modules
}

func (emqx *EmqxBroker) GetUsername() string { return emqx.Spec.EmqxTemplate.Username }

func (emqx *EmqxBroker) SetUsername(username string) {
	emqx.Spec.EmqxTemplate.Username = username
}

func (emqx *EmqxBroker) GetPassword() string { return emqx.Spec.EmqxTemplate.Password }

func (emqx *EmqxBroker) SetPassword(password string) {
	emqx.Spec.EmqxTemplate.Password = password
}

func (emqx *EmqxBroker) GetStatus() Status       { return emqx.Status }
func (emqx *EmqxBroker) SetStatus(status Status) { emqx.Status = status }
