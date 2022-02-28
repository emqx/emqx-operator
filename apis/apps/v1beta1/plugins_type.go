package v1beta1

//+kubebuilder:object:generate=true
type Plugin struct {
	Name   string `json:"name,omitempty"`
	Enable bool   `json:"enable,omitempty"`
}

func generatePlugins(plugins []Plugin) []Plugin {
	if plugins == nil {
		return defaultLoadedPlugins()
	}

	if containsPlugins(plugins, "emqx_management") == -1 {
		plugins = append(plugins, Plugin{Name: "emqx_management", Enable: true})
	}

	return plugins
}

func containsPlugins(plugins []Plugin, name string) int {
	for index, value := range plugins {
		if value.Name == name {
			return index
		}
	}
	return -1
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
