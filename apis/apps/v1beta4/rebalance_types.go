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

// RebalanceSpec represents the desired spec of Rebalance
type RebalanceSpec struct {
	// +kubebuilder:validation:required
	// InstanceName represents the name of EmqxEnterprise CR
	InstanceName string `json:"instanceName,omitempty"`
	// +kubebuilder:validation:required
	// RebalanceStrategy represents the strategy of EMQX rebalancing
	// More info: https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing
	RebalanceStrategy RebalanceStrategy `json:"rebalanceStrategy,omitempty"`
}

// RebalanceStrategy represents the strategy of EMQX rebalancing
type RebalanceStrategy struct {
	// ConnEvictRate represents the source node client disconnect rate per second.
	// same to conn-evict-rate in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)
	ConnEvictRate int32 `json:"connEvictRate,omitempty"`
	// SessEvictRate represents the source node session evacuation rate per second.
	// same to sess-evict-rate in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)
	SessEvictRate int32 `json:"sessEvictRate,omitempty"`
	// WaitTakeover represents the time in seconds to wait for a client to
	// reconnect to take over the session after all connections are disconnected.
	// same to wait-takeover in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)
	WaitTakeover int32 `json:"waitTakeover,omitempty"`
	// WaitHealthCheck represents the time (in seconds) to wait for the LB to
	// remove the source node from the list of active backend nodes. After the
	// specified waiting time is exceeded,the rebalancing task will start.
	// same to wait-health-check in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)
	WaitHealthCheck int32 `json:"waitHealthCheck,omitempty"`
	// AbsConnThreshold represents the absolute threshold for checking connection balance.
	// same to abs-conn-threshold in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)
	AbsConnThreshold int32 `json:"absConnThreshold,omitempty"`
	// RelConnThreshold represents the relative threshold for checkin connection balance.
	// same to rel-conn-threshold in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)
	// the usage of float highly discouraged, as support for them varies across languages.
	// So we define the RelConnThreshold field as string type and you not float type
	RelConnThreshold string `json:"relConnThreshold,omitempty"`
	// AbsSessThreshold represents the absolute threshold for checking session connection balance.
	// same to abs-sess-threshold in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)
	AbsSessThreshold int32 `json:"absSessThreshold,omitempty"`
	// RelSessThreshold represents the relative threshold for checking session connection balance.
	// same to rel-sess-threshold in [EMQX Rebalancing](https://docs.emqx.com/en/enterprise/v4.4/advanced/rebalancing.html#rebalancing)
	// the usage of float highly discouraged, as support for them varies across languages.
	// So we define the RelSessThreshold field as string type and you not float type
	RelSessThreshold string `json:"relSessThreshold,omitempty"`
}

// RebalanceStatus represents the current state of Rebalance
type RebalanceStatus struct {
	// The latest available observations of an object's current state.
	// When Rebalance fails, the condition will have type "Failed" and status false.
	// When Rebalance is in processing, the condition will have a type "Processing" and status true.
	// When Rebalance is completed, the condition will have a type "Complete" and status true.
	Conditions []RebalanceCondition `json:"conditions,omitempty"`
	// Phase represents the phase of Rebalance.
	Phase           string           `json:"phase,omitempty"`
	RebalanceStates []RebalanceState `json:"rebalanceStates,omitempty"`
	// StartTime Represents the time when rebalance job start.
	StartTime metav1.Time `json:"startTime,omitempty"`
	// CompletionTime Represents the time when the rebalance job was completed.
	CompletionTime metav1.Time `json:"completionTime,omitempty"`
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

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// Rebalance is the Schema for the rebalances API
type Rebalance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RebalanceSpec   `json:"spec,omitempty"`
	Status RebalanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RebalanceList contains a list of Rebalance
type RebalanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Rebalance `json:"items"`
}

// RebalanceCondition describes current state of a EMQX rebalancing job.
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

type RebalanceConditionType string

// These are built-in conditions of a EMQX rebalancing job.
const (
	RebalanceProcessing RebalanceConditionType = "Processing"
	RebalanceCompleted  RebalanceConditionType = "Completed"
	RebalanceFailed     RebalanceConditionType = "Failed"
)

func init() {
	SchemeBuilder.Register(&Rebalance{}, &RebalanceList{})
}
