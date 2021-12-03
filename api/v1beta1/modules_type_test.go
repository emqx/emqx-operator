package v1beta1_test

import (
	"testing"

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

	m := emqxBroker.GetLoadedModules()["conf"]
	if m != "{foo, true}.\n{bar, false}.\n" {
		t.Errorf("unexpected data: %s", m)
	}
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

	m := emqxEnterprise.GetLoadedModules()["conf"]

	if m != `[{"name":"fake","enable":true,"configs":{"foo":"bar"}}]` {
		t.Errorf("unexpected data: %s", m)
	}
}
