package util

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NameForLicense(emqx client.Object) string {
	return fmt.Sprintf("%s-%s", emqx.GetName(), "license")
}
func NameForACL(emqx client.Object) string {
	return fmt.Sprintf("%s-%s", emqx.GetName(), "acl")
}

func NameForPlugins(emqx client.Object) string {
	return fmt.Sprintf("%s-%s", emqx.GetName(), "loaded-plugins")
}

func NameForModules(emqx client.Object) string {
	return fmt.Sprintf("%s-%s", emqx.GetName(), "loaded-modules")
}

func NameForData(emqx client.Object) string {
	return fmt.Sprintf("%s-%s", emqx.GetName(), "data")
}

func NameForLog(emqx client.Object) string {
	return fmt.Sprintf("%s-%s", emqx.GetName(), "log")
}

func NameForMQTTSCertificate(emqx client.Object) string {
	return fmt.Sprintf("%s-%s-%s", emqx.GetName(), "mqtts", "cert")
}

func NameForWSSCertificate(emqx client.Object) string {
	return fmt.Sprintf("%s-%s-%s", emqx.GetName(), "wss", "cert")
}

func NameForTelegraf(emqx client.Object) string {
	return fmt.Sprintf("%s-%s", emqx.GetName(), "telegraf")
}
