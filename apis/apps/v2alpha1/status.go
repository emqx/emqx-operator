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

type ConditionType string

const (
	ClusterCreating     ConditionType = "Creating"
	ClusterCoreUpdating ConditionType = "CoreNodesUpdating"
	ClusterCoreReady    ConditionType = "CoreNodesReady"
	ClusterRunning      ConditionType = "Running"
)

type Condition struct {
	// Status of cluster condition.
	Type ConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	// The last time this condition was updated.
	LastUpdateTime string      `json:"lastUpdateTime,omitempty"`
	LastUpdateAt   metav1.Time `json:"-"`
}

type EMQXNode struct {
	// EMQX node name, example: emqx@127.0.0.1
	Node string `json:"node,omitempty"`
	// EMQX node status, example: Running
	NodeStatus string `json:"node_status,omitempty"`
	// Erlang/OTP version used by EMQX, example: 24.2/12.2
	OTPRelease string `json:"otp_release,omitempty"`
	// EMQX version
	Version string `json:"version,omitempty"`
	// EMQX cluster node role
	Role string `json:"role,omitempty"`
}

// EMQXStatus defines the observed state of EMQX
type EMQXStatus struct {
	// CurrentImage, indicates the image of the EMQX used to generate Pods in the
	CurrentImage string `json:"currentImage,omitempty"`
	// CoreNodeReplicas is the number of EMQX core node Pods created by the EMQX controller.
	CoreNodeReplicas int32 `json:"coreNodeReplicas,omitempty"`
	// CoreNodeReadyReplicas is the number of EMQX core node Pods created for this EMQX Custom Resource with a Ready Condition.
	CoreNodeReadyReplicas int32 `json:"coreNodeReadyReplicas,omitempty"`
	// ReplicantNodeReplicas is the number of EMQX replicant node Pods created by the EMQX controller.
	ReplicantNodeReplicas int32 `json:"replicantNodeReplicas,omitempty"`
	// ReplicantNodeReadyReplicas is the number of EMQX replicant node Pods created for this EMQX Custom Resource with a Ready Condition.
	ReplicantNodeReadyReplicas int32 `json:"replicantNodeReadyReplicas,omitempty"`
	// EMQX nodes info
	EMQXNodes []EMQXNode `json:"emqxNodes,omitempty"`
	// Represents the latest available observations of a EMQX Custom Resource current state.
	Conditions []Condition `json:"conditions,omitempty"`
}

// EMQX Status
func NewCondition(condType ConditionType, status corev1.ConditionStatus, reason, message string) *Condition {
	return &Condition{
		Type:    condType,
		Status:  status,
		Reason:  reason,
		Message: message,
	}
}

func (s *EMQXStatus) IsCreating() bool {
	index := indexCondition(s, ClusterCreating)
	return index == 0 && s.Conditions[index].Status == corev1.ConditionTrue
}

func (s *EMQXStatus) IsCoreNodesUpdating() bool {
	index := indexCondition(s, ClusterCoreUpdating)
	return index == 0 && s.Conditions[index].Status == corev1.ConditionTrue
}

func (s *EMQXStatus) IsCoreNodesReady() bool {
	index := indexCondition(s, ClusterCoreReady)
	return index == 0 && s.Conditions[index].Status == corev1.ConditionTrue
}

func (s *EMQXStatus) IsRunning() bool {
	index := indexCondition(s, ClusterRunning)
	return index == 0 && s.Conditions[index].Status == corev1.ConditionTrue
}

func (s *EMQXStatus) SetCondition(c Condition) {
	now := metav1.Now()
	c.LastUpdateAt = now
	c.LastUpdateTime = now.Format(time.RFC3339)
	c.LastTransitionTime = now.Format(time.RFC3339)
	pos := indexCondition(s, c.Type)
	if pos >= 0 {
		if s.Conditions[pos].Status == c.Status && s.Conditions[pos].LastTransitionTime != "" {
			c.LastTransitionTime = s.Conditions[pos].LastTransitionTime
		}
		s.Conditions[pos] = c
	} else {
		s.Conditions = append(s.Conditions, c)
	}
	s.sortConditions(s.Conditions)
}

func (s *EMQXStatus) RemoveCondition(t ConditionType) {
	pos := indexCondition(s, t)
	if pos == -1 {
		return
	}
	s.Conditions = append(s.Conditions[:pos], s.Conditions[pos+1:]...)
}

func (s *EMQXStatus) sortConditions(conditions []Condition) {
	sort.Slice(conditions, func(i, j int) bool {
		return s.Conditions[j].LastUpdateAt.Before(&s.Conditions[i].LastUpdateAt)
	})
}

func indexCondition(status *EMQXStatus, t ConditionType) int {
	for i, c := range status.Conditions {
		if t == c.Type {
			return i
		}
	}
	return -1
}
