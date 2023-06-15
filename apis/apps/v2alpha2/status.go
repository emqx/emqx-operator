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
	"sort"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EMQXStatus defines the observed state of EMQX
type EMQXStatus struct {
	// CurrentImage, indicates the image of the EMQX used to generate Pods in the
	CurrentImage string `json:"currentImage,omitempty"`
	// Represents the latest available observations of a EMQX Custom Resource current state.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	CoreNodeStatus      EMQXNodesStatus `json:"coreNodeStatus,omitempty"`
	ReplicantNodeStatus EMQXNodesStatus `json:"replicantNodeStatus,omitempty"`
}

type EMQXNodesStatus struct {
	// EMQX nodes info
	Nodes          []EMQXNode `json:"nodes,omitempty"`
	Replicas       int32      `json:"replicas,omitempty"`
	ReadyReplicas  int32      `json:"readyReplicas,omitempty"`
	CurrentVersion int32      `json:"currentVersion,omitempty"`
	CollisionCount *int32     `json:"collisionCount,omitempty"`
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
	// EMQX cluster node role, enum: "core" "replicant"
	Role string `json:"role,omitempty"`
	// EMQX cluster node edition, enum: "Opensource" "Enterprise"
	Edition string `json:"edition,omitempty"`
	// EMQX node uptime, milliseconds
	Uptime int64 `json:"uptime,omitempty"`
}

const (
	Initialized               string = "Initialized"
	CoreNodesProgressing      string = "CoreNodesProgressing"
	CodeNodesReady            string = "CodeNodesReady"
	ReplicantNodesProgressing string = "ReplicantNodesProgressing"
	ReplicantNodesReady       string = "ReplicantNodesReady"
	Ready                     string = "Ready"
)

const (
	// https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-readiness-gate
	PodOnServing corev1.PodConditionType = "apps.emqx.io/on-serving"
)

func (s *EMQXStatus) SetNodes(nodes []EMQXNode) {
	var coreNodes, replNodes []EMQXNode = nil, nil

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Uptime < nodes[j].Uptime
	})

	for _, node := range nodes {
		if node.Role == "core" {
			coreNodes = append(coreNodes, node)
		}
		if node.Role == "replicant" {
			replNodes = append(replNodes, node)
		}
	}
	s.CoreNodeStatus.Nodes = coreNodes
	s.ReplicantNodeStatus.Nodes = replNodes
}

func (s *EMQXStatus) SetCondition(c metav1.Condition) {
	c.LastTransitionTime = metav1.Now()
	pos, _ := s.GetCondition(c.Type)
	if pos >= 0 {
		if s.Conditions[pos].Status == c.Status && !s.Conditions[pos].LastTransitionTime.IsZero() {
			c.LastTransitionTime = s.Conditions[pos].LastTransitionTime
		}
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
