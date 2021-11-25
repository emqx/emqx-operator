package util_test

import (
	"testing"

	"github.com/emqx/emqx-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
)

func TestGenerateEnv(t *testing.T) {
	env := []corev1.EnvVar{
		{
			Name:  "EMQX_NAME",
			Value: "foo",
		},
		{
			Name:  "EMQX_CLUSTER__K8S__NAMESPACE",
			Value: "bar",
		},
		{
			Name:  "EMQX_FOO",
			Value: "bar",
		},
	}

	matched := 0

	for _, e := range util.GenerateEnv("emqx", "default", env) {
		if e.Name == "EMQX_NAME" {
			if e.Value == "foo" {
				matched += 1
			} else {
				t.Errorf("unexpected data: %+v", e)
			}
		}
		if e.Name == "EMQX_CLUSTER__K8S__NAMESPACE" {
			if e.Value == "bar" {
				matched += 1
			} else {
				t.Errorf("unexpected data: %+v", e)
			}
		}
		if e.Name == "EMQX_FOO" {
			if e.Value == "bar" {
				matched += 1
			} else {
				t.Errorf("unexpected data: %+v", e)
			}
		}
	}

	if matched != len(env) {
		t.Errorf("unexpected data: %+v", env)
	}

}
