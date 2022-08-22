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

package v2alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ServiceTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              corev1.ServiceSpec `json:"spec,omitempty"`
}

type EMQXReplicantTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              EMQXReplicantTemplateSpec `json:"spec,omitempty"`
}

type EMQXReplicantTemplateSpec struct {
	Affinity     *corev1.Affinity    `json:"affinity,omitempty"`
	ToleRations  []corev1.Toleration `json:"toleRations,omitempty"`
	NodeName     string              `json:"nodeName,omitempty"`
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`

	//+kubebuilder:default:=3
	Replicas  *int32                      `json:"replicas,omitempty"`
	Args      []string                    `json:"args,omitempty"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	SecurityContext   *corev1.SecurityContext `json:"securityContext,omitempty"`
	InitContainers    []corev1.Container      `json:"initContainers,omitempty"`
	ExtraContainers   []corev1.Container      `json:"Containers,omitempty"`
	ExtraVolumes      []corev1.Volume         `json:"extraVolumes,omitempty"`
	ExtraVolumeMounts []corev1.VolumeMount    `json:"extraVolumeMounts,omitempty"`

	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`
	LivenessProbe  *corev1.Probe `json:"livenessProbe,omitempty"`
	StartupProbe   *corev1.Probe `json:"startupProbe,omitempty"`
}

type EMQXCoreTemplateSpec struct {
	// More than EMQXReplicantTemplateSpec
	Persistent corev1.PersistentVolumeClaimSpec `json:"persistent,omitempty"`

	Affinity     *corev1.Affinity    `json:"affinity,omitempty"`
	ToleRations  []corev1.Toleration `json:"toleRations,omitempty"`
	NodeName     string              `json:"nodeName,omitempty"`
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`

	//+kubebuilder:default:=3
	Replicas  *int32                      `json:"replicas,omitempty"`
	Args      []string                    `json:"args,omitempty"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	SecurityContext   *corev1.SecurityContext `json:"securityContext,omitempty"`
	InitContainers    []corev1.Container      `json:"initContainers,omitempty"`
	ExtraContainers   []corev1.Container      `json:"Containers,omitempty"`
	ExtraVolumes      []corev1.Volume         `json:"extraVolumes,omitempty"`
	ExtraVolumeMounts []corev1.VolumeMount    `json:"extraVolumeMounts,omitempty"`

	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`
	LivenessProbe  *corev1.Probe `json:"livenessProbe,omitempty"`
	StartupProbe   *corev1.Probe `json:"startupProbe,omitempty"`
}

type EMQXCoreTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              EMQXCoreTemplateSpec `json:"spec,omitempty"`
}

// EMQXSpec defines the desired state of EMQX
type EMQXSpec struct {
	Image            string                        `json:"image,omitempty"`
	ImagePullPolicy  corev1.PullPolicy             `json:"imagePullPolicy,omitempty"`
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	SecurityContext  *corev1.PodSecurityContext    `json:"securityContext,omitempty"`

	BootstrapConfig string `json:"bootstrapConfig,omitempty"`

	CoreTemplate      EMQXCoreTemplate      `json:"coreTemplate,omitempty"`
	ReplicantTemplate EMQXReplicantTemplate `json:"replicantTemplate,omitempty"`

	DashboardServiceTemplate corev1.Service `json:"dashboardServiceTemplate,omitempty"`
	ListenersServiceTemplate corev1.Service `json:"listenersServiceTemplate,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=emqx
//+kubebuilder:storageversion

// EMQX is the Schema for the emqxes API
type EMQX struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EMQXSpec   `json:"spec,omitempty"`
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
