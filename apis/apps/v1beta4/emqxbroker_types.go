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
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EmqxBrokerSpec defines the desired state of EmqxBroker
type EmqxBrokerSpec struct {
	//+kubebuilder:default:=3
	Replicas *int32 `json:"replicas,omitempty"`

	// VolumeClaimTemplates describes the common attributes of storage devices
	VolumeClaimTemplates []corev1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty"`

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

func (s *EmqxBrokerSpec) GetVolumeClaimTemplates() []corev1.PersistentVolumeClaim {
	return s.VolumeClaimTemplates
}

func (s *EmqxBrokerSpec) SetVolumeClaimTemplates(volumeClaimTemplates []corev1.PersistentVolumeClaim) {
	s.VolumeClaimTemplates = volumeClaimTemplates
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
}

func (s *EmqxBrokerStatus) IsRunning() bool {
	index := indexCondition(s.Conditions, ConditionRunning)
	return index == 0 && s.Conditions[index].Status == corev1.ConditionTrue
}

func (s *EmqxBrokerStatus) IsInitResourceReady() bool {
	index := indexCondition(s.Conditions, ConditionInitResourceReady)
	if index == -1 {
		return false
	}
	return index == len(s.Conditions)-1 && s.Conditions[index].Status == corev1.ConditionTrue
}

func (s *EmqxBrokerStatus) GetConditions() []Condition {
	return s.Conditions
}

func (s *EmqxBrokerStatus) SetCondition(c Condition) {
	now := metav1.Now()
	c.LastUpdateAt = now
	c.LastUpdateTime = now.Format(time.RFC3339)
	pos := indexCondition(s.Conditions, c.Type)
	// condition exist
	if pos >= 0 {
		s.Conditions[pos] = c
	} else { // condition not exist
		c.LastTransitionTime = now.Format(time.RFC3339)
		s.Conditions = append(s.Conditions, c)
	}

	sortConditions(s.Conditions)
}

func (s *EmqxBrokerStatus) GetEmqxNodes() []EmqxNode {
	return s.EmqxNodes
}

func (s *EmqxBrokerStatus) SetEmqxNodes(emqxNodes []EmqxNode) {
	s.EmqxNodes = emqxNodes
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

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas
//+kubebuilder:storageversion

// EmqxBroker is the Schema for the emqxbrokers API
type EmqxBroker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EmqxBrokerSpec   `json:"spec,omitempty"`
	Status EmqxBrokerStatus `json:"status,omitempty"`
}

func (e *EmqxBroker) GetSpec() EmqxSpec {
	return &e.Spec
}

func (e *EmqxBroker) GetStatus() EmqxStatus {
	return &e.Status
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
