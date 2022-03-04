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

func (n Names) Plugins() string {
	return fmt.Sprintf("%s-%s", n.Object.GetName(), "loaded-plugins")
}

func (n Names) Modules() string {
	return fmt.Sprintf("%s-%s", n.Object.GetName(), "loaded-modules")
}

func (n Names) Data() string {
	return fmt.Sprintf("%s-%s", n.Object.GetName(), "data")
}

func (n Names) Log() string {
	return fmt.Sprintf("%s-%s", n.Object.GetName(), "log")
}

func (n Names) MQTTSCertificate() string {
	return fmt.Sprintf("%s-%s-%s", n.Object.GetName(), "mqtts", "cert")
}

func (n Names) WSSCertificate() string {
	return fmt.Sprintf("%s-%s-%s", n.Object.GetName(), "wss", "cert")
}

func (n Names) Telegraf() string {
	return fmt.Sprintf("%s-%s", n.Object.GetName(), "telegraf")
}
