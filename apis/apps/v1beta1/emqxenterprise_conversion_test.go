package v1beta1_test

import (
	"testing"

	"github.com/emqx/emqx-operator/apis/apps/v1beta1"
	"github.com/emqx/emqx-operator/apis/apps/v1beta2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

var v1beta1Enterprise = &v1beta1.EmqxEnterprise{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "emqx",
		Namespace: "emqx",
	},
	Spec: v1beta1.EmqxEnterpriseSpec{
		Image: "emqx/emqx-ee:4.4.0",
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
		Modules: []v1beta2.EmqxEnterpriseModules{
			{
				Name:    "internal_cal",
				Enable:  true,
				Configs: runtime.RawExtension{Raw: []byte(`{"acl_rule_file": "/mounted/acl/acl.conf"}`)},
			},
			{
				Name:   "retainer",
				Enable: true,
				Configs: runtime.RawExtension{Raw: []byte(`{
							"expiry_interval": 0,
							"max_payload_size": "1MB",
							"max_retained_messages": 0,
							"storage_type": "ram"
						}`)},
			},
		},
		Listener: v1beta2.Listener{
			Ports: v1beta2.Ports{
				MQTTS: 8885,
			},
		},
	},
}

var v1beta2Enterprise = &v1beta2.EmqxEnterprise{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "emqx",
		Namespace: "emqx",
	},
	Spec: v1beta2.EmqxEnterpriseSpec{
		Image: "emqx/emqx-ee:4.4.0",
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
		EmqxTemplate: v1beta2.EmqxEnterpriseTemplate{
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
			Modules: []v1beta2.EmqxEnterpriseModules{
				{
					Name:    "internal_cal",
					Enable:  true,
					Configs: runtime.RawExtension{Raw: []byte(`{"acl_rule_file": "/mounted/acl/acl.conf"}`)},
				},
				{
					Name:   "retainer",
					Enable: true,
					Configs: runtime.RawExtension{Raw: []byte(`{
							"expiry_interval": 0,
							"max_payload_size": "1MB",
							"max_retained_messages": 0,
							"storage_type": "ram"
						}`)},
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

func TestConvertToEnterprise(t *testing.T) {
	emqx := &v1beta2.EmqxEnterprise{}
	err := v1beta1Enterprise.ConvertTo(emqx)
	assert.Nil(t, err)
	assert.Equal(t, emqx, v1beta2Enterprise)
}

func TestConvertFromEnterprise(t *testing.T) {
	emqx := &v1beta1.EmqxEnterprise{}
	err := emqx.ConvertFrom(v1beta2Enterprise)
	assert.Nil(t, err)
	assert.Equal(t, emqx, v1beta1Enterprise)
}
