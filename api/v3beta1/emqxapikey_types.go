/*
Copyright 2026.

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

package v3beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=emqxapikeys,shortName=emqxapikey
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.status==\"True\")].type"
// +kubebuilder:printcolumn:name="Name",type="string",JSONPath=".spec.name"
// +kubebuilder:printcolumn:name="Role",type="string",JSONPath=".spec.role"
// +kubebuilder:printcolumn:name="Expires At",type="date",JSONPath=".spec.expiresAt"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// EMQXAPIKey is the Schema for EMQX Dashboard API keys.
type EMQXAPIKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired state of the EMQX API key.
	Spec EMQXAPIKeySpec `json:"spec,omitempty"`
	// Current status of the EMQX API key.
	Status EMQXAPIKeyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EMQXAPIKeyList contains a list of EMQXAPIKey.
type EMQXAPIKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EMQXAPIKey `json:"items"`
}

func (a *EMQXAPIKey) SetCondition(ty string, status metav1.ConditionStatus, reason, message string) bool {
	return a.Status.SetCondition(ty, status, a.Generation, reason, message)
}

func (a *EMQXAPIKey) AttachOwnerReference(owner metav1.Object, scheme *runtime.Scheme) bool {
	for _, o := range a.GetOwnerReferences() {
		if o.UID == owner.GetUID() {
			return false
		}
	}
	_ = controllerutil.SetOwnerReference(owner, a, scheme)
	return true
}

func init() {
	SchemeBuilder.Register(&EMQXAPIKey{}, &EMQXAPIKeyList{})
}
