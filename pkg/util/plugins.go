package util

import (
	"fmt"

	"github.com/emqx/emqx-operator/apis/apps/v1beta2"
)

func StringLoadedPlugins(plugins []v1beta2.Plugin) string {
	var p string
	for _, plugin := range plugins {
		p = fmt.Sprintf("%s{%s, %t}.\n", p, plugin.Name, plugin.Enable)
	}
	return p
}
