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

package v1alpha2

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas
// EmqxEnterprise is the Schema for the emqxenterprises API
type EmqxEnterprise struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EmqxBrokerSpec `json:"spec,omitempty"`
	Status `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
// EmqxEnterpriseList contains a list of EmqxEnterprise
type EmqxEnterpriseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EmqxEnterprise `json:"items"`
}

func (emqx *EmqxEnterprise) String() string {
	return fmt.Sprintf("EmqxEnterprise instance [%s],Image [%s]",
		emqx.ObjectMeta.Name,
		emqx.Spec.Image,
	)
}

func init() {
	SchemeBuilder.Register(&EmqxEnterprise{}, &EmqxEnterpriseList{})
}
