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

package v2beta1

import (
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// EMQXStatus defines the observed state of EMQX
type EMQXStatus struct {
	// Represents the latest available observations of a EMQX Custom Resource current state.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	CoreNodes       []EMQXNode       `json:"coreNodes,omitempty"`
	CoreNodesStatus *EMQXNodesStatus `json:"coreNodesStatus,omitempty"`

	ReplicantNodes       []EMQXNode       `json:"replicantNodes,omitempty"`
	ReplicantNodesStatus *EMQXNodesStatus `json:"replicantNodesStatus,omitempty"`

	NodeEvacuationsStatus []NodeEvacuationStatus `json:"nodEvacuationsStatus,omitempty"`
}

type NodeEvacuationStatus struct {
	Node                   string              `json:"node,omitempty"`
	Stats                  NodeEvacuationStats `json:"stats,omitempty"`
	State                  string              `json:"state,omitempty"`
	SessionRecipients      []string            `json:"session_recipients,omitempty"`
	SessionGoal            int32               `json:"session_goal,omitempty"`
	SessionEvictionRate    int32               `json:"session_eviction_rate,omitempty"`
	ConnectionGoal         int32               `json:"connection_goal,omitempty"`
	ConnectionEvictionRate int32               `json:"connection_eviction_rate,omitempty"`
}

type NodeEvacuationStats struct {
	InitialSessions  *int32 `json:"initial_sessions,omitempty"`
	InitialConnected *int32 `json:"initial_connected,omitempty"`
	CurrentSessions  *int32 `json:"current_sessions,omitempty"`
	CurrentConnected *int32 `json:"current_connected,omitempty"`
}

type EMQXNodesStatus struct {
	Replicas      int32 `json:"replicas,omitempty"`
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	CurrentRevision string `json:"currentRevision,omitempty"`
	CurrentReplicas int32  `json:"currentReplicas,omitempty"`

	UpdateRevision string `json:"updateRevision,omitempty"`
	UpdateReplicas int32  `json:"updateReplicas,omitempty"`

	CollisionCount *int32 `json:"collisionCount,omitempty"`
}

type EMQXNode struct {
	ControllerUID types.UID `json:"controllerUID,omitempty"`
	PodUID        types.UID `json:"podUID,omitempty"`
	// EMQX node name, example: emqx@127.0.0.1
	Node string `json:"node,omitempty"`
	// EMQX node status, example: Running
	NodeStatus string `json:"node_status,omitempty"`
	// Erlang/OTP version used by EMQX, example: 24.2/12.2
	OTPRelease string `json:"otp_release,omitempty"`
	// EMQX version
	Version string `json:"version,omitempty"`
	// EMQX cluster node role, enum: "core" "replicant"
	Role string `json:"role,omitempty"`
	// EMQX cluster node edition, enum: "Opensource" "Enterprise"
	Edition string `json:"edition,omitempty"`
	// In EMQX's API of `/api/v5/nodes`, the `connections` field means the number of MQTT session count,
	Session int64 `json:"connections,omitempty"`
	// In EMQX's API of `/api/v5/nodes`, the `live_connections` field means the number of connected MQTT clients.
	// THe `live_connections` just work in EMQX 5.1 or later.
	Connections int64 `json:"live_connections,omitempty"`
	// EMQX node uptime, milliseconds
	Uptime int64 `json:"-"`
}

const (
	Initialized               string = "Initialized"
	CoreNodesProgressing      string = "CoreNodesProgressing"
	CoreNodesReady            string = "CoreNodesReady"
	ReplicantNodesProgressing string = "ReplicantNodesProgressing"
	ReplicantNodesReady       string = "ReplicantNodesReady"
	Available                 string = "Available"
	Ready                     string = "Ready"
)

func (s *EMQXStatus) SetCondition(c metav1.Condition) {
	c.LastTransitionTime = metav1.Now()
	pos, _ := s.GetCondition(c.Type)
	if pos >= 0 {
		s.Conditions[pos] = c
	} else {
		s.Conditions = append(s.Conditions, c)
	}
	sort.Slice(s.Conditions, func(i, j int) bool {
		return s.Conditions[j].LastTransitionTime.Before(&s.Conditions[i].LastTransitionTime)
	})
}

func (s *EMQXStatus) GetLastTrueCondition() *metav1.Condition {
	for i := range s.Conditions {
		c := s.Conditions[i]
		if c.Status == metav1.ConditionTrue {
			return &c
		}
	}
	return nil
}

func (s *EMQXStatus) GetCondition(conditionType string) (int, *metav1.Condition) {
	for i := range s.Conditions {
		c := s.Conditions[i]
		if c.Type == conditionType {
			return i, c.DeepCopy()
		}
	}
	return -1, nil
}

func (s *EMQXStatus) IsConditionTrue(conditionType string) bool {
	_, condition := s.GetCondition(conditionType)
	if condition == nil {
		return false
	}
	return condition.Status == metav1.ConditionTrue
}

func (s *EMQXStatus) RemoveCondition(conditionType string) {
	pos, _ := s.GetCondition(conditionType)
	if pos == -1 {
		return
	}
	s.Conditions = append(s.Conditions[:pos], s.Conditions[pos+1:]...)
}
