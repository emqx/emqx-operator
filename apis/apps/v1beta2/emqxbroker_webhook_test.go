package v1beta2_test

import (
	"testing"

	v1beta2 "github.com/emqx/emqx-operator/apis/apps/v1beta2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDefaultForEnv(t *testing.T) {
	emqx := &v1beta2.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: v1beta2.EmqxBrokerSpec{
			Image: "emqx/emqx:4.3.11",
			Env: []corev1.EnvVar{
				{
					Name:  "EMQX_NAME",
					Value: "foo",
				},
				{
					Name:  "EMQX_FOO",
					Value: "bar",
				},
			},
		},
	}

	emqx.Default()

	assert.ElementsMatch(t, emqx.Spec.Env,
		[]corev1.EnvVar{
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

	emqx.Spec.Image = "emqx/emqx:4.4.0"
	emqx.Spec.Env = []corev1.EnvVar{}
	emqx.Default()

	assert.ElementsMatch(t, emqx.Spec.Env,
		[]corev1.EnvVar{
			{
				Name:  "EMQX_NAME",
				Value: "emqx",
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
	emqx := &v1beta2.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		Spec: v1beta2.EmqxBrokerSpec{
			Image: "emqx/emqx:4.3.11",
			Labels: map[string]string{
				"cluster": "emqx",
			},
			EmqxTemplate: v1beta2.EmqxBrokerTemplate{
				Plugins: []v1beta2.Plugin{
					{
						Name:   "foo",
						Enable: true,
					},
					{
						Name:   "bar",
						Enable: false,
					},
				},
				Modules: []v1beta2.EmqxBrokerModules{
					{
						Name:   "foo",
						Enable: true,
					},
					{
						Name:   "bar",
						Enable: false,
					},
				},
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

	// ACL
	assert.ElementsMatch(t, emqx.Spec.EmqxTemplate.ACL,
		[]v1beta2.ACL{
			{
				Permission: "allow",
				Username:   "dashboard",
				Action:     "subscribe",
				Topics: v1beta2.Topics{
					Filter: []string{
						"$STS?#",
					},
				},
			},
			{
				Permission: "allow",
				IPAddress:  "127.0.0.1",
				Topics: v1beta2.Topics{
					Filter: []string{
						"$SYS/#",
						"#",
					},
				},
			},
			{
				Permission: "deny",
				Action:     "subscribe",
				Topics: v1beta2.Topics{
					Filter: []string{"$SYS/#"},
					Equal:  []string{"#"},
				},
			},
			{
				Permission: "allow",
			},
		},
	)

	// Plugins
	assert.ElementsMatch(t, emqx.Spec.EmqxTemplate.Plugins,
		[]v1beta2.Plugin{
			{
				Name:   "foo",
				Enable: true,
			},
			{
				Name:   "bar",
				Enable: false,
			},
			{
				Name:   "emqx_management",
				Enable: true,
			},
		},
	)

	// Modules
	assert.ElementsMatch(t, emqx.Spec.EmqxTemplate.Modules,
		[]v1beta2.EmqxBrokerModules{
			{
				Name:   "foo",
				Enable: true,
			},
			{
				Name:   "bar",
				Enable: false,
			},
			{
				Name:   "emqx_mod_acl_internal",
				Enable: true,
			},
		},
	)

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
}
