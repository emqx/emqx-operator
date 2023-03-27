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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EmqxRebalanceSpec defines the desired state of EmqxRebalance
type EmqxRebalanceSpec struct {
	EmqxInstance      string             `json:"emqxInstance,omitempty"`
	RebalanceStrategy *RebalanceStrategy `json:"rebalanceStrategy,omitempty"`
}

type RebalanceStrategy struct {
	ConnEvictRate    *int32  `json:"connEvictRate,omitempty"`
	SessEvictRate    *int32  `json:"sessEvictRate,omitempty"`
	WaitTakeover     *int32  `json:"waitTakeover,omitempty"`
	WaitHealthCheck  *int32  `json:"waitHealthCheck,omitempty"`
	AbsConnThreshold *int32  `json:"absConnThreshold,omitempty"`
	RelConnThreshold *string `json:"relConnThreshold,omitempty"`
	AbsSessThreshold *int32  `json:"absSessThreshold,omitempty"`
	RelSessThreshold *string `json:"relSessThreshold,omitempty"`
}

// EmqxRebalanceStatus defines the observed state of EmqxRebalance
type EmqxRebalanceStatus struct {
	Conditions []Condition `json:"conditions,omitempty"`
	Phase      string      `json:"phase,omitempty"`
	Rebalances []Rebalance `json:"rebalances,omitempty"`
	StartedAt  metav1.Time `json:"startedAt,omitempty"`
	EndedAt    metav1.Time `json:"endedAt,omitempty"`
}

type Rebalance struct {
	State                  string   `json:"state,omitempty"`
	SessionEvictionRate    int32    `json:"sessionEvictionRate,omitempty"`
	Recipients             []string `json:"recipients,omitempty"`
	Node                   string   `json:"node,omitempty"`
	Donors                 []string `json:"donors,omitempty"`
	CoordinatorNode        string   `json:"coordinatorNodebalances,omitempty"`
	ConnectionEvictionRate int32    `json:"connectionEvictionRate,omitempty"`
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

func init() {
	SchemeBuilder.Register(&EmqxRebalance{}, &EmqxRebalanceList{})
}
