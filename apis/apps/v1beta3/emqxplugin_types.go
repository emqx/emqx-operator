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

package v1beta3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EmqxPluginSpec defines the desired state of EmqxPlugin
type EmqxPluginSpec struct {
	// More info: https://www.emqx.io/docs/en/v4.4/advanced/plugins.html#list-of-plugins
	//+kubebuilder:validation:Required
	PluginName string `json:"pluginName,omitempty"`
	// Selector matches the labels of the EMQX
	//+kubebuilder:validation:Required
	Selector map[string]string `json:"selector,omitempty"`
	// Config defines the configurations of the EMQX plugins
	Config map[string]string `json:"config,omitempty"`
}

type phase string

const (
	EmqxPluginStatusLoaded phase = "loaded"
)

// EmqxPluginStatus defines the observed state of EmqxPlugin
type EmqxPluginStatus struct {
	Phase phase `json:"phase,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:unservedversion

// EmqxPlugin is the Schema for the emqxplugins API
type EmqxPlugin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EmqxPluginSpec   `json:"spec,omitempty"`
	Status EmqxPluginStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EmqxPluginList contains a list of EmqxPlugin
type EmqxPluginList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EmqxPlugin `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EmqxPlugin{}, &EmqxPluginList{})
}
