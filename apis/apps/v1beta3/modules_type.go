package v1beta3

import (
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
	defaultModule := EmqxBrokerModule{
		Name:   "emqx_mod_acl_internal",
		Enable: true,
	}
	if list.Items == nil {
		list.Items = []EmqxBrokerModule{defaultModule}
	}
	_, index := list.Lookup(defaultModule.Name)
	if index == -1 {
		list.Items = append(list.Items, defaultModule)
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

func (list *EmqxEnterpriseModuleList) Default() {
	defaultModules := []EmqxEnterpriseModule{
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

func (list *EmqxEnterpriseModuleList) Lookup(name string) (*EmqxEnterpriseModule, int) {
	for index, module := range list.Items {
		if module.Name == name {
			return &module, index
		}
	}
	return nil, -1
}

func (list *EmqxEnterpriseModuleList) Append(modules []EmqxEnterpriseModule) {
	for _, module := range modules {
		_, index := list.Lookup(module.Name)
		if index == -1 {
			list.Items = append(list.Items, module)
		}
	}
}

func (list *EmqxEnterpriseModuleList) Overwrite(modules []EmqxEnterpriseModule) {
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
