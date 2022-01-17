package v1beta1_test

import (
	"testing"

	"github.com/emqx/emqx-operator/apis/apps/v1beta1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestDefaultEnterprise(t *testing.T) {
	emqx := &v1beta1.EmqxEnterprise{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: v1beta1.EmqxEnterpriseSpec{
			Image: "emqx/emqx-ee:4.3.6",
			Modules: []v1beta1.EmqxEnterpriseModules{
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
	}

	emqx.Default()
	assert.ElementsMatch(t, emqx.Spec.Modules,
		[]v1beta1.EmqxEnterpriseModules{
			{
				Name:    "fake",
				Enable:  true,
				Configs: runtime.RawExtension{Raw: []byte(`{"foo": "bar"}`)},
			},
			{
				Name:    "internal_cal",
				Enable:  true,
				Configs: runtime.RawExtension{Raw: []byte(`{"acl_rule_file": "etc/acl.conf"}`)},
			},
			{
				Name:   "retainer",
				Enable: false,
			},
		},
	)
}
