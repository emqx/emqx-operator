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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EmqxSpec defines the desired state of Emqx
type EmqxSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The fields of Broker.
	//The replicas of emqx broker
	Replicas *int32 `json:"replicas,omitempty"`

	//The image name of the emqx
	Image string `json:"image,omitempty"`

	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	License string `json:"license,omitempty"`

	Storage Storage `json:"storage,omitempty"`

	//+kubebuilder:validation:Optional
	Cluster Cluster `json:"cluster,omitempty"`

	Env []corev1.EnvVar `json:"env,omitempty"`

	// //+kubebuilder:validation:Optional
	// Node Node `json:"node,omitempty"`

	// //+kubebuilder:validation:Optional
	// RPC RPC `json:"rpc,omitempty"`

	// //+kubebuilder:validation:Optional
	// Log Log `json:"log,omitempty"`

	// //+kubebuilder:validation:Optional
	// AuthAcl AuthAcl `json:"authacl,omitempty"`

	// //+kubebuilder:validation:Optional
	// Mqtt Mqtt `json:"mqtt,omitempty"`

	// //+kubebuilder:validation:Optional
	// Zone []Zone `json:"zone,omitempty"`

	// //+kubebuilder:validation:Optional
	// Listener []Listener `json:"listener,omitempty"`

	// //+kubebuilder:validation:Optional
	// Plugins Plugins `json:"plugins,omitempty"`

	// //+kubebuilder:validation:Optional
	// Broker Broker `json:"broker,omitempty"`

	// //+kubebuilder:validation:Optional
	// Monitor Monitor `json:"monitor,omitempty"`
}

// EmqxStatus defines the observed state of Emqx
type EmqxStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Emqx is the Schema for the emqxes API
type Emqx struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EmqxSpec   `json:"spec,omitempty"`
	Status EmqxStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EmqxList contains a list of Emqx
type EmqxList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Emqx `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Emqx{}, &EmqxList{})
}
