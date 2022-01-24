package v1beta2_test

import (
	"testing"

	"github.com/emqx/emqx-operator/apis/apps/v1beta1"
	"github.com/emqx/emqx-operator/apis/apps/v1beta2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var v1beta1Broker = &v1beta1.EmqxBroker{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "emqx",
		Namespace: "emqx",
	},
	Spec: v1beta1.EmqxBrokerSpec{
		Image: "emqx/emqx:4.3.11",
		Labels: map[string]string{
			"cluster": "emqx",
		},
		Env: []corev1.EnvVar{
			{
				Name:  "EMQX_LOG__LEVEL",
				Value: "debug",
			},
		},
		Storage: &v1beta1.Storage{
			VolumeClaimTemplate: v1beta1.EmbeddedPersistentVolumeClaim{
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						"ReadWriteOnce",
					},
				},
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
				Name:   "emqx_mod_acl_internal",
				Enable: true,
			},
		},
		Listener: v1beta1.Listener{
			Ports: v1beta1.Ports{
				MQTTS: 8885,
			},
		},
	},
}

var v1beta2Broker = &v1beta2.EmqxBroker{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "emqx",
		Namespace: "emqx",
	},
	Spec: v1beta2.EmqxBrokerSpec{
		Image: "emqx/emqx:4.3.11",
		Labels: map[string]string{
			"cluster": "emqx",
		},
		Env: []corev1.EnvVar{
			{
				Name:  "EMQX_LOG__LEVEL",
				Value: "debug",
			},
		},
		Storage: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				"ReadWriteOnce",
			},
		},
		EmqxTemplate: v1beta2.EmqxBrokerTemplate{
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
					Name:   "emqx_mod_acl_internal",
					Enable: true,
				},
			},
			Listener: v1beta1.Listener{
				Ports: v1beta1.Ports{
					MQTTS: 8885,
				},
			},
		},
	},
}

func TestConvertToBroker(t *testing.T) {
	emqx := &v1beta1.EmqxBroker{}
	err := v1beta2Broker.ConvertTo(emqx)
	assert.Nil(t, err)
	assert.Equal(t, emqx, v1beta1Broker)
}

func TestConvertFromBroker(t *testing.T) {
	emqx := &v1beta2.EmqxBroker{}
	err := emqx.ConvertFrom(v1beta1Broker)
	assert.Nil(t, err)
	assert.Equal(t, emqx, v1beta2Broker)
}
