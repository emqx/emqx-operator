package v1beta2_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/emqx/emqx-operator/apis/apps/v1beta2"
)

func TestEmqxBrokerModulesDefault(t *testing.T) {
	modules := &v1beta2.EmqxBrokerModulesList{
		Items: []v1beta2.EmqxBrokerModules{
			{
				Name:   "foo",
				Enable: true,
			},
			{
				Name:   "bar",
				Enable: false,
			},
		},
	}

	modules.Default()
	assert.ElementsMatch(t, modules.Items,
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

}

func TestEmqxBrokerModulesString(t *testing.T) {
	modules := &v1beta2.EmqxBrokerModulesList{
		Items: []v1beta2.EmqxBrokerModules{
			{
				Name:   "foo",
				Enable: true,
			},
			{
				Name:   "bar",
				Enable: false,
			},
		},
	}

	assert.Equal(t, modules.String(),
		"{foo, true}.\n{bar, false}.\n",
	)
}

func TestEmqxEnterpriseModulesDefault(t *testing.T) {
	modules := &v1beta2.EmqxEnterpriseModulesList{
		Items: []v1beta2.EmqxEnterpriseModules{
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
	}

	modules.Default()
	assert.ElementsMatch(t, modules.Items,
		[]v1beta2.EmqxEnterpriseModules{
			{
				Name:    "fake",
				Enable:  true,
				Configs: runtime.RawExtension{Raw: []byte(`{"foo": "bar"}`)},
			},
			{
				Name:    "internal_acl",
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
