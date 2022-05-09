package v1beta3_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
)

func TestEnv(t *testing.T) {
	envs := &v1beta3.EnvList{
		Items: []corev1.EnvVar{
			{
				Name:  "EMQX_NAME",
				Value: "foo",
			},
			{
				Name:  "EMQX_FOO",
				Value: "bar",
			},
		},
	}

	emqx := &v1beta3.EmqxEnterprise{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
	}
	envs.ClusterForDNS(emqx)

	assert.ElementsMatch(t, envs.Items,
		[]corev1.EnvVar{
			{
				Name:  "EMQX_LOG__TO",
				Value: "both",
			},
			{
				Name:  "EMQX_FOO",
				Value: "bar",
			},
			{
				Name:  "EMQX_NAME",
				Value: "foo",
			},
			{
				Name:  "EMQX_CLUSTER__DISCOVERY",
				Value: "dns",
			},
			{
				Name:  "EMQX_CLUSTER__DNS__TYPE",
				Value: "srv",
			},
			{
				Name:  "EMQX_CLUSTER__DNS__APP",
				Value: "emqx",
			},
			{
				Name:  "EMQX_CLUSTER__DNS__NAME",
				Value: "emqx-headless.emqx.svc.cluster.local",
			},
		},
	)
}
