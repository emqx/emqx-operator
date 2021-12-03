package v1beta1_test

import (
	"testing"

	"github.com/emqx/emqx-operator/api/v1beta1"
)

func TestGenerateLoadedPlugins(t *testing.T) {
	plugins := []v1beta1.Plugin{
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
			Plugins: plugins,
		},
	}

	p := emqxBroker.GetLoadedPlugins()["conf"]
	if p != "{foo, true}.\n{bar, false}.\n" {
		t.Errorf("unexpected data: %s", p)
	}
}
