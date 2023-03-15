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

// EmqxBrokerSpec defines the desired state of EmqxBroker
type EmqxBrokerSpec struct {
	//+kubebuilder:default:=3
	Replicas *int32 `json:"replicas,omitempty"`

	// Persistent describes the common attributes of storage devices
	Persistent *corev1.PersistentVolumeClaimTemplate `json:"persistent,omitempty"`

	Template EmqxTemplate `json:"template,omitempty"`

	// ServiceTemplate defines a logical set of ports and a policy by which to access them
	ServiceTemplate ServiceTemplate `json:"serviceTemplate,omitempty"`
}

func (s *EmqxBrokerSpec) GetReplicas() *int32 {
	return s.Replicas
}

func (s *EmqxBrokerSpec) SetReplicas(replicas int32) {
	s.Replicas = &replicas
}

func (s *EmqxBrokerSpec) GetPersistent() *corev1.PersistentVolumeClaimTemplate {
	return s.Persistent
}

func (s *EmqxBrokerSpec) SetPersistent(persistent *corev1.PersistentVolumeClaimTemplate) {
	s.Persistent = persistent
}

func (s *EmqxBrokerSpec) GetTemplate() EmqxTemplate {
	return s.Template
}

func (s *EmqxBrokerSpec) SetTemplate(template EmqxTemplate) {
	s.Template = template
}

func (s *EmqxBrokerSpec) GetServiceTemplate() ServiceTemplate {
	return s.ServiceTemplate
}
func (s *EmqxBrokerSpec) SetServiceTemplate(serviceTemplate ServiceTemplate) {
	s.ServiceTemplate = serviceTemplate
}

// EmqxBrokerStatus defines the observed state of EmqxBroker
type EmqxBrokerStatus struct {
	// Represents the latest available observations of a EMQX current state.
	Conditions []Condition `json:"conditions,omitempty"`
	// Nodes of the EMQX cluster
	EmqxNodes []EmqxNode `json:"emqxNodes,omitempty"`
	// replicas is the number of Pods created by the EMQX Custom Resource controller.
	Replicas int32 `json:"replicas,omitempty"`
	// readyReplicas is the number of pods created for this EMQX Custom Resource with a EMQX Ready.
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	CurrentStatefulSetVersion string `json:"currentStatefulSetVersion,omitempty"`
}

func (s *EmqxBrokerStatus) GetReplicas() int32 {
	return s.Replicas
}

func (s *EmqxBrokerStatus) SetReplicas(replicas int32) {
	s.Replicas = replicas
}

func (s *EmqxBrokerStatus) GetReadyReplicas() int32 {
	return s.ReadyReplicas
}

func (s *EmqxBrokerStatus) SetReadyReplicas(readyReplicas int32) {
	s.ReadyReplicas = readyReplicas
}

func (s *EmqxBrokerStatus) GetEmqxNodes() []EmqxNode {
	return s.EmqxNodes
}

func (s *EmqxBrokerStatus) SetEmqxNodes(nodes []EmqxNode) {
	s.EmqxNodes = nodes
}

func (s *EmqxBrokerStatus) GetCurrentStatefulSetVersion() string {
	return s.CurrentStatefulSetVersion
}

func (s *EmqxBrokerStatus) SetCurrentStatefulSetVersion(version string) {
	s.CurrentStatefulSetVersion = version
}

func (s *EmqxBrokerStatus) GetConditions() []Condition {
	return s.Conditions
}

func (s *EmqxBrokerStatus) AddCondition(condType ConditionType, status corev1.ConditionStatus, reason, message string) {
	s.Conditions = addCondition(s.Conditions, Condition{
		Type:    condType,
		Status:  status,
		Reason:  reason,
		Message: message,
	})
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.status==\"True\")].type"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// EmqxBroker is the Schema for the emqxbrokers API
type EmqxBroker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EmqxBrokerSpec   `json:"spec,omitempty"`
	Status EmqxBrokerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EmqxBrokerList contains a list of EmqxBroker
type EmqxBrokerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EmqxBroker `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EmqxBroker{}, &EmqxBrokerList{})
}

func (emqx *EmqxBroker) GetSpec() EmqxSpec {
	return &emqx.Spec
}

func (emqx *EmqxBroker) GetStatus() EmqxStatus {
	return &emqx.Status
}
