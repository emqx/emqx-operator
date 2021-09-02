package controllers

import (
	"github.com/emqx/emqx-operator/api/v1alpha1"
)

func makeSecretStringData(instance *v1alpha1.Emqx) map[string]string {
	license := instance.Spec.License
	stringData := map[string]string{"emqx.lic": license}
	return stringData
}
