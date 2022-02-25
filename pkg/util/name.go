package util

import (
	"fmt"

	"github.com/emqx/emqx-operator/apis/apps/v1beta1"
)

func NameForLicense(emqx v1beta1.Emqx) string {
	return fmt.Sprintf("%s-%s", emqx.GetName(), "license")
}
func NameForACL(emqx v1beta1.Emqx) string {
	return fmt.Sprintf("%s-%s", emqx.GetName(), "acl")
}

func NameForPlugins(emqx v1beta1.Emqx) string {
	return fmt.Sprintf("%s-%s", emqx.GetName(), "loaded-plugins")
}

func NameForModules(emqx v1beta1.Emqx) string {
	return fmt.Sprintf("%s-%s", emqx.GetName(), "loaded-modules")
}

func NameForData(emqx v1beta1.Emqx) string {
	return fmt.Sprintf("%s-%s", emqx.GetName(), "data")
}

func NameForLog(emqx v1beta1.Emqx) string {
	return fmt.Sprintf("%s-%s", emqx.GetName(), "log")
}

func NameForMQTTSCertificate(emqx v1beta1.Emqx) string {
	return fmt.Sprintf("%s-%s-%s", emqx.GetName(), "mqtts", "cert")
}

func NameForWSSCertificate(emqx v1beta1.Emqx) string {
	return fmt.Sprintf("%s-%s-%s", emqx.GetName(), "wss", "cert")
}

func NameForTelegraf(emqx v1beta1.Emqx) string {
	return fmt.Sprintf("%s-%s", emqx.GetName(), "telegraf")
}
