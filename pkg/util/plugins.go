package util

import (
	"fmt"
)

//+kubebuilder:object:generate=true
type Plugin struct {
	Name   string `json:"name,omitempty"`
	Enable bool   `json:"enable,omitempty"`
}

func GenerateLoadedPlugins(plugins []Plugin) string {
	if plugins == nil {
		plugins = defaultLoadedPlugins()
	}
	var p string
	for _, plugin := range plugins {
		p = fmt.Sprintf("%s{%s, %t}.\n", p, plugin.Name, plugin.Enable)
	}
	return p
}

func defaultLoadedPlugins() []Plugin {
	return []Plugin{
		{
			Name:   "emqx_management",
			Enable: true,
		},
		{
			Name:   "emqx_recon",
			Enable: true,
		},
		{
			Name:   "emqx_retainer",
			Enable: true,
		},
		{
			Name:   "emqx_dashboard",
			Enable: true,
		},
		{
			Name:   "emqx_telemetry",
			Enable: true,
		},
		{
			Name:   "emqx_rule_engine",
			Enable: true,
		},
	}
}
