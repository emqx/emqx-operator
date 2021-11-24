package util

import (
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
)

//+kubebuilder:object:generate=true
type EmqxBrokerModules struct {
	Name   string `json:"name,omitempty"`
	Enable bool   `json:"enable,omitempty"`
}

func GenerateEmqxBrokerLoadedModules(modules []EmqxBrokerModules) string {
	if modules == nil {
		modules = defaultEmqxBrokerModules()
	}
	var p string
	for _, module := range modules {
		p = fmt.Sprintf("%s{%s, %t}.\n", p, module.Name, module.Enable)
	}
	return p
}

func defaultEmqxBrokerModules() []EmqxBrokerModules {
	return []EmqxBrokerModules{
		{
			Name:   "emqx_mod_acl_internal",
			Enable: true,
		},
		{
			Name:   "emqx_mod_presence",
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

func GenerateEmqxEnterpriseLoadedModules(modules []EmqxEnterpriseModules) string {
	if modules == nil {
		modules = defaultEmqxEnterpriseModules()
	}
	data, _ := json.Marshal(modules)
	return fmt.Sprintf(string(data))
}

func defaultEmqxEnterpriseModules() []EmqxEnterpriseModules {
	return []EmqxEnterpriseModules{
		{
			Name:    "internal_cal",
			Enable:  true,
			Configs: runtime.RawExtension{Raw: []byte(`{"acl_rule_file": "etc/acl.conf"}`)},
		},
		{
			Name:    "presence",
			Enable:  true,
			Configs: runtime.RawExtension{Raw: []byte(`{"qos": 0}`)},
		},
		{
			Name:   "recon",
			Enable: true,
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
