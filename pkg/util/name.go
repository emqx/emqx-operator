package util

import (
	"fmt"

	"github.com/emqx/emqx-operator/api/v1alpha1"
)

const (
	BASE_NAME = "emqx"
	EMQX_NAME = "-cluster"
)

func GenerateName(typeName, metaName string) string {
	return fmt.Sprintf("%s%s-%s", BASE_NAME, typeName, metaName)
}

// GetEmqxName returns the name for Emqx resources
func GetEmqxName(emqx *v1alpha1.Emqx) string {
	return GenerateName(EMQX_NAME, emqx.Name)
}

func GetEmqxSecret(emqx *v1alpha1.Emqx) string {
	return GenerateName("-secret", emqx.Name)
}

func GetEmqxHeadlessSvc(emqx *v1alpha1.Emqx) string {
	return GenerateName("-headless-svc", emqx.Name)
}

func GetEmqxConfigMapForAcl(emqx *v1alpha1.Emqx) string {
	return GenerateName("-configmap-acl", emqx.Name)
}

func GetEmqxConfigMapForLM(emqx *v1alpha1.Emqx) string {
	return GenerateName("-configmap-loaded-modules", emqx.Name)
}

func GetEmqxConfigMapForPG(emqx *v1alpha1.Emqx) string {
	return GenerateName("-configmap-loaded-plugins", emqx.Name)
}

func GetPvcName(s string) string {
	return GenerateName("-pvc", s)
}
