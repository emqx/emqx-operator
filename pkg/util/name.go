package util

import (
	"fmt"

	"github.com/emqx/emqx-operator/apis/apps/v1beta1"
)

func Name4License(emqx v1beta1.Emqx) string {
	return fmt.Sprintf("%s-%s", emqx.GetName(), "license")
}
func Name4ACL(emqx v1beta1.Emqx) string {
	return fmt.Sprintf("%s-%s", emqx.GetName(), "acl")
}

func Name4Plugins(emqx v1beta1.Emqx) string {
	return fmt.Sprintf("%s-%s", emqx.GetName(), "loaded-plugins")
}

func Name4Modules(emqx v1beta1.Emqx) string {
	return fmt.Sprintf("%s-%s", emqx.GetName(), "loaded-modules")
}

func Name4Data(emqx v1beta1.Emqx) string {
	return fmt.Sprintf("%s-%s", emqx.GetName(), "data")
}

func Name4Log(emqx v1beta1.Emqx) string {
	return fmt.Sprintf("%s-%s", emqx.GetName(), "log")
}

func Name4MQTTSCertificate(emqx v1beta1.Emqx) string {
	return fmt.Sprintf("%s-%s-%s", emqx.GetName(), "mqtts", "cert")
}

func Name4WSSCertificate(emqx v1beta1.Emqx) string {
	return fmt.Sprintf("%s-%s-%s", emqx.GetName(), "wss", "cert")
}

func Name4Telegraf(emqx v1beta1.Emqx) string {
	return fmt.Sprintf("%s-%s", emqx.GetName(), "telegraf")
}
