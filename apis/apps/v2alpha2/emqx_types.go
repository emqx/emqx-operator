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

package v2alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ServiceTemplate struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec defines the behavior of a service.
	// https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	Spec corev1.ServiceSpec `json:"spec,omitempty"`
}

type EMQXReplicantTemplate struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the desired behavior of the EMQX replicant node.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	Spec EMQXReplicantTemplateSpec `json:"spec,omitempty"`
}

type EMQXReplicantTemplateSpec struct {
	// NodeSelector is a selector which must be true for the pod to fit on a node. Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// NodeName is a request to schedule this pod onto a specific node. If it is non-empty, the scheduler simply schedules this pod onto that node, assuming that it fits resource requirements.
	NodeName string `json:"nodeName,omitempty"`
	// Affinity for pod assignment
	// ref: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// If specified, the pod's tolerations.
	// The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator .
	ToleRations []corev1.Toleration `json:"toleRations,omitempty"`
	// Replicas is the desired number of replicas of the given Template.
	// These are replicas in the sense that they are instantiations of the
	// same Template, but individual replicas also have a consistent identity.
	// Defaults to 2.
	//+kubebuilder:default:=2
	Replicas *int32 `json:"replicas,omitempty"`
	// Entrypoint array. Not executed within a shell.
	// The container image's ENTRYPOINT is used if this is not provided.
	// Variable references $(VAR_NAME) are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced
	// to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will
	// produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless
	// of whether the variable exists or not. Cannot be updated.
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +optional
	Command []string `json:"command,omitempty"`
	// Arguments to the entrypoint.
	// The container image's CMD is used if this is not provided.
	// Variable references $(VAR_NAME) are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced
	// to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will
	// produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless
	// of whether the variable exists or not. Cannot be updated.
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	Args []string `json:"args,omitempty"`
	// List of ports to expose from the container. Exposing a port here gives
	// the system additional information about the network connections a
	// container uses, but is primarily informational. Not specifying a port here
	// DOES NOT prevent that port from being exposed. Any port which is
	// listening on the default "0.0.0.0" address inside a container will be
	// accessible from the network.
	// Cannot be updated.
	Ports []corev1.ContainerPort `json:"ports,omitempty" patchStrategy:"merge" patchMergeKey:"containerPort" protobuf:"bytes,6,rep,name=ports"`
	// List of environment variables to set in the container.
	// Cannot be updated.
	Env []corev1.EnvVar `json:"env,omitempty"`
	// List of sources to populate environment variables in the container.
	// The keys defined within a source must be a C_IDENTIFIER. All invalid keys
	// will be reported as an event when the container is starting. When a key exists in multiple
	// sources, the value associated with the last source will take precedence.
	// Values defined by an Env with a duplicate key will take precedence.
	// Cannot be updated.
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty" protobuf:"bytes,19,rep,name=envFrom"`
	// Compute Resources required by this container.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// SecurityContext holds pod-level security attributes and common container settings.
	//+kubebuilder:default={runAsUser:1000,runAsGroup:1000,fsGroup:1000,fsGroupChangePolicy:Always,supplementalGroups: {1000}}
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
	// SecurityContext defines the security options the container should be run with.
	// If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/
	//+kubebuilder:default={runAsUser:1000,runAsGroup:1000,runAsNonRoot:true}
	ContainerSecurityContext *corev1.SecurityContext `json:"containerSecurityContext,omitempty"`
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
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/
	InitContainers []corev1.Container `json:"initContainers,omitempty"`
	// ExtraContainers represents extra containers to be added to the pod.
	// See https://github.com/emqx/emqx-operator/issues/252
	ExtraContainers []corev1.Container `json:"extraContainers,omitempty"`
	// See https://github.com/emqx/emqx-operator/pull/72
	ExtraVolumes []corev1.Volume `json:"extraVolumes,omitempty"`
	// See https://github.com/emqx/emqx-operator/pull/72
	ExtraVolumeMounts []corev1.VolumeMount `json:"extraVolumeMounts,omitempty"`
	// Periodic probe of container liveness.
	// Container will be restarted if the probe fails.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`
	// Periodic probe of container service readiness.
	// Container will be removed from service endpoints if the probe fails.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`
	// StartupProbe indicates that the Pod has successfully initialized.
	// If specified, no other probes are executed until this completes successfully.
	// If this probe fails, the Pod will be restarted, just as if the livenessProbe failed.
	// This can be used to provide different probe parameters at the beginning of a Pod's lifecycle,
	// when it might take a long time to load data or warm a cache, than during steady-state operation.
	// This cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	StartupProbe *corev1.Probe `json:"startupProbe,omitempty"`
	// Actions that the management system should take in response to container lifecycle events.
	// Cannot be updated.
	Lifecycle *corev1.Lifecycle `json:"lifecycle,omitempty" protobuf:"bytes,12,opt,name=lifecycle"`
}

