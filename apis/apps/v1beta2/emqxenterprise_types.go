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

package v1beta2

import (
	v1beta1 "github.com/emqx/emqx-operator/apis/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type EmqxEnterpriseTemplate struct {
	License  string                          `json:"license,omitempty"`
	Listener v1beta1.Listener                `json:"listener,omitempty"`
	ACL      []v1beta1.ACL                   `json:"acl,omitempty"`
	Plugins  []v1beta1.Plugin                `json:"plugins,omitempty"`
	Modules  []v1beta1.EmqxEnterpriseModules `json:"modules,omitempty"`
}

// EmqxEnterpriseSpec defines the desired state of EmqxEnterprise
type EmqxEnterpriseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The fields of Broker.
	//The replicas of emqx broker
	//+kubebuilder:validation:Minimum=3
	Replicas *int32 `json:"replicas,omitempty"`

	//+kubebuilder:validation:Required
	Image            string                        `json:"image,omitempty"`
	ImagePullPolicy  corev1.PullPolicy             `json:"imagePullPolicy,omitempty"`
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// The service account name which is being bind with the service
	// account of the crd instance.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	Storage corev1.PersistentVolumeClaimSpec `json:"storage,omitempty"`

	// The labels configure must be specified.
	Labels      v1beta1.Labels    `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`

	Affinity     *corev1.Affinity    `json:"affinity,omitempty"`
	ToleRations  []corev1.Toleration `json:"toleRations,omitempty"`
	NodeName     string              `json:"nodeName,omitempty"`
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`

	ExtraVolumes      []corev1.Volume      `json:"extraVolumes,omitempty"`
	ExtraVolumeMounts []corev1.VolumeMount `json:"extraVolumeMounts,omitempty"`

	Env []corev1.EnvVar `json:"env,omitempty"`

	EmqxTemplate     EmqxEnterpriseTemplate    `json:"emqxTemplate,omitempty"`
	TelegrafTemplate *v1beta1.TelegrafTemplate `json:"telegrafTemplate,omitempty"`
}

// EmqxEnterpriseStatus defines the observed state of EmqxEnterprise
type EmqxEnterpriseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	v1beta1.Status `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

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
