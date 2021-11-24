package util_test

import (
	"testing"

	"github.com/emqx/emqx-operator/pkg/util"
)

func TestGenerateLoadedPlugins(t *testing.T) {
	plugins := []util.Plugin{
		{
			Name:   "foo",
			Enable: true,
		},
		{
			Name:   "bar",
			Enable: false,
		},
	}

	p := util.GenerateLoadedPlugins(plugins)
	if p != "{foo, true}.\n{bar, false}.\n" {
		t.Errorf("unexpected data: %s", p)
	}
}
