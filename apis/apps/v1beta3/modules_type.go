package v1beta3

import (
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
)

//+kubebuilder:object:generate=true
type EmqxBrokerModule struct {
	Name   string `json:"name,omitempty"`
	Enable bool   `json:"enable,omitempty"`
}

//+kubebuilder:object:generate=false
type EmqxBrokerModuleList struct {
	Items []EmqxBrokerModule
}

func (list *EmqxBrokerModuleList) String() string {
	var str string
	for _, module := range list.Items {
		str = fmt.Sprintf("%s{%s, %t}.\n", str, module.Name, module.Enable)
	}
	return str
}

//+kubebuilder:object:generate=true
type EmqxEnterpriseModule struct {
	Name   string `json:"name,omitempty"`
	Enable bool   `json:"enable,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Configs runtime.RawExtension `json:"configs,omitempty"`
}

//+kubebuilder:object:generate=false
type EmqxEnterpriseModuleList struct {
	Items []EmqxEnterpriseModule
}

func (list *EmqxEnterpriseModuleList) String() string {
	data, _ := json.Marshal(list.Items)
	str := string(data)
	if str == "null" {
		return ""
	}
	return str
}
