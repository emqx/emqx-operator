package util

import (
	"fmt"

	"github.com/emqx/emqx-operator/apis/apps/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetLoadedPlugins(obj client.Object) map[string]string {
	if emqx, ok := obj.(*v1beta1.EmqxBroker); ok {
		return map[string]string{
			"name":      fmt.Sprintf("%s-%s", emqx.GetName(), "loaded-plugins"),
			"mountPath": "/opt/emqx/data/loaded_plugins",
			"subPath":   "loaded_plugins",
			"conf":      stringLoadedPlugins(emqx.Spec.Plugins),
		}
	}
	if emqx, ok := obj.(*v1beta1.EmqxEnterprise); ok {
		return map[string]string{
			"name":      fmt.Sprintf("%s-%s", emqx.GetName(), "loaded-plugins"),
			"mountPath": "/opt/emqx/data/loaded_plugins",
			"subPath":   "loaded_plugins",
			"conf":      stringLoadedPlugins(emqx.Spec.Plugins),
		}
	}
	return map[string]string{}
}

func stringLoadedPlugins(plugins []v1beta1.Plugin) string {
	var p string
	for _, plugin := range plugins {
		p = fmt.Sprintf("%s{%s, %t}.\n", p, plugin.Name, plugin.Enable)
	}
	return p
}
