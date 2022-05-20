package v1beta2_test

import (
	"testing"

	v1beta2 "github.com/emqx/emqx-operator/apis/apps/v1beta2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDefaultForClusterEnv(t *testing.T) {
	emqx := &v1beta2.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: v1beta2.EmqxBrokerSpec{
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

func TestDefaultForServiceAccountName(t *testing.T) {
	emqx := &v1beta2.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: v1beta2.EmqxBrokerSpec{
			Image: "emqx/emqx:4.3.11",
		},
	}
	emqx.Default()
	assert.Equal(t, emqx.Spec.ServiceAccountName, "emqx")

	emqx.Spec.Image = "emqx/emqx:4.4.0"
	emqx.Spec.Env = []corev1.EnvVar{}
	emqx.Spec.ServiceAccountName = "fake"
	emqx.Default()
	assert.Equal(t, emqx.Spec.ServiceAccountName, "")
}

func TestDefaultBroker(t *testing.T) {
	defaultReplicas := int32(3)
	emqx := &v1beta2.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		Spec: v1beta2.EmqxBrokerSpec{
			Replicas: &defaultReplicas,
			Image:    "emqx/emqx:4.3.11",
			Labels: map[string]string{
				"cluster": "emqx",
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

}

func TestValidateCreateBroker(t *testing.T) {
	emqx := v1beta2.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: v1beta2.EmqxBrokerSpec{
			Image: "emqx/emqx:4.3.11",
		},
	}

	emqx.Default()
	assert.Nil(t, emqx.ValidateCreate())

	emqx.Spec.Image = "emqx/emqx:fake"
	assert.Error(t, emqx.ValidateCreate())

	emqx.Spec.Image = "127.0.0.1:8443/emqx/emqx:4.3.11"
	assert.Nil(t, emqx.ValidateCreate())
}

func TestValidateUpdateBroker(t *testing.T) {
	emqx := v1beta2.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: v1beta2.EmqxBrokerSpec{
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

	emqx.Spec.Image = "127.0.0.1:8443/emqx/emqx:4.3.11"
	assert.Nil(t, emqx.ValidateUpdate(new))
}
