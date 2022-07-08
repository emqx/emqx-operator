package v1beta3_test

import (
	"testing"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestMergePorts(t *testing.T) {
	serviceTemplate := &v1beta3.ServiceTemplate{
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "exist",
					Port: 8080,
				},
			},
		},
	}

	serviceTemplate.MergePorts([]corev1.ServicePort{
		{
			Name: "exist",
			Port: 8081,
		},
		{
			Name: "not-exist",
			Port: 8082,
		},
	})

	assert.ElementsMatch(t, serviceTemplate.Spec.Ports, []corev1.ServicePort{
		{
			Name: "exist",
			Port: 8080,
		},
		{
			Name: "not-exist",
			Port: 8082,
		},
	})
}
