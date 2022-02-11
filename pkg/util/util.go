package util

import (
	"fmt"

	"github.com/emqx/emqx-operator/apis/apps/v1beta1"
)

func GetDataVolume(emqx v1beta1.Emqx) map[string]string {
	return map[string]string{
		"name":      fmt.Sprintf("%s-%s", emqx.GetName(), "data"),
		"mountPath": "/opt/emqx/data",
	}
}

func GetLogVolume(emqx v1beta1.Emqx) map[string]string {
	return map[string]string{
		"name":      fmt.Sprintf("%s-%s", emqx.GetName(), "log"),
		"mountPath": "/opt/emqx/log",
	}
}

func GetLicense(emqx v1beta1.EmqxEnterprise) map[string]string {
	return map[string]string{
		"name":      fmt.Sprintf("%s-%s", emqx.GetName(), "secret"),
		"mountPath": "/mounted/license/emqx.lic",
		"subPath":   "emqx.lic",
	}
}
