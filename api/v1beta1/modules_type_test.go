package v1beta1_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/emqx/emqx-operator/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGenerateEmqxBrokerLoadedModules(t *testing.T) {
	modules := []v1beta1.EmqxBrokerModules{
		{
			Name:   "foo",
			Enable: true,
		},
		{
			Name:   "bar",
			Enable: false,
		},
	}

	emqxBroker := v1beta1.EmqxBroker{
		Spec: v1beta1.EmqxBrokerSpec{
			Modules: modules,
		},
	}

	assert.Equal(t,
		emqxBroker.GetLoadedModules()["conf"],
		"{foo, true}.\n{bar, false}.\n",
	)
}

func TestGenerateEmqxEnterpriseLoadedModules(t *testing.T) {
	modules := []v1beta1.EmqxEnterpriseModules{
		{
			Name:    "fake",
			Enable:  true,
			Configs: runtime.RawExtension{Raw: []byte(`{"foo": "bar"}`)},
		},
	}

	emqxEnterprise := v1beta1.EmqxEnterprise{
		Spec: v1beta1.EmqxEnterpriseSpec{
			Modules: modules,
		},
	}

	assert.Equal(t,
		emqxEnterprise.GetLoadedModules()["conf"],
		`[{"name":"fake","enable":true,"configs":{"foo":"bar"}}]`,
	)
}
