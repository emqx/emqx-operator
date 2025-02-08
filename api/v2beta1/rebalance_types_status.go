/*
Copyright 2025.

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

package v2beta1

import (
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RebalanceStatus struct {
	// The latest available observations of an object's current state.
	// When Rebalance fails, the condition will have type "Failed" and status false.
	// When Rebalance is in processing, the condition will have a type "Processing" and status true.
	// When Rebalance is completed, the condition will have a type "Complete" and status true.
	Conditions []RebalanceCondition `json:"conditions,omitempty"`
	// Phase represents the phase of Rebalance.
	Phase           RebalancePhase   `json:"phase,omitempty"`
	RebalanceStates []RebalanceState `json:"rebalanceStates,omitempty"`
	// StartedTime Represents the time when rebalance job start.
	StartedTime metav1.Time `json:"startedTime,omitempty"`
	// CompletedTime Represents the time when the rebalance job was completed.
	CompletedTime metav1.Time `json:"completedTime,omitempty"`
}

type RebalanceCondition struct {
	// Status of rebalance condition type. one of Processing, Complete, Failed.
	Type RebalanceConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

// Rebalance defines the observed Rebalancing state of EMQX
type RebalanceState struct {
	// State represents the state of emqx cluster rebalancing.
	State string `json:"state,omitempty"`
	// SessionEvictionRate represents the node session evacuation rate per second.
	SessionEvictionRate int32 `json:"session_eviction_rate,omitempty"`
	// Recipients represent the target node for rebalancing.
	Recipients []string `json:"recipients,omitempty"`
	// Node represents the rebalancing scheduling node.
	Node string `json:"node,omitempty"`
	// Donors represent the source nodes for rebalancing.
	Donors []string `json:"donors,omitempty"`
	// CoordinatorNode represents the node currently undergoing rebalancing.
	CoordinatorNode string `json:"coordinator_node,omitempty"`
	// ConnectionEvictionRate represents the node session evacuation rate per second.
	ConnectionEvictionRate int32 `json:"connection_eviction_rate,omitempty"`
}

type RebalancePhase string

const (
	RebalancePhaseProcessing RebalancePhase = "Processing"
	RebalancePhaseCompleted  RebalancePhase = "Completed"
	RebalancePhaseFailed     RebalancePhase = "Failed"
)

type RebalanceConditionType string

// These are built-in conditions of a EMQX rebalancing job.
const (
	RebalanceConditionProcessing RebalanceConditionType = "Processing"
	RebalanceConditionCompleted  RebalanceConditionType = "Completed"
	RebalanceConditionFailed     RebalanceConditionType = "Failed"
)

func (s *RebalanceStatus) SetFailed(condition RebalanceCondition) error {
	if condition.Type != RebalanceConditionFailed {
		return fmt.Errorf("condition type must be %s", RebalanceConditionFailed)
	}
	s.Phase = RebalancePhaseFailed
	s.SetCondition(condition)
	return nil
}

func (s *RebalanceStatus) SetCompleted(condition RebalanceCondition) error {
	if s.Phase != RebalancePhaseProcessing {
		return fmt.Errorf("rebalance job is not in processing")
	}
	if condition.Type != RebalanceConditionCompleted {
		return fmt.Errorf("condition type must be %s", RebalanceConditionCompleted)
	}
	s.Phase = RebalancePhaseCompleted
	s.CompletedTime = metav1.Now()
	s.SetCondition(condition)
	return nil
}

func (s *RebalanceStatus) SetProcessing(condition RebalanceCondition) error {
	if s.Phase == RebalancePhaseFailed {
		return fmt.Errorf("rebalance job has been failed")
	}
	if s.Phase == RebalancePhaseCompleted {
		return fmt.Errorf("rebalance job has been completed")
	}
	if condition.Type != RebalanceConditionProcessing {
		return fmt.Errorf("condition type must be %s", RebalanceConditionProcessing)
	}
	s.Phase = RebalancePhaseProcessing
	if s.StartedTime.IsZero() {
		s.StartedTime = metav1.Now()
	}
	s.SetCondition(condition)
	return nil
}

func (s *RebalanceStatus) SetCondition(condition RebalanceCondition) {
	condition.LastUpdateTime = metav1.Now()
	condition.LastTransitionTime = metav1.Now()
	pos := getConditionIndex(s, condition.Type)
	if pos >= 0 {
		if s.Conditions[pos].Status == condition.Status {
			condition.LastTransitionTime = s.Conditions[pos].LastTransitionTime
		}
		s.Conditions[pos] = condition
	} else {
		s.Conditions = append(s.Conditions, condition)
	}
	sort.Slice(s.Conditions, func(i, j int) bool {
		return s.Conditions[j].LastUpdateTime.Before(&s.Conditions[i].LastUpdateTime)
	})
}

func getConditionIndex(status *RebalanceStatus, condType RebalanceConditionType) int {
	for i, c := range status.Conditions {
		if condType == c.Type {
			return i
		}
	}
	return -1
}
