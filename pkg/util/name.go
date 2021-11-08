package util

import (
	"fmt"

	"github.com/emqx/emqx-operator/api/v1alpha2"
)

const (
	BASE_NAME = "emqx"
	EMQX_NAME = "-cluster"
)

func GenerateName(typeName, metaName string) string {
	return fmt.Sprintf("%s%s-%s", BASE_NAME, typeName, metaName)
}

// GetEmqxBrokerName returns the name for EmqxBroker resources
func GetEmqxBrokerName(emqx *v1alpha2.EmqxBroker) string {
	return GenerateName(EMQX_NAME, emqx.Name)
}

func GetEmqxBrokerSecret(emqx *v1alpha2.EmqxBroker) string {
	return GenerateName("-secret", emqx.Name)
}

func GetEmqxBrokerHeadlessSvc(emqx *v1alpha2.EmqxBroker) string {
	return GenerateName("-headless-svc", emqx.Name)
}

func GetEmqxBrokerConfigMapForAcl(emqx *v1alpha2.EmqxBroker) string {
	return GenerateName("-configmap-acl", emqx.Name)
}

func GetEmqxBrokerConfigMapForLM(emqx *v1alpha2.EmqxBroker) string {
	return GenerateName("-configmap-loaded-modules", emqx.Name)
}

func GetEmqxBrokerConfigMapForPG(emqx *v1alpha2.EmqxBroker) string {
	return GenerateName("-configmap-loaded-plugins", emqx.Name)
}

func GetPvcName(s string) string {
	return GenerateName("-pvc", s)
}
