package v1beta2_test

import (
	"testing"

	v1beta2 "github.com/emqx/emqx-operator/apis/apps/v1beta2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDefaultEnterprise(t *testing.T) {
	defaultReplicas := int32(3)
	emqx := &v1beta2.EmqxEnterprise{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		Spec: v1beta2.EmqxEnterpriseSpec{
			Replicas: &defaultReplicas,
			Image:    "emqx/emqx:4.3.11",
			Labels: map[string]string{
				"cluster": "emqx",
			},
			EmqxTemplate: v1beta2.EmqxEnterpriseTemplate{
				Listener: v1beta2.Listener{
					Ports: v1beta2.Ports{
						MQTTS: 8885,
					},
				},
			},
		},
	}

	emqx.Default()

	assert.Equal(t, *emqx.Spec.Replicas, int32(3))

	// Labels
	assert.Contains(t, emqx.Labels, "foo")
	assert.Contains(t, emqx.Labels, "cluster")
	assert.Contains(t, emqx.Labels, "apps.emqx.io/managed-by")
	assert.Contains(t, emqx.Labels, "apps.emqx.io/instance")

	assert.Contains(t, emqx.Spec.Labels, "foo")
	assert.Contains(t, emqx.Spec.Labels, "cluster")
	assert.Contains(t, emqx.Spec.Labels, "apps.emqx.io/managed-by")
	assert.Contains(t, emqx.Spec.Labels, "apps.emqx.io/instance")

	// Listener
	assert.Equal(t, emqx.Spec.EmqxTemplate.Listener.Type, corev1.ServiceType("ClusterIP"))
	assert.Equal(t, emqx.Spec.EmqxTemplate.Listener.Ports.MQTTS, int32(8885))
	assert.Equal(t, emqx.Spec.EmqxTemplate.Listener.Ports.API, int32(8081))

	telegrafConf := `
[global_tags]
  instanceID = "test"

[[inputs.http]]
  urls = ["http://127.0.0.1:8081/api/v4/emqx_prometheus"]
  method = "GET"
  timeout = "5s"
  username = "admin"
  password = "public"
  data_format = "json"
[[inputs.tail]]
  files = ["/opt/emqx/log/emqx.log.[1-5]"]
  from_beginning = false
  max_undelivered_lines = 64
  character_encoding = "utf-8"
  data_format = "grok"
  grok_patterns = ['^%{TIMESTAMP_ISO8601:timestamp:ts-"2006-01-02T15:04:05.999999999-07:00"} \[%{LOGLEVEL:level}\] (?m)%{GREEDYDATA:messages}$']

[[outputs.discard]]
`
	emqx.Spec.TelegrafTemplate = &v1beta2.TelegrafTemplate{
		Image: "telegraf:1.19.3",
		Conf:  &telegrafConf,
	}
	emqx.Default()
	assert.Subset(t, emqx.Spec.EmqxTemplate.Plugins,
		[]v1beta2.Plugin{
			{
				Name:   "emqx_prometheus",
				Enable: true,
			},
		},
	)
	assert.Subset(t, emqx.Spec.Env,
		[]corev1.EnvVar{
			{
				Name:  "EMQX_PROMETHEUS__PUSH__GATEWAY__SERVER",
				Value: "",
			},
		},
	)
}
