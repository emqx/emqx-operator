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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=emqx,path=emqxes
// +kubebuilder:subresource:scale:specpath=.spec.replicantTemplate.spec.replicas,statuspath=.status.replicantNodeReplicas
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.status==\"True\")].type"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// EMQX is the Schema for the emqxes API.
type EMQX struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EMQXSpec   `json:"spec,omitempty"`
	Status EMQXStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EMQXList contains a list of EMQX.
type EMQXList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EMQX `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EMQX{}, &EMQXList{})
}
