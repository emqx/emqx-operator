package v1beta2_test

import (
	"testing"

	"github.com/emqx/emqx-operator/apis/apps/v1beta2"
	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var v1beta2Broker = &v1beta2.EmqxBroker{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "emqx",
		Namespace: "emqx",
		Labels: map[string]string{
			"emqx": "cluster",
		},
		Annotations: map[string]string{
			"bar": "foo",
		},
	},
	Spec: v1beta2.EmqxBrokerSpec{
		Image: "emqx/emqx:4.3.11",
		Labels: map[string]string{
			"cluster": "emqx",
		},
		Annotations: map[string]string{
			"foo": "bar",
		},
		Storage: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				"ReadWriteOnce",
			},
		},
		EmqxTemplate: v1beta2.EmqxBrokerTemplate{
			Plugins: []v1beta3.Plugin{
				{
					Name:   "foo",
					Enable: true,
				},
				{
					Name:   "bar",
					Enable: false,
				},
			},
			Modules: []v1beta3.EmqxBrokerModule{
				{
					Name:   "emqx_mod_acl_internal",
					Enable: true,
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

var v1beta3Broker = &v1beta3.EmqxBroker{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "emqx",
		Namespace: "emqx",
		Labels: map[string]string{
			"cluster": "emqx",
			"emqx":    "cluster",
		},
		Annotations: map[string]string{
			"foo": "bar",
			"bar": "foo",
		},
	},
	Spec: v1beta3.EmqxBrokerSpec{
		Image: "emqx/emqx:4.3.11",
		Persistent: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				"ReadWriteOnce",
			},
		},
		EmqxTemplate: v1beta3.EmqxBrokerTemplate{
			Plugins: []v1beta3.Plugin{
				{
					Name:   "foo",
					Enable: true,
				},
				{
					Name:   "bar",
					Enable: false,
				},
			},
			Modules: []v1beta3.EmqxBrokerModule{
				{
					Name:   "emqx_mod_acl_internal",
					Enable: true,
				},
			},
			Listener: v1beta3.Listener{
				MQTTS: v1beta3.ListenerPort{
					Port: int32(8885),
				},
			},
		},
	},
}

func TestConvertToBroker(t *testing.T) {
	emqx := &v1beta3.EmqxBroker{}
	err := v1beta2Broker.ConvertTo(emqx)

	assert.Nil(t, err)
	assert.Equal(t, emqx, v1beta3Broker)
}

func TestConvertFromBroker(t *testing.T) {
	emqx := &v1beta2.EmqxBroker{}
	err := emqx.ConvertFrom(v1beta3Broker)

	v1beta2Broker.Labels = v1beta3Broker.Labels
	v1beta2Broker.Annotations = v1beta3Broker.Annotations
	v1beta2Broker.Spec.Labels = v1beta3Broker.Labels
	v1beta2Broker.Spec.Annotations = v1beta3Broker.Annotations

	assert.Nil(t, err)
	assert.Equal(t, emqx, v1beta2Broker)
}
