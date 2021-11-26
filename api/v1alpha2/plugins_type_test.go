package v1alpha2_test

import (
	"testing"

	"github.com/emqx/emqx-operator/api/v1alpha2"
)

func TestGenerateLoadedPlugins(t *testing.T) {
	plugins := []v1alpha2.Plugin{
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
			Plugins: plugins,
		},
	}

	p := emqxBroker.GetLoadedPlugins()["conf"]
	if p != "{foo, true}.\n{bar, false}.\n" {
		t.Errorf("unexpected data: %s", p)
	}
}
