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
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ServiceTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              corev1.ServiceSpec `json:"spec,omitempty"`
}

type EMQXTemplateSpec struct {
	Affinity     *corev1.Affinity    `json:"affinity,omitempty"`
	ToleRations  []corev1.Toleration `json:"toleRations,omitempty"`
	NodeName     string              `json:"nodeName,omitempty"`
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`

	Replicas        *int32                      `json:"replicas,omitempty"`
	SecurityContext *corev1.PodSecurityContext  `json:"securityContext,omitempty"`
	ImagePullPolicy corev1.PullPolicy           `json:"imagePullPolicy,omitempty"`
	Image           string                      `json:"image,omitempty"`
	Args            []string                    `json:"args,omitempty"`
	Resources       corev1.ResourceRequirements `json:"resources,omitempty"`

	InitContainers []corev1.Container `json:"initContainers,omitempty"`

	ExtraContainers   []corev1.Container   `json:"Containers,omitempty"`
	ExtraVolumes      []corev1.Volume      `json:"extraVolumes,omitempty"`
	ExtraVolumeMounts []corev1.VolumeMount `json:"extraVolumeMounts,omitempty"`

	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`
	LivenessProbe  *corev1.Probe `json:"livenessProbe,omitempty"`
	StartupProbe   *corev1.Probe `json:"startupProbe,omitempty"`

	ServiceTemplate ServiceTemplate `json:"serviceTemplate,omitempty"`
}

type EMQXReplicantTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              EMQXTemplateSpec `json:"spec,omitempty"`
}

type EMQXCoreTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              EMQXTemplateSpec `json:"spec,omitempty"`
}

// EMQXSpec defines the desired state of EMQX
type EMQXSpec struct {
	ImagePullSecrets  []corev1.LocalObjectReference    `json:"imagePullSecrets,omitempty"`
	SecurityContext   *corev1.PodSecurityContext       `json:"securityContext,omitempty"`
	Persistent        corev1.PersistentVolumeClaimSpec `json:"persistent,omitempty"`
	CoreTemplate      EMQXCoreTemplate                 `json:"coreTemplate,omitempty"`
	ReplicantTemplate EMQXReplicantTemplate            `json:"replicantTemplate,omitempty"`
}

type ConditionType string

const (
	ClusterRunning ConditionType = "Running"
)

type Condition struct {
	Type               ConditionType          `json:"type"`
	Status             corev1.ConditionStatus `json:"status"`
	LastUpdateTime     string                 `json:"lastUpdateTime,omitempty"`
	LastUpdateAt       metav1.Time            `json:"-"`
	LastTransitionTime string                 `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

// EMQXStatus defines the observed state of EMQX
type EMQXStatus struct {
	Conditions []Condition `json:"conditions,omitempty"`
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

// EMQX Status
func NewCondition(condType ConditionType, status corev1.ConditionStatus, reason, message string) *Condition {
	now := metav1.Now()
	nowString := now.Format(time.RFC3339)
	return &Condition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     nowString,
		LastUpdateAt:       now,
		LastTransitionTime: nowString,
		Reason:             reason,
		Message:            message,
	}
}

func (s *EMQXStatus) SetCondition(c Condition) {
	pos, cp := getCondition(s, c.Type)
	if cp != nil &&
		cp.Status == c.Status && cp.Reason == c.Reason && cp.Message == c.Message {
		now := metav1.Now()
		nowString := now.Format(time.RFC3339)
		s.Conditions[pos].LastUpdateAt = now
		s.Conditions[pos].LastUpdateTime = nowString
		s.sortConditions(s.Conditions)
		return
	}

	if cp != nil {
		s.Conditions[pos] = c
	} else {
		s.Conditions = append(s.Conditions, c)
	}

	s.sortConditions(s.Conditions)
}

func (s *EMQXStatus) sortConditions(conditions []Condition) {
	sort.Slice(conditions, func(i, j int) bool {
		return s.Conditions[j].LastUpdateAt.Before(&s.Conditions[i].LastUpdateAt)
	})
}

func getCondition(status *EMQXStatus, t ConditionType) (int, *Condition) {
	for i, c := range status.Conditions {
		if t == c.Type {
			return i, &c
		}
	}
	return -1, nil
}
