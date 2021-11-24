package util_test

import (
	"fmt"
	"testing"

	"github.com/emqx/emqx-operator/pkg/util"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGenEmqxBrokerLoadedModules(t *testing.T) {
	modules := []util.EmqxBrokerModules{
		{
			Name:   "foo",
			Enable: true,
		},
		{
			Name:   "bar",
			Enable: false,
		},
	}

	m := util.GenEmqxBrokerLoadedModules(modules)
	if m != "{foo, true}.\n{bar, false}.\n" {
		t.Errorf("unexpected data: %s", m)
	}
}

func TestGenEmqxEnterpriseLoadedModules(t *testing.T) {
	modules := []util.EmqxEnterpriseModules{
		{
			Name:    "fake",
			Enable:  true,
			Configs: runtime.RawExtension{Raw: []byte(`{"foo": "bar"}`)},
		},
	}

	m := util.GenEmqxEnterpriseLoadedModules(modules)

	fmt.Printf("%+v", m)
	// if m != `[{"name":"fake","enable":true,"configs":{"foo":"bar"}}]` {
	// 	t.Errorf("unexpected data: %s", m)
	// }
}
