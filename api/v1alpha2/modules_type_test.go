package v1alpha2_test

import (
	"fmt"
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

	m := v1alpha2.GenerateEmqxBrokerLoadedModules(modules)
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

	m := v1alpha2.GenerateEmqxEnterpriseLoadedModules(modules)

	fmt.Printf("%+v", m)
	if m != `[{"name":"fake","enable":true,"configs":{"foo":"bar"}}]` {
		t.Errorf("unexpected data: %s", m)
	}
}
