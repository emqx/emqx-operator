package v1beta2_test

import (
	"testing"

	v1beta2 "github.com/emqx/emqx-operator/apis/apps/v1beta2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestDefaultEnterprise(t *testing.T) {
	emqx := &v1beta2.EmqxEnterprise{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: v1beta2.EmqxEnterpriseSpec{
			Image: "emqx/emqx-ee:4.3.6",
			EmqxTemplate: v1beta2.EmqxEnterpriseTemplate{
				Modules: []v1beta2.EmqxEnterpriseModules{
					{
						Name:    "fake",
						Enable:  true,
						Configs: runtime.RawExtension{Raw: []byte(`{"foo": "bar"}`)},
					},
					{
						Name:   "retainer",
						Enable: false,
					},
				},
			},
		},
	}

	emqx.Default()
	assert.ElementsMatch(t, emqx.Spec.EmqxTemplate.Modules,
		[]v1beta2.EmqxEnterpriseModules{
			{
				Name:    "fake",
				Enable:  true,
				Configs: runtime.RawExtension{Raw: []byte(`{"foo": "bar"}`)},
			},
			{
				Name:    "internal_cal",
				Enable:  true,
				Configs: runtime.RawExtension{Raw: []byte(`{"acl_rule_file": "/mounted/acl/acl.conf"}`)},
			},
			{
				Name:   "retainer",
				Enable: false,
			},
		},
	)
}
