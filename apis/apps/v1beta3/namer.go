package v1beta3

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

//+kubebuilder:object:generate=false
type Names struct {
	client.Object
}

func (n Names) HeadlessSvc() string {
	return fmt.Sprintf("%s-%s", n.Object.GetName(), "headless")
}

func (n Names) License() string {
	return fmt.Sprintf("%s-%s", n.Object.GetName(), "license")
}

func (n Names) ACL() string {
	return fmt.Sprintf("%s-%s", n.Object.GetName(), "acl")
}

func (n Names) PluginsConfig() string {
	return fmt.Sprintf("%s-%s", n.Object.GetName(), "plugins-config")
}

func (n Names) LoadedPlugins() string {
	return fmt.Sprintf("%s-%s", n.Object.GetName(), "loaded-plugins")
}

func (n Names) LoadedModules() string {
	return fmt.Sprintf("%s-%s", n.Object.GetName(), "loaded-modules")
}

func (n Names) Data() string {
	return fmt.Sprintf("%s-%s", n.Object.GetName(), "data")
}
