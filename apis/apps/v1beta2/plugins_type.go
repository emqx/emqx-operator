package v1beta2

import "fmt"

//+kubebuilder:object:generate=true
type Plugin struct {
	Name   string `json:"name,omitempty"`
	Enable bool   `json:"enable,omitempty"`
}

//+kubebuilder:object:generate=false
type PluginList struct {
	Items []Plugin
}

func (list *PluginList) Default() {
	defultPlugins := []Plugin{
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
	if list.Items == nil {
		list.Items = defultPlugins
	}
	_, index := list.Lookup("emqx_management")
	if index == -1 {
		list.Items = append(list.Items, Plugin{Name: "emqx_management", Enable: true})
	}
}

func (list *PluginList) Lookup(name string) (*Plugin, int) {
	for index, plugin := range list.Items {
		if plugin.Name == name {
			return &plugin, index
		}
	}
	return nil, -1
}

func (list *PluginList) String() string {
	var str string
	for _, plugin := range list.Items {
		str = fmt.Sprintf("%s{%s, %t}.\n", str, plugin.Name, plugin.Enable)
	}
	return str

}
