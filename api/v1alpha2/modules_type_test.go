package v1alpha2_test

import (
	"testing"

	"github.com/emqx/emqx-operator/api/v1alpha2"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGenerateEmqxBrokerLoadedModules(t *testing.T) {
	modules := []v1alpha2.EmqxBrokerModules{
		{
			Name:   "foo",
			Enable: true,
		},
		{
			Name:   "bar",
			Enable: false,
		},
	}

	emqxBroker := v1alpha2.EmqxBroker{
		Spec: v1alpha2.EmqxBrokerSpec{
			Modules: modules,
		},
	}

	m := emqxBroker.GetLoadedModules()["conf"]
	if m != "{foo, true}.\n{bar, false}.\n" {
		t.Errorf("unexpected data: %s", m)
	}
}

func TestGenerateEmqxEnterpriseLoadedModules(t *testing.T) {
	modules := []v1alpha2.EmqxEnterpriseModules{
		{
			Name:    "fake",
			Enable:  true,
			Configs: runtime.RawExtension{Raw: []byte(`{"foo": "bar"}`)},
		},
	}

	emqxEnterprise := v1alpha2.EmqxEnterprise{
		Spec: v1alpha2.EmqxEnterpriseSpec{
			Modules: modules,
		},
	}

	m := emqxEnterprise.GetLoadedModules()["conf"]

	if m != `[{"name":"fake","enable":true,"configs":{"foo":"bar"}}]` {
		t.Errorf("unexpected data: %s", m)
	}
}
