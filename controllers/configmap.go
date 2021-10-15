package controllers

import (
	"github.com/emqx/emqx-operator/api/v1alpha1"
)

type ConfigMapItem struct {
	Name string            `json:"name"`
	Data map[string]string `json:"data"`
}

func makeConfigMap(instance *v1alpha1.Emqx) []ConfigMapItem {
	var confData []ConfigMapItem
	if instance.Spec.AclConf != "" {
		confData = append(confData,
			ConfigMapItem{
				Name: instance.Name + "-" + EMQX_ACL_CONF_NAME,
				Data: map[string]string{"acl.conf": instance.Spec.AclConf},
			})
	}
	if instance.Spec.LoadedModulesConf != "" {
		confData = append(confData,
			ConfigMapItem{
				Name: instance.Name + "-" + EMQX_LOADED_MODULES_NAME,
				Data: map[string]string{"loaded_modules": instance.Spec.LoadedModulesConf},
			})
	}
	if instance.Spec.LoadedPluginConf != "" {
		confData = append(confData,
			ConfigMapItem{
				Name: instance.Name + "-" + EMQX_LOADED_PLUGINS_NAME,
				Data: map[string]string{"loaded_plugins": instance.Spec.LoadedPluginConf},
			})
	}
	return confData
}
