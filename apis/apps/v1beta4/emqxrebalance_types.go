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

package v1beta4

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EmqxRebalanceSpec defines the desired spec of EmqxRebalance
type EmqxRebalanceSpec struct {
	// +kubebuilder:validation:required
	InstanceName string `json:"instanceName,omitempty"`
	// +kubebuilder:validation:required
	RebalanceStrategy *RebalanceStrategy `json:"rebalanceStrategy,omitempty"`
}

type RebalanceStrategy struct {
	// ConnEvictRate represents the source node client disconnect rate per second.
	ConnEvictRate int32 `json:"connEvictRate,omitempty"`
	// SessEvictRate represents the source node session evacuation rate per second.
	SessEvictRate int32 `json:"sessEvictRate,omitempty"`
	// WaitTakeover represents the time in seconds to wait for a client to
	// reconnect to take over the session after all connections are disconnected.
	WaitTakeover int32 `json:"waitTakeover,omitempty"`
	// WaitHealthCheck represents the time (in seconds) to wait for the LB to
	// remove the source node from the list of active backend nodes. After the
	// specified waiting time is exceeded,the rebalancing task will start.
	WaitHealthCheck int32 `json:"waitHealthCheck,omitempty"`
	// AbsConnThreshold represents the absolute threshold for checking
	// connection balance.
	AbsConnThreshold int32 `json:"absConnThreshold,omitempty"`
	// RelConnThreshold represents the relative threshold for checking
	// connection balance.
	RelConnThreshold string `json:"relConnThreshold,omitempty"`
	// AbsSessThreshold represents the absolute threshold for checking session
	// connection balance.
	AbsSessThreshold int32 `json:"absSessThreshold,omitempty"`
	// RelSessThreshold represents the relative threshold for checking session
	// connection balance.
	RelSessThreshold string `json:"relSessThreshold,omitempty"`
}

// EmqxRebalanceStatus represents the current state of EmqxRebalance
type EmqxRebalanceStatus struct {
	// Conditions represents the condition of emqxrebalance
	Conditions []RebalanceCondition `json:"conditions,omitempty"`
	// Phase represents the  phase of emqxrebalance
	Phase      string      `json:"phase,omitempty"`
	Rebalances []Rebalance `json:"rebalances,omitempty"`
	// Represents the time when rebalance job start
	StartTime metav1.Time `json:"startTime,omitempty"`
	// Represents the time when the rebalance job was completed.
	CompletionTime metav1.Time `json:"completionTime,omitempty"`
}

// Rebalance defines the observed Rebalance state of EMQX
// More info: https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing
type Rebalance struct {
	// Represents the state of emqx cluster rebalancing
	State string `json:"state,omitempty"`
	// SessionEvictionRate represents the node session evacuation rate per second.
	SessionEvictionRate int32 `json:"sessionEvictionRate,omitempty"`
	// Recipients represent the target node for rebalancing
	Recipients []string `json:"recipients,omitempty"`
	// Node represents the rebalancing scheduling node
	Node string `json:"node,omitempty"`
	// Recipients represent rebalanced source nodes
	Donors []string `json:"donors,omitempty"`
	// CoordinatorNode represents the node currently undergoing rebalancing
	CoordinatorNode string `json:"coordinatorNodebalances,omitempty"`
	// ConnectionEvictionRate represents the node session evacuation rate per second.
	ConnectionEvictionRate int32 `json:"connectionEvictionRate,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// EmqxRebalance is the Schema for the emqxrebalances API
type EmqxRebalance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EmqxRebalanceSpec   `json:"spec,omitempty"`
	Status EmqxRebalanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EmqxRebalanceList contains a list of EmqxRebalance
type EmqxRebalanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EmqxRebalance `json:"items"`
}

type RebalanceCondition struct {
	// Status of rebalance condition type.
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

type RebalanceConditionType string

const (
	ConditionProcess  RebalanceConditionType = "Process"
	ConditionComplete RebalanceConditionType = "Complete"
	ConditionFailed   RebalanceConditionType = "Failed"
)

func init() {
	SchemeBuilder.Register(&EmqxRebalance{}, &EmqxRebalanceList{})
}