type EMQXCoreTemplateSpec struct {
	EMQXReplicantTemplateSpec `json:",inline"`

	// VolumeClaimTemplates is a list of claims that pods are allowed to reference.
	// The StatefulSet controller is responsible for mapping network identities to
	// claims in a way that maintains the identity of a pod. Every claim in
	// this list must have at least one matching (by name) volumeMount in one
	// container in the template. A claim in this list takes precedence over
	// any volumes in the template, with the same name.
	// More than EMQXReplicantTemplateSpec
	VolumeClaimTemplates corev1.PersistentVolumeClaimSpec `json:"volumeClaimTemplates,omitempty"`
}

type EMQXCoreTemplate struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the desired behavior of the EMQX core node.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	Spec EMQXCoreTemplateSpec `json:"spec,omitempty"`
}

type BootstrapAPIKey struct {
	// +kubebuilder:validation:Pattern:=`^[a-zA-Z\d_]+$`
	Key string `json:"key"`
	// +kubebuilder:validation:MinLength:=3
	// +kubebuilder:validation:MaxLength:=32
	Secret string `json:"secret"`
}

type EvacuationStrategy struct {
	//+kubebuilder:validation:Minimum=0
	WaitTakeover int32 `json:"waitTakeover,omitempty"`
	// Just work in EMQX Enterprise.
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:default=1000
	ConnEvictRate int32 `json:"connEvictRate,omitempty"`
	// Just work in EMQX Enterprise.
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:default=1000
	SessEvictRate int32 `json:"sessEvictRate,omitempty"`
}

type UpdateStrategy struct {
	//+kubebuilder:validation:Enum=Recreate
	//+kubebuilder:default=Recreate
	Type string `json:"type,omitempty"`
	// Number of seconds before evacuation connection start.
	InitialDelaySeconds int32 `json:"initialDelaySeconds,omitempty"`
	// Number of seconds before evacuation connection timeout.
	EvacuationStrategy EvacuationStrategy `json:"evacuationStrategy,omitempty"`
}

// EMQXSpec defines the desired state of EMQX
type EMQXSpec struct {
	// EMQX image name.
	// More info: https://kubernetes.io/docs/concepts/containers/images
	Image string `json:"image"`
	// Image pull policy.
	// One of Always, Never, IfNotPresent.
	// Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/containers/images#updating-images
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use.
	// More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	//+kubebuilder:default:="cluster.local"
	ClusterDomain string `json:"clusterDomain,omitempty"`

	// UpdateStrategy is the object that describes the EMQX blue-green update strategy
	//+kubebuilder:default={type:Recreate,initialDelaySeconds:30,evacuationStrategy:{waitTakeover:30,connEvictRate:1000,sessEvictRate:1000}}
	UpdateStrategy UpdateStrategy `json:"updateStrategy,omitempty"`
	// EMQX bootstrap user
	// Cannot be updated.
	BootstrapAPIKeys []BootstrapAPIKey `json:"bootstrapAPIKeys,omitempty"`
	// EMQX bootstrap config, HOCON style, like emqx.conf
	// Cannot be updated.
	BootstrapConfig string `json:"bootstrapConfig,omitempty"`

	DashboardServiceTemplate corev1.Service `json:"dashboardServiceTemplate,omitempty"`
	// ListenersServiceTemplate is the object that describes the EMQX listener service that will be created
	// If the EMQX replicant node exist, this service will selector the EMQX replicant node
	// Else this service will selector EMQX core node
	ListenersServiceTemplate corev1.Service `json:"listenersServiceTemplate,omitempty"`

	// CoreTemplate is the object that describes the EMQX core node that will be created
	CoreTemplate EMQXCoreTemplate `json:"coreTemplate,omitempty"`
	// ReplicantTemplate is the object that describes the EMQX replicant node that will be created
	ReplicantTemplate *EMQXReplicantTemplate `json:"replicantTemplate,omitempty"`
	// DashboardServiceTemplate is the object that describes the EMQX dashboard service that will be created
	// This service always selector the EMQX core node
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:shortName=emqx
//+kubebuilder:storageversion
//+kubebuilder:subresource:status
//+kubebuilder:subresource:scale:specpath=.spec.replicantTemplate.spec.replicas,statuspath=.status.replicantNodeReplicas
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.status==\"True\")].type"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// EMQX is the Schema for the emqxes API
type EMQX struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec defines the desired identities of EMQX nodes in this set.
	Spec EMQXSpec `json:"spec,omitempty"`
	// Status is the current status of EMQX nodes. This data
	// may be out of date by some window of time.
	Status EMQXStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EMQXList contains a list of EMQX
type EMQXList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EMQX `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EMQX{}, &EMQXList{})
}
