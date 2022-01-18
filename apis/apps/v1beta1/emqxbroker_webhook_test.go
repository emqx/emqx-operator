package v1beta1_test

import (
	"testing"

	"github.com/emqx/emqx-operator/apis/apps/v1beta1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDefaultBroker(t *testing.T) {
	emqx := &v1beta1.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		Spec: v1beta1.EmqxBrokerSpec{
			Image: "emqx/emqx:4.3.11",
			Labels: map[string]string{
				"cluster": "emqx",
			},
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
			Plugins: []v1beta1.Plugin{
				{
					Name:   "foo",
					Enable: true,
				},
				{
					Name:   "bar",
					Enable: false,
				},
			},
			Modules: []v1beta1.EmqxBrokerModules{
				{
					Name:   "foo",
					Enable: true,
				},
				{
					Name:   "bar",
					Enable: false,
				},
			},
			Listener: v1beta1.Listener{
				Ports: v1beta1.Ports{
					MQTTS: 8885,
				},
			},
		},
	}

	emqx.Default()

	assert.Equal(t, *emqx.Spec.Replicas, int32(3))
	assert.Equal(t, emqx.Spec.ServiceAccountName, emqx.Name)

	// Labels
	assert.Contains(t, emqx.Labels, "foo")
	assert.Contains(t, emqx.Labels, "cluster")
	assert.Contains(t, emqx.Labels, "apps.emqx.io/managed-by")
	assert.Contains(t, emqx.Labels, "apps.emqx.io/instance")

	assert.Contains(t, emqx.Spec.Labels, "foo")
	assert.Contains(t, emqx.Spec.Labels, "cluster")
	assert.Contains(t, emqx.Spec.Labels, "apps.emqx.io/managed-by")
	assert.Contains(t, emqx.Spec.Labels, "apps.emqx.io/instance")

	// ENV
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
				Value: emqx.GetName(),
			},
			{
				Name:  "EMQX_CLUSTER__K8S__SERVICE_NAME",
				Value: emqx.GetHeadlessServiceName(),
			},
			{
				Name:  "EMQX_CLUSTER__K8S__NAMESPACE",
				Value: emqx.GetNamespace(),
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

	// ACL
	assert.ElementsMatch(t, emqx.Spec.ACL,
		[]v1beta1.ACL{
			{
				Permission: "allow",
				Username:   "dashboard",
				Action:     "subscribe",
				Topics: v1beta1.Topics{
					Filter: []string{
						"$STS?#",
					},
				},
			},
			{
				Permission: "allow",
				IPAddress:  "127.0.0.1",
				Topics: v1beta1.Topics{
					Filter: []string{
						"$SYS/#",
						"#",
					},
				},
			},
			{
				Permission: "deny",
				Action:     "subscribe",
				Topics: v1beta1.Topics{
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
	assert.ElementsMatch(t, emqx.Spec.Plugins,
		[]v1beta1.Plugin{
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
	assert.ElementsMatch(t, emqx.Spec.Modules,
		[]v1beta1.EmqxBrokerModules{
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
	assert.Equal(t, emqx.Spec.Listener.Type, corev1.ServiceType("ClusterIP"))
	assert.Equal(t, emqx.Spec.Listener.Ports.MQTTS, int32(8885))
	assert.Equal(t, emqx.Spec.Listener.Ports.API, int32(8081))
}

func TestValidateCreateBroker(t *testing.T) {
	emqx := v1beta1.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: v1beta1.EmqxBrokerSpec{
			Image: "emqx/emqx:4.3.11",
		},
	}

	emqx.Default()
	assert.Nil(t, emqx.ValidateCreate())

	emqx.Spec.Image = "emqx/emqx:fake"
	assert.Error(t, emqx.ValidateCreate())
}

func TestValidateUpdateBroker(t *testing.T) {
	emqx := v1beta1.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: v1beta1.EmqxBrokerSpec{
			Image: "emqx/emqx:4.3.11",
		},
	}

	emqx.Default()
	assert.Nil(t, emqx.ValidateCreate())

	new := &emqx
	new.Spec.Image = "emqx/emqx:fake"
	assert.Error(t, emqx.ValidateUpdate(new))
}
