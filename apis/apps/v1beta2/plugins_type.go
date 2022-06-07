package v1beta2

import "fmt"

//+kubebuilder:object:generate=true
type Plugin struct {
	Name   string `json:"name,omitempty"`
	Enable bool   `json:"enable,omitempty"`
}

//+kubebuilder:object:generate=false
type Plugins struct {
	Items []Plugin
}

func (p *Plugins) Default(emqx Emqx) {
	defaultPlugins := []Plugin{
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
	if p.Items == nil {
		p.Items = defaultPlugins
	} else {
		if _, index := p.Lookup("emqx_management"); index == -1 {
			p.Items = append(p.Items, Plugin{Name: "emqx_management", Enable: true})
		}
	}
	if _, ok := emqx.(*EmqxEnterprise); ok {
		if _, index := p.Lookup("emqx_modules"); index == -1 {
			p.Items = append(p.Items, Plugin{Name: "emqx_modules", Enable: true})
		}
	}

}

func (p *Plugins) Lookup(name string) (*Plugin, int) {
	for index, plugin := range p.Items {
		if plugin.Name == name {
			return &plugin, index
		}
	}
	return nil, -1
}

func (p *Plugins) String() string {
	var str string
	for _, plugin := range p.Items {
		str = fmt.Sprintf("%s{%s, %t}.\n", str, plugin.Name, plugin.Enable)
	}
	return str

}
