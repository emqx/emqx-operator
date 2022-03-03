package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

//+kubebuilder:object:generate=true
type EmqxBrokerModules struct {
	Name   string `json:"name,omitempty"`
	Enable bool   `json:"enable,omitempty"`
}

func generateEmqxBrokerModules(modules []EmqxBrokerModules) []EmqxBrokerModules {
	if modules == nil {
		return defaultEmqxBrokerModules()
	}

	contains := func(m []EmqxBrokerModules, Name string) int {
		for index, value := range m {
			if value.Name == Name {
				return index
			}
		}
		return -1
	}

	for _, value := range defaultEmqxBrokerModules() {
		r := contains(modules, value.Name)
		if r == -1 {
			modules = append(modules, value)
		}
	}
	return modules
}

func defaultEmqxBrokerModules() []EmqxBrokerModules {
	return []EmqxBrokerModules{
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

func generateEmqxEnterpriseModules(modules []EmqxEnterpriseModules) []EmqxEnterpriseModules {
	if modules == nil {
		return defaultEmqxEnterpriseModules()
	}

	contains := func(m []EmqxEnterpriseModules, Name string) int {
		for index, value := range m {
			if value.Name == Name {
				return index
			}
		}
		return -1
	}

	for _, value := range defaultEmqxEnterpriseModules() {
		r := contains(modules, value.Name)
		if r == -1 {
			modules = append(modules, value)
		}
	}
	return modules
}

func defaultEmqxEnterpriseModules() []EmqxEnterpriseModules {
	return []EmqxEnterpriseModules{
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
