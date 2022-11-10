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

// EmqxEnterpriseStatus defines the observed state of EmqxEnterprise
type EmqxEnterpriseStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas
//+kubebuilder:storageversion

// EmqxEnterprise is the Schema for the emqxenterprises API
type EmqxEnterprise struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EmqxEnterpriseSpec `json:"spec,omitempty"`
	Status `json:"status,omitempty"`
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

func (emqx *EmqxEnterprise) GetReplicas() *int32 {
	return emqx.Spec.Replicas
}

func (emqx *EmqxEnterprise) SetReplicas(replicas int32) {
	emqx.Spec.Replicas = &replicas
}

func (emqx *EmqxEnterprise) GetVolumeClaimTemplates() []corev1.PersistentVolumeClaim {
	return emqx.Spec.VolumeClaimTemplates
}

func (emqx *EmqxEnterprise) SetVolumeClaimTemplates(volumeClaimTemplates []corev1.PersistentVolumeClaim) {
	emqx.Spec.VolumeClaimTemplates = volumeClaimTemplates
}

func (emqx *EmqxEnterprise) GetTemplate() EmqxTemplate {
	return emqx.Spec.Template
}

func (emqx *EmqxEnterprise) SetTemplate(template EmqxTemplate) {
	emqx.Spec.Template = template
}

func (emqx *EmqxEnterprise) GetServiceTemplate() ServiceTemplate {
	return emqx.Spec.ServiceTemplate
}
func (emqx *EmqxEnterprise) SetServiceTemplate(serviceTemplate ServiceTemplate) {
	emqx.Spec.ServiceTemplate = serviceTemplate
}

func (emqx *EmqxEnterprise) GetStatus() Status {
	return emqx.Status
}
func (emqx *EmqxEnterprise) SetStatus(status Status) {
	emqx.Status = status
}
