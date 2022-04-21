package v1beta3_test

import (
	"testing"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDefaultForClusterEnv(t *testing.T) {
	emqx := &v1beta3.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: v1beta3.EmqxBrokerSpec{
			Image: "emqx/emqx:4.3.11",
		},
	}

	emqx.Default()

	assert.Subset(t, emqx.Spec.Env,
		[]corev1.EnvVar{
			{
				Name:  "EMQX_CLUSTER__DISCOVERY",
				Value: "k8s",
			},
		},
	)

	emqx.Spec.Image = "emqx/emqx:4.4.0"
	emqx.Spec.Env = []corev1.EnvVar{}
	emqx.Default()

	assert.Subset(t, emqx.Spec.Env,
		[]corev1.EnvVar{
			{
				Name:  "EMQX_CLUSTER__DISCOVERY",
				Value: "dns",
			},
		},
	)
}

func TestDefaultBroker(t *testing.T) {
	emqx := &v1beta3.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
	}

	emqx.Default()

	assert.Equal(t, *emqx.Spec.Replicas, int32(3))

	// Labels
	assert.Contains(t, emqx.Labels, "foo")
	assert.Contains(t, emqx.Labels, "apps.emqx.io/managed-by")
	assert.Contains(t, emqx.Labels, "apps.emqx.io/instance")

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
	emqx.Spec.TelegrafTemplate = &v1beta3.TelegrafTemplate{
		Image: "telegraf:1.19.3",
		Conf:  &telegrafConf,
	}
	emqx.Default()
	assert.Subset(t, emqx.Spec.EmqxTemplate.Plugins,
		[]v1beta3.Plugin{
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

func TestValidateCreateBroker(t *testing.T) {
	emqx := v1beta3.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: v1beta3.EmqxBrokerSpec{
			Image: "emqx/emqx:4.3.11",
		},
	}

	emqx.Default()
	assert.Nil(t, emqx.ValidateCreate())

	emqx.Spec.Image = "emqx/emqx:fake"
	assert.Error(t, emqx.ValidateCreate())
}

func TestValidateUpdateBroker(t *testing.T) {
	emqx := v1beta3.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: v1beta3.EmqxBrokerSpec{
			Image: "emqx/emqx:4.3.11",
		},
	}

	emqx.Default()
	assert.Nil(t, emqx.ValidateCreate())

	new := &emqx
	new.Spec.Image = "emqx/emqx:fake"
	assert.Error(t, emqx.ValidateUpdate(new))

	new.Spec.Image = "emqx/emqx:latest"
	assert.Nil(t, emqx.ValidateUpdate(new))
}
