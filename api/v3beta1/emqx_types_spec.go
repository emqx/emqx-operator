/*
Copyright 2025-2026.

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

package v3beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// EMQXSpec defines the desired state of EMQX.
// +kubebuilder:validation:XValidation:rule="!has(self.replicantTemplate) || !has(self.replicantTemplate.spec.replicas) || self.replicantTemplate.spec.replicas == 0 || self.coreTemplate.spec.replicas >= 2",message="Core-replicant clusters require at least 2 core replicas for rolling updates."
type EMQXSpec struct {
	// EMQX container image.
	// More info: https://kubernetes.io/docs/concepts/containers/images
	Image string `json:"image"`
	// Container image pull policy.
	// One of `Always`, `Never`, `IfNotPresent`.
	// Defaults to `Always` if `:latest` tag is specified, or `IfNotPresent` otherwise.
	// More info: https://kubernetes.io/docs/concepts/containers/images#updating-images
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use.
	// More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// ServiceAccount name.
	// Managed ReplicaSets and StatefulSets are associated with the specified ServiceAccount for authentication purposes.
	// More info: https://kubernetes.io/docs/concepts/security/service-accounts
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// EMQX Configuration.
	Config Config `json:"config,omitempty"`

	// Kubernetes cluster domain.
	// +kubebuilder:default:="cluster.local"
	ClusterDomain string `json:"clusterDomain,omitempty"`

	// Number of old ReplicaSets to retain to allow rollback.
	// +kubebuilder:default:=3
	RevisionHistoryLimit int32 `json:"revisionHistoryLimit,omitempty"`

	// Cluster upgrade strategy settings.
	// +kubebuilder:default={type:RollingUpdate}
	UpdateStrategy UpdateStrategy `json:"updateStrategy,omitempty"`

	// Template for Pods running EMQX core nodes.
	// +kubebuilder:default={spec:{replicas:2,persistentVolumeClaimSpec:{accessModes:{"ReadWriteOnce"},resources:{requests:{storage:"500Mi"}}}}}
	CoreTemplate EMQXCoreTemplate `json:"coreTemplate,omitempty"`

	// Template for Pods running EMQX replicant nodes.
	ReplicantTemplate *EMQXReplicantTemplate `json:"replicantTemplate,omitempty"`

	// Template for Service exposing the EMQX Dashboard.
	// Dashboard Service always points to the set of EMQX core nodes.
	DashboardServiceTemplate *ServiceTemplate `json:"dashboardServiceTemplate,omitempty"`

	// Template for Service exposing enabled EMQX listeners.
	// Listeners Service points to the set of EMQX replicant nodes if they are enabled and exist.
	// Otherwise, it points to the set of EMQX core nodes.
	ListenersServiceTemplate *ServiceTemplate `json:"listenersServiceTemplate,omitempty"`
}

type Config struct {
	// Determines how configuration updates are applied.
	// * `Merge`: Merge the new configuration into the existing configuration.
	// * `Replace`: Replace the whole configuration.
	// +kubebuilder:validation:Enum=Merge;Replace
	// +kubebuilder:default=Merge
	Mode string `json:"mode,omitempty"`
	// EMQX configuration, in HOCON format.
	// This configuration will be supplied as `base.hocon` to the container. See respective
	// [documentation](https://docs.emqx.com/en/emqx/latest/configuration/configuration.html#base-configuration-file).
	Data string `json:"data,omitempty"`
}

type UpdateStrategy struct {
	// Determines how cluster upgrade is performed.
	// * `RollingUpdate`: Perform a rolling upgrade, updating pods gradually; core pods are
	//    always updated one at a time, updating of replicants is controlled by `replicants`
	//    strategy.
	// +kubebuilder:validation:Enum=RollingUpdate
	// +kubebuilder:default=RollingUpdate
	Type string `json:"type,omitempty"`
	// Evacuation strategy settings.
	EvacuationStrategy EvacuationStrategy `json:"evacuationStrategy,omitempty"`
	// Parameters of the rolling update for replicant ReplicaSet rollouts.
	Replicants *ReplicantsUpdateStrategy `json:"replicants,omitempty"`
}

// ReplicantsUpdateStrategy controls the pace of replicant ReplicaSet rollouts.
// Semantics mirror `apps/v1 Deployment.spec.strategy.rollingUpdate`.
// +kubebuilder:validation:XValidation:rule="!(has(self.maxUnavailable) && has(self.maxSurge) && (type(self.maxUnavailable) == int ? self.maxUnavailable == 0 : self.maxUnavailable == '0%') && (type(self.maxSurge) == int ? self.maxSurge == 0 : self.maxSurge == '0%'))",message="maxUnavailable and maxSurge cannot both be zero"
// +kubebuilder:validation:XValidation:rule="!(has(self.maxUnavailable) && (type(self.maxUnavailable) == string && self.maxUnavailable == '100%') && (!has(self.maxSurge) || (type(self.maxSurge) == int ? self.maxSurge == 0 : self.maxSurge == '0%')))",message="maxSurge must be greater than zero when maxUnavailable is 100%"
type ReplicantsUpdateStrategy struct {
	// MaxUnavailable is the maximum number of old replicant pods that may be drained (evacuating,
	// terminating, or marked for deletion) at once during a replicant ReplicaSet rollout.
	// Integers are absolute counts; strings are percentages of desired replicant replicas (e.g. "25%").
	// Defaults to 1 (serial drain).
	// +kubebuilder:validation:XIntOrString
	// +kubebuilder:validation:XValidation:rule="type(self) == int ? self >= 0 : true",message="maxUnavailable must be non-negative when specified as an integer"
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
	// MaxSurge is the number of extra replicant pods allowed above the desired replica count on the
	// new ReplicaSet during a template rollout. Integers are absolute; strings are percentages of desired replicas.
	// Defaults to 0.
	// +kubebuilder:validation:XIntOrString
	// +kubebuilder:validation:XValidation:rule="type(self) == int ? self >= 0 : true",message="maxSurge must be non-negative when specified as an integer"
	MaxSurge *intstr.IntOrString `json:"maxSurge,omitempty"`
}

type EvacuationStrategyType string

const (
	NodeEvacuationStrategy     EvacuationStrategyType = "NodeEvacuation"
	DisabledEvacuationStrategy EvacuationStrategyType = "Disabled"
)

type EvacuationStrategy struct {
	// Type of the evacuation policy.
	// +kubebuilder:validation:Enum=NodeEvacuation;Disabled
	// +kubebuilder:default=NodeEvacuation
	Type EvacuationStrategyType `json:"type"`

	// Client disconnect rate (number per second).
	// Same as `conn-evict-rate` in [EMQX Node Evacuation](https://docs.emqx.com/en/emqx/v5.10/deploy/cluster/rebalancing.html#node-evacuation).
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1000
	ConnEvictRate int32 `json:"connectionEvictionRate,omitempty"`
	// Session evacuation rate (number per second).
	// Same as `sess-evict-rate` in [EMQX Node Evacuation](https://docs.emqx.com/en/emqx/v5.10/deploy/cluster/rebalancing.html#node-evacuation).
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1000
	SessEvictRate int32 `json:"sessionEvictionRate,omitempty"`
	// Amount of time (in seconds) to wait before starting session evacuation.
	// Same as `wait-takeover` in [EMQX Node Evacuation](https://docs.emqx.com/en/emqx/v5.10/deploy/cluster/rebalancing.html#node-evacuation).
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=10
	WaitTakeover int32 `json:"waitTakeover,omitempty"`
	// Duration (in seconds) during which the node waits for the Load Balancer to remove it from the active backend node list.
	// Same as `wait-health-check` in [EMQX Node Evacuation](https://docs.emqx.com/en/emqx/v5.10/deploy/cluster/rebalancing.html#node-evacuation).
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=60
	WaitHealthCheck int32 `json:"waitHealthCheck,omitempty"`
}

type EMQXCoreTemplate struct {
	// Standard object metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the desired state of a core node.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	Spec EMQXCoreTemplateSpec `json:"spec,omitempty"`
}

type EMQXReplicantTemplate struct {
	// Standard object metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the desired state of a replicant node.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +kubebuilder:validation:XValidation:rule="has(self.minAvailable) && has(self.maxUnavailable) ? false : true",message="minAvailable cannot be set when maxUnavailable is specified. These fields are mutually exclusive in PodDisruptionBudget."
	Spec EMQXReplicantTemplateSpec `json:"spec,omitempty"`
}

type EMQXCoreTemplateSpec struct {
	// +kubebuilder:validation:XValidation:rule="has(self.minAvailable) && has(self.maxUnavailable) ? false : true",message="minAvailable cannot be set when maxUnavailable is specified. These fields are mutually exclusive in PodDisruptionBudget."
	EMQXReplicantTemplateSpec `json:",inline"`

	// PVC specification for a core node data storage.
	PersistentVolumeClaimSpec corev1.PersistentVolumeClaimSpec `json:"persistentVolumeClaimSpec,omitempty"`
}

type EMQXReplicantTemplateSpec struct {
	// Selector which must be true for the pod to fit on a node.
	// Must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/config/assign-pod-node/
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// Request to schedule this pod onto a specific node.
	// If it is non-empty, the scheduler simply schedules this pod onto that node, assuming that it fits resource requirements.
	NodeName string `json:"nodeName,omitempty"`
	// Affinity for pod assignment
	// ref: https://kubernetes.io/docs/concepts/config/assign-pod-node/#affinity-and-anti-affinity
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// Pod tolerations.
	// If specified, Pod tolerates any taint that matches the triple <key,value,effect> using the matching operator.
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// Specifies how to spread matching pods among the given topology.
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`

	// Desired number of instances.
	// In case of core nodes, each instance has a consistent identity.
	// +kubebuilder:default:=2
	// +kubebuilder:validation:Minimum=0
	Replicas *int32 `json:"replicas,omitempty"`
	// An eviction is allowed if at least "minAvailable" pods selected by
	// "selector" will still be available after the eviction, i.e. even in the
	// absence of the evicted pod.  So for example you can prevent all voluntary
	// evictions by specifying "100%".
	// +kubebuilder:validation:XIntOrString
	MinAvailable *intstr.IntOrString `json:"minAvailable,omitempty"`
	// An eviction is allowed if at most "maxUnavailable" pods selected by
	// "selector" are unavailable after the eviction, i.e. even in absence of
	// the evicted pod. For example, one can prevent all voluntary evictions
	// by specifying 0. This is a mutually exclusive setting with "minAvailable".
	// +kubebuilder:validation:XIntOrString
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
	// MinReadySeconds is the minimum time (seconds) a pod must be Ready before it counts as available.
	// For core nodes this is applied to the StatefulSet (mirrors apps/v1 StatefulSetSpec.minReadySeconds);
	// for replicants, to the ReplicaSet (mirrors apps/v1 ReplicaSetSpec.minReadySeconds).
	// Omitted or zero matches the apps/v1 default (0).
	// +kubebuilder:validation:Minimum=0
	MinReadySeconds int32 `json:"minReadySeconds,omitempty"`

	// Entrypoint array. Not executed within a shell.
	// The container image's ENTRYPOINT is used if this is not provided.
	// Variable references `$(VAR_NAME)` are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. Double `$$` are reduced
	// to a single `$`, which allows for escaping the `$(VAR_NAME)` syntax: i.e. `$$(VAR_NAME)` will
	// produce the string literal `$(VAR_NAME)`. Escaped references will never be expanded, regardless
	// of whether the variable exists or not. Cannot be updated.
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +optional
	Command []string `json:"command,omitempty"`
	// Arguments to the entrypoint.
	// The container image's CMD is used if this is not provided.
	// Variable references `$(VAR_NAME)` are expanded using the container's environment. If a variable
	// cannot be resolved, the reference in the input string will be unchanged. Double `$$` are reduced
	// to a single `$`, which allows for escaping the `$(VAR_NAME)` syntax: i.e. `$$(VAR_NAME)` will
	// produce the string literal `$(VAR_NAME)`. Escaped references will never be expanded, regardless
	// of whether the variable exists or not.
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	Args []string `json:"args,omitempty"`
	// List of ports to expose from the container.
	// Exposing a port here gives the system additional information about the network connections a
	// container uses, but is primarily informational. Not specifying a port here DOES NOT prevent that
	// port from being exposed. Any port which is listening on the default `0.0.0.0` address inside a
	// container will be accessible from the network.
	Ports []corev1.ContainerPort `json:"ports,omitempty" patchStrategy:"merge" patchMergeKey:"containerPort" protobuf:"bytes,6,rep,name=ports"`
	// List of environment variables to set in the container.
	Env []corev1.EnvVar `json:"env,omitempty"`
	// List of sources to populate environment variables from in the container.
	// The keys defined within a source must be a C_IDENTIFIER. All invalid keys
	// will be reported as an event when the container is starting. When a key exists in multiple
	// sources, the value associated with the last source will take precedence.
	// Values defined by an Env with a duplicate key will take precedence.
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty" protobuf:"bytes,19,rep,name=envFrom"`
	// Compute Resources required by this container.
	// More info: https://kubernetes.io/docs/concepts/config/manage-resources-containers/
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// Pod-level security attributes and common container settings.
	// +kubebuilder:default={runAsUser:1000,runAsGroup:1000,fsGroup:1000,fsGroupChangePolicy:Always,supplementalGroups: {1000}}
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
	// Security options the container should be run with.
	// If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/
	// +kubebuilder:default={runAsUser:1000,runAsGroup:1000,runAsNonRoot:true}
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
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/
	InitContainers []corev1.Container `json:"initContainers,omitempty"`
	// Additional containers to run alongside the main container.
	ExtraContainers []corev1.Container `json:"extraContainers,omitempty"`
	// Additional volumes to provide to a Pod.
	ExtraVolumes []corev1.Volume `json:"extraVolumes,omitempty"`
	// Specifies how additional volumes are mounted into the main container.
	ExtraVolumeMounts []corev1.VolumeMount `json:"extraVolumeMounts,omitempty"`
	// Periodic probe of container liveness.
	// Container will be restarted if the probe fails.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +kubebuilder:default={initialDelaySeconds:60,periodSeconds:30,failureThreshold:3,httpGet: {path:/status, port:"dashboard"}}
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`
	// Periodic probe of container service readiness.
	// Container will be removed from service endpoints if the probe fails.
	// Strongly advised to keep the current default: it takes into account ongoing node evacuations managed
	// by the Operator as part of scaling operations and rolling updates.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +kubebuilder:default={initialDelaySeconds:10,periodSeconds:5,timeoutSeconds:3,failureThreshold:1,httpGet:{path:"/api/v5/load_rebalance/availability_check", port:"dashboard"}}
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`
	// StartupProbe indicates that the Pod has successfully initialized.
	// If specified, no other probes are executed until this completes successfully.
	// If this probe fails, the Pod will be restarted, just as if the `livenessProbe` failed.
	// This can be used to provide different probe parameters at the beginning of a Pod's lifecycle,
	// when it might take a long time to load data or warm a cache, than during steady-state operation.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	StartupProbe *corev1.Probe `json:"startupProbe,omitempty"`
	// Actions that the management system should take in response to container lifecycle events.
	Lifecycle *corev1.Lifecycle `json:"lifecycle,omitempty" protobuf:"bytes,12,opt,name=lifecycle"`
}

type ServiceTemplate struct {
	// Specifies whether the Service should be created.
	// +kubebuilder:default:=true
	Enabled *bool `json:"enabled,omitempty"`
	// Standard object metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the desired state of a Service.
	// https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	Spec corev1.ServiceSpec `json:"spec,omitempty"`
}

func (spec *EMQXSpec) HasReplicants() bool {
	return spec.ReplicantTemplate != nil && spec.ReplicantTemplate.Spec.Replicas != nil && *spec.ReplicantTemplate.Spec.Replicas > 0
}

func (spec *EMQXSpec) IsEvacuationEnabled() bool {
	return spec.UpdateStrategy.EvacuationStrategy.Type != DisabledEvacuationStrategy
}

func (s *ServiceTemplate) IsEnabled() bool {
	return s.Enabled != nil && *s.Enabled
}

func (spec *EMQXSpec) NumCoreReplicas() int32 {
	if spec.CoreTemplate.Spec.Replicas != nil {
		return *spec.CoreTemplate.Spec.Replicas
	}
	return 1
}

func (spec *EMQXSpec) NumReplicantReplicas() int32 {
	if spec.ReplicantTemplate != nil && spec.ReplicantTemplate.Spec.Replicas != nil {
		return *spec.ReplicantTemplate.Spec.Replicas
	}
	return 0
}

func (spec *EMQXSpec) NumMaxUnavailableReplicantReplicas() int32 {
	numReplicas := int(spec.NumReplicantReplicas())
	r := spec.UpdateStrategy.Replicants
	if r == nil || r.MaxUnavailable == nil {
		return 1
	}
	v, err := intstr.GetScaledValueFromIntOrPercent(r.MaxUnavailable, numReplicas, true)
	if err != nil {
		return 1
	}
	if v < 0 {
		v = 0
	}
	return int32(v)
}

func (spec *EMQXSpec) NumMaxSurgeReplicantReplicas() int32 {
	numReplicas := int(spec.NumReplicantReplicas())
	r := spec.UpdateStrategy.Replicants
	if r == nil || r.MaxSurge == nil {
		return 0
	}
	v, err := intstr.GetScaledValueFromIntOrPercent(r.MaxSurge, numReplicas, true)
	if err != nil {
		return 0
	}
	if v < 0 {
		v = 0
	}
	return int32(v)
}
