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

type EmqxBlueGreenUpdate struct {
	EvacuationStrategy EvacuationStrategy `json:"evacuationStrategy,omitempty"`
}

type EvacuationStrategy struct {
	WaitTakeover  int32 `json:"waitTakeover,omitempty"`
	ConnEvictRate int32 `json:"connEvictRate,omitempty"`
	SessEvictRate int32 `json:"sessEvictRate,omitempty"`
}

// EmqxEnterpriseSpec defines the desired state of EmqxEnterprise
type EmqxEnterpriseSpec struct {
	//+kubebuilder:default:=3
	Replicas *int32 `json:"replicas,omitempty"`

	// VolumeClaimTemplates describes the common attributes of storage devices
	VolumeClaimTemplates []corev1.PersistentVolumeClaim `json:"persistent,omitempty"`

	EmqxBlueGreenUpdate *EmqxBlueGreenUpdate `json:"blueGreenUpdate,omitempty"`

	Template EmqxTemplate `json:"template,omitempty"`

	// ServiceTemplate defines a logical set of ports and a policy by which to access them
	ServiceTemplate ServiceTemplate `json:"serviceTemplate,omitempty"`
}

func (s *EmqxEnterpriseSpec) GetReplicas() *int32 {
	return s.Replicas
}

func (s *EmqxEnterpriseSpec) SetReplicas(replicas int32) {
	s.Replicas = &replicas
}

func (s *EmqxEnterpriseSpec) GetVolumeClaimTemplates() []corev1.PersistentVolumeClaim {
	return s.VolumeClaimTemplates
}

func (s *EmqxEnterpriseSpec) SetVolumeClaimTemplates(volumeClaimTemplates []corev1.PersistentVolumeClaim) {
	s.VolumeClaimTemplates = volumeClaimTemplates
}

func (s *EmqxEnterpriseSpec) GetTemplate() EmqxTemplate {
	return s.Template
}

func (s *EmqxEnterpriseSpec) SetTemplate(template EmqxTemplate) {
	s.Template = template
}

func (s *EmqxEnterpriseSpec) GetServiceTemplate() ServiceTemplate {
	return s.ServiceTemplate
}
func (s *EmqxEnterpriseSpec) SetServiceTemplate(serviceTemplate ServiceTemplate) {
	s.ServiceTemplate = serviceTemplate
}

// EmqxEnterpriseStatus defines the observed state of EmqxEnterprise
type EmqxEnterpriseStatus struct {
	// Represents the latest available observations of a EMQX current state.
	Conditions []Condition `json:"conditions,omitempty"`
	// Nodes of the EMQX cluster
	EmqxNodes []EmqxNode `json:"emqxNodes,omitempty"`
	// replicas is the number of Pods created by the EMQX Custom Resource controller.
	Replicas int32 `json:"replicas,omitempty"`
	// readyReplicas is the number of pods created for this EMQX Custom Resource with a EMQX Ready.
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	EvacuationsStatus []EmqxEvacuationStatus `json:"evacuationsStatus,omitempty"`
}

func (s *EmqxEnterpriseStatus) IsRunning() bool {
	index := indexCondition(s.Conditions, ConditionRunning)
	return index == 0 && s.Conditions[index].Status == corev1.ConditionTrue
}

func (s *EmqxEnterpriseStatus) IsInitResourceReady() bool {
	index := indexCondition(s.Conditions, ConditionInitResourceReady)
	if index == -1 {
		return false
	}
	return index == len(s.Conditions)-1 && s.Conditions[index].Status == corev1.ConditionTrue
}

func (s *EmqxEnterpriseStatus) GetReplicas() int32 {
	return s.Replicas
}

func (s *EmqxEnterpriseStatus) SetReplicas(replicas int32) {
	s.Replicas = replicas
}

func (s *EmqxEnterpriseStatus) GetReadyReplicas() int32 {
	return s.ReadyReplicas
}

func (s *EmqxEnterpriseStatus) SetReadyReplicas(readyReplicas int32) {
	s.ReadyReplicas = readyReplicas
}

func (s *EmqxEnterpriseStatus) GetEmqxNodes() []EmqxNode {
	return s.EmqxNodes
}

func (s *EmqxEnterpriseStatus) SetEmqxNodes(nodes []EmqxNode) {
	s.EmqxNodes = nodes
}

func (s *EmqxEnterpriseStatus) GetConditions() []Condition {
	return s.Conditions
}

func (s *EmqxEnterpriseStatus) AddCondition(condType ConditionType, status corev1.ConditionStatus, reason, message string) {
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

// EmqxEnterprise is the Schema for the emqxenterprises API
type EmqxEnterprise struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EmqxEnterpriseSpec   `json:"spec,omitempty"`
	Status EmqxEnterpriseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EmqxEnterpriseList contains a list of EmqxEnterprise
type EmqxEnterpriseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EmqxEnterprise `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EmqxEnterprise{}, &EmqxEnterpriseList{})
}

func (emqx *EmqxEnterprise) GetSpec() EmqxSpec {
	return &emqx.Spec
}

func (emqx *EmqxEnterprise) GetStatus() EmqxStatus {
	return &emqx.Status
}
