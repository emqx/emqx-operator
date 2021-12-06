package v1beta1_test

import (
	"testing"

	"github.com/emqx/emqx-operator/api/v1beta1"
	"github.com/stretchr/testify/assert"
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

	assert.Equal(t,
		emqxBroker.GetLoadedPlugins()["conf"],
		"{foo, true}.\n{bar, false}.\n{emqx_management, true}.\n",
	)
}
