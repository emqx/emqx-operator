package v1beta1_test

import (
	"testing"

	"github.com/emqx/emqx-operator/api/v1beta1"
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

	emqxBroker := v1beta1.EmqxBroker{
		Spec: v1beta1.EmqxBrokerSpec{
			Env: env,
		},
	}

	matched := 0

	for _, e := range emqxBroker.GetEnv() {
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
