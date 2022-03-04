package v1beta3_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
)

func TestEnvK8S(t *testing.T) {
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

	emvBroker := &v1beta3.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
	}
	envs.ClusterForK8S(emvBroker)

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
				Value: "k8s",
			},
			{
				Name:  "EMQX_CLUSTER__K8S__APP_NAME",
				Value: "emqx",
			},
			{
				Name:  "EMQX_CLUSTER__K8S__SERVICE_NAME",
				Value: "emqx-headless",
			},
			{
				Name:  "EMQX_CLUSTER__K8S__NAMESPACE",
				Value: "emqx",
			},
			{
				Name:  "EMQX_CLUSTER__K8S__APISERVER",
				Value: "https://kubernetes.default.svc:443",
			},
			{
				Name:  "EMQX_CLUSTER__K8S__ADDRESS_TYPE",
				Value: "hostname",
			},
			{
				Name:  "EMQX_CLUSTER__K8S__SUFFIX",
				Value: "svc.cluster.local",
			},
		},
	)
}

func TestEnvDNS(t *testing.T) {
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
