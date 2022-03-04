package v1beta2

import (
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

func (list *EmqxBrokerModulesList) Merge(modules []EmqxBrokerModules) {
	if list.Items == nil {
		list.Default()
	}
	for _, module := range modules {
		_, index := list.Lookup(module.Name)
		if index == -1 {
			list.Items = append(list.Items, module)
		} else {
			list.Items[index].Enable = module.Enable
		}
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

func (list *EmqxBrokerModulesList) Default() {
	list.Items = []EmqxBrokerModules{
		{
			Name:   "emqx_mod_acl_internal",
			Enable: true,
		},
	}
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

func (list *EmqxEnterpriseModulesList) Merge(modules []EmqxEnterpriseModules) {
	if list.Items == nil {
		list.Default()
	}
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

func (list *EmqxEnterpriseModulesList) Lookup(name string) (*EmqxEnterpriseModules, int) {
	for index, module := range list.Items {
		if module.Name == name {
			return &module, index
		}
	}
	return nil, -1
}

func (list *EmqxEnterpriseModulesList) Default() {
	list.Items = []EmqxEnterpriseModules{
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
}
