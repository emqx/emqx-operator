package v1beta2

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
)

//+kubebuilder:object:generate=true
type EmqxBrokerModules struct {
	Name   string `json:"name,omitempty"`
	Enable bool   `json:"enable,omitempty"`
}

//+kubebuilder:object:generate=false
type EmqxBrokerModulesList struct {
	Items []EmqxBrokerModules
}

func (list *EmqxBrokerModulesList) Default() {
	defaultModule := EmqxBrokerModules{
		Name:   "emqx_mod_acl_internal",
		Enable: true,
	}
	if list.Items == nil {
		list.Items = []EmqxBrokerModules{defaultModule}
	}
	_, index := list.Lookup(defaultModule.Name)
	if index == -1 {
		list.Items = append(list.Items, defaultModule)
	}
}

func (list *EmqxBrokerModulesList) Lookup(name string) (*EmqxBrokerModules, int) {
	for index, module := range list.Items {
		if module.Name == name {
			return &module, index
		}
	}
	return nil, -1
}

func (list *EmqxBrokerModulesList) String() string {
	var str string
	for _, module := range list.Items {
		str = fmt.Sprintf("%s{%s, %t}.\n", str, module.Name, module.Enable)
	}
	return str
}

//+kubebuilder:object:generate=true
type EmqxEnterpriseModules struct {
	Name   string `json:"name,omitempty"`
	Enable bool   `json:"enable,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Configs runtime.RawExtension `json:"configs,omitempty"`
}

//+kubebuilder:object:generate=false
type EmqxEnterpriseModulesList struct {
	Items []EmqxEnterpriseModules
}

func (list *EmqxEnterpriseModulesList) Default() {
	defaultModules := []EmqxEnterpriseModules{
		{
			Name:    "internal_cal",
			Enable:  true,
			Configs: runtime.RawExtension{Raw: []byte(`{"acl_rule_file": "/mounted/acl/acl.conf"}`)},
		},
		{
			Name:   "retainer",
			Enable: true,
			Configs: runtime.RawExtension{Raw: []byte(`{
				"expiry_interval": 0,
				"max_payload_size": "1MB",
				"max_retained_messages": 0,
				"storage_type": "ram"
			}`)},
		},
	}
	list.Append(defaultModules)
}

func (list *EmqxEnterpriseModulesList) Lookup(name string) (*EmqxEnterpriseModules, int) {
	for index, module := range list.Items {
		if module.Name == name {
			return &module, index
		}
	}
	return nil, -1
}

func (list *EmqxEnterpriseModulesList) Append(modules []EmqxEnterpriseModules) {
	for _, module := range modules {
		_, index := list.Lookup(module.Name)
		if index == -1 {
			list.Items = append(list.Items, module)
		}
	}
}

func (list *EmqxEnterpriseModulesList) Overwrite(modules []EmqxEnterpriseModules) {
	for _, module := range modules {
		_, index := list.Lookup(module.Name)
		if index == -1 {
			list.Items = append(list.Items, module)
		} else {
			list.Items[index].Enable = module.Enable
			list.Items[index].Configs = module.Configs
		}
	}
}
