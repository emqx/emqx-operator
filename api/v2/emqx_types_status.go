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

package v2

import (
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EMQXStatus defines the observed state of EMQX
type EMQXStatus struct {
	// Conditions representing the current status of the EMQX Custom Resource.
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Status of each core node in the cluster.
	CoreNodes []EMQXNode `json:"coreNodes,omitempty"`
	// Summary status of the set of core nodes.
	CoreNodesStatus EMQXNodesStatus `json:"coreNodesStatus,omitempty"`

	// Status of each replicant node in the cluster.
	ReplicantNodes []EMQXNode `json:"replicantNodes,omitempty"`
	// Summary status of the set of replicant nodes.
	ReplicantNodesStatus EMQXNodesStatus `json:"replicantNodesStatus,omitempty"`

	// Status of active node evacuations in the cluster.
	NodeEvacuationsStatus []NodeEvacuationStatus `json:"nodeEvacuationsStatus,omitempty"`
	// Status of EMQX Durable Storage replication.
	DSReplication DSReplicationStatus `json:"dsReplication,omitempty"`
}

type NodeEvacuationStatus struct {
	// Evacuated node name
	// +kubebuilder:example="emqx@10.0.0.1"
	NodeName string `json:"nodeName,omitempty"`
	// Evacuation state
	// +kubebuilder:example=evicting_sessions
	State string `json:"state,omitempty"`
	// Session recipients
	// +kubebuilder:example={"emqx@10.0.0.2", "emqx@10.0.0.3"}
	SessionRecipients []string `json:"sessionRecipients,omitempty"`
	// Session eviction rate, in sessions per second.
	SessionEvictionRate int32 `json:"sessionEvictionRate,omitempty"`
	// Connection eviction rate, in connections per second.
	ConnectionEvictionRate int32 `json:"connectionEvictionRate,omitempty"`
	// Initial number of sessions on this node
	InitialSessions int32 `json:"initialSessions,omitempty"`
	// Initial number of connections to this node
	InitialConnections int32 `json:"initialConnections,omitempty"`
}

type EMQXNodesStatus struct {
	// Total number of replicas.
	Replicas int32 `json:"replicas,omitempty"`
	// Number of ready replicas.
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`
	// Current revision of the respective core or replicant set.
	CurrentRevision string `json:"currentRevision,omitempty"`
	// Number of replicas running current revision.
	CurrentReplicas int32 `json:"currentReplicas,omitempty"`
	// Update revision of the respective core or replicant set.
	// When different from the current revision, the set is being updated.
	UpdateRevision string `json:"updateRevision,omitempty"`
	// Number of replicas running update revision.
	UpdateReplicas int32 `json:"updateReplicas,omitempty"`

	CollisionCount *int32 `json:"collisionCount,omitempty"`
}

type EMQXNode struct {
	// Node name
	// +kubebuilder:example="emqx@emqx-core-557c8b7684-0.emqx-headless.default.svc.cluster.local"
	Name string `json:"name,omitempty"`
	// Corresponding pod name
	// +kubebuilder:example="emqx-core-557c8b7684-0"
	PodName string `json:"podName,omitempty"`
	// Node status
	// +kubebuilder:example=running
	Status string `json:"status,omitempty"`
	// Erlang/OTP version node is running on
	// +kubebuilder:example="27.3.4.2-2/15.2.7.1"
	OTPRelease string `json:"otpRelease,omitempty"`
	// EMQX version
	// +kubebuilder:example="5.10.1"
	Version string `json:"version,omitempty"`
	// Node role, either "core" or "replicant"
	// +kubebuilder:example=core
	Role string `json:"role,omitempty"`
	// Number of MQTT sessions
	Sessions int64 `json:"sessions,omitempty"`
	// Number of connected MQTT clients
	Connections int64 `json:"connections,omitempty"`
}

func (s EMQXStatus) FindNode(node string) *EMQXNode {
	for _, n := range s.CoreNodes {
		if n.Name == node {
			return &n
		}
	}
	for _, n := range s.ReplicantNodes {
		if n.Name == node {
			return &n
		}
	}
	return nil
}

func (s EMQXStatus) FindNodeByPodName(pod string) *EMQXNode {
	for _, n := range s.CoreNodes {
		if n.PodName == pod {
			return &n
		}
	}
	for _, n := range s.ReplicantNodes {
		if n.PodName == pod {
			return &n
		}
	}
	return nil
}

// Summary of DS replication status per database.
type DSReplicationStatus struct {
	DBs []DSDBReplicationStatus `json:"dbs,omitempty"`
}

type DSDBReplicationStatus struct {
	// Name of the database
	// +kubebuilder:example="messages"
	Name string `json:"name"`
	// Number of shards of the database
	// +kubebuilder:example=16
	NumShards int32 `json:"numShards"`
	// Total number of shard replicas
	// +kubebuilder:example=48
	NumShardReplicas int32 `json:"numShardReplicas"`
	// Total number of shard replicas belonging to lost sites
	LostShardReplicas int32 `json:"lostShardReplicas"`
	// Current number of shard ownership transitions
	NumTransitions int32 `json:"numTransitions"`
	// Minimum replication factor among database shards
	MinReplicas int32 `json:"minReplicas"`
	// Maximum replication factor among database shards
	MaxReplicas int32 `json:"maxReplicas"`
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

func (s *EMQXStatus) ResetConditions(reason string) {
	conditionTypes := []string{}
	for _, c := range s.Conditions {
		if c.Type != Initialized && c.Status == metav1.ConditionTrue {
			conditionTypes = append(conditionTypes, c.Type)
		}
	}
	for _, conditionType := range conditionTypes {
		s.SetFalseCondition(conditionType, reason)
	}
}

func (s *EMQXStatus) SetCondition(c metav1.Condition) {
	s.RemoveCondition(c.Type)
	c.LastTransitionTime = metav1.Now()
	s.Conditions = slices.Insert(s.Conditions, 0, c)
}

func (s *EMQXStatus) SetTrueCondition(conditionType string) {
	s.SetCondition(metav1.Condition{
		Type:   conditionType,
		Status: metav1.ConditionTrue,
		Reason: conditionType,
	})
}

func (s *EMQXStatus) SetFalseCondition(conditionType string, reason string) {
	s.SetCondition(metav1.Condition{
		Type:   conditionType,
		Status: metav1.ConditionFalse,
		Reason: reason,
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
	if pos != -1 {
		s.Conditions = slices.Delete(s.Conditions, pos, pos+1)
	}
}

func (s *DSReplicationStatus) IsStable() bool {
	for _, db := range s.DBs {
		if db.NumTransitions > 0 {
			return false
		}
		if db.MinReplicas != db.MaxReplicas {
			return false
		}
	}
	return true
}
