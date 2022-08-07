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

func (list *EmqxBrokerModuleList) Default() {
	defaultModules := []EmqxBrokerModule{
		{
			Name:   "emqx_mod_acl_internal",
			Enable: true,
		},
		{
			Name:   "emqx_mod_presence",
			Enable: true,
		},
	}
	if list.Items == nil {
		list.Items = defaultModules
	}
	for _, defaultModule := range defaultModules {
		_, index := list.Lookup(defaultModule.Name)
		if index == -1 {
			list.Items = append(list.Items, defaultModule)
		}
	}
}

func (list *EmqxBrokerModuleList) Lookup(name string) (*EmqxBrokerModule, int) {
	for index, module := range list.Items {
		if module.Name == name {
			return &module, index
		}
	}
	return nil, -1
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
