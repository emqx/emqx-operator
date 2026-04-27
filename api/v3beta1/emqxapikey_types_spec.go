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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type EMQXAPIKeyRole string

const (
	EMQXAPIKeyRoleAdministrator EMQXAPIKeyRole = "Administrator"
	EMQXAPIKeyRoleViewer        EMQXAPIKeyRole = "Viewer"
	EMQXAPIKeyRolePublisher     EMQXAPIKeyRole = "Publisher"
)

// EMQXAPIKeySpec defines the desired state of an EMQX API key.
type EMQXAPIKeySpec struct {
	// EMQXRef references the EMQX custom resource this API key belongs to.
	// The referenced EMQX must be in the same namespace as this API key.
	// +kubebuilder:validation:Required
	EMQXRef LocalEMQXReference `json:"emqxRef"`

	// SecretRef references the Secret where the generated API key and secret are stored.
	// The referenced Secret is created in the same namespace as this API key.
	// +kubebuilder:validation:Required
	SecretRef LocalSecretReference `json:"secretRef"`

	// Description is an optional note describing the API key.
	Description string `json:"description,omitempty"`

	// Role controls the API key permissions.
	// +kubebuilder:validation:Enum=Administrator;Viewer;Publisher
	// +kubebuilder:default=Administrator
	Role EMQXAPIKeyRole `json:"role,omitempty"`

	// ExpiresAt is the optional expiration timestamp for the API key.
	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`
}

// LocalEMQXReference identifies an EMQX resource in the same namespace.
type LocalEMQXReference struct {
	// Name is the name of the referenced EMQX resource.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="spec.emqxRef.name is immutable"
	Name string `json:"name"`
}

// LocalSecretReference identifies a Secret in the same namespace.
type LocalSecretReference struct {
	// Name is the name of the referenced Secret.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="spec.secretRef.name is immutable"
	Name string `json:"name"`
}
