package util

import (
	"encoding/json"
	"fmt"

	"github.com/emqx/emqx-operator/apis/apps/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetLoadedModules(obj client.Object) map[string]string {
	if emqx, ok := obj.(*v1beta1.EmqxBroker); ok {
		return map[string]string{
			"name":      fmt.Sprintf("%s-%s", emqx.GetName(), "loaded-modules"),
			"mountPath": "/opt/emqx/data/loaded_modules",
			"subPath":   "loaded_modules",
			"conf":      stringEmqxBrokerLoadedModules(emqx.Spec.Modules),
		}
	}
	if emqx, ok := obj.(*v1beta1.EmqxEnterprise); ok {
		data, _ := json.Marshal(emqx.Spec.Modules)
		return map[string]string{
			"name":      fmt.Sprintf("%s-%s", emqx.GetName(), "loaded-modules"),
			"mountPath": "/opt/emqx/data/loaded_modules",
			"subPath":   "loaded_modules",
			"conf":      string(data),
		}
	}
	return map[string]string{}
}

func stringEmqxBrokerLoadedModules(modules []v1beta1.EmqxBrokerModules) string {
	var p string
	for _, module := range modules {
		p = fmt.Sprintf("%s{%s, %t}.\n", p, module.Name, module.Enable)
	}
	return p
}
