package util_test

import (
	"testing"

	"github.com/emqx/emqx-operator/apis/apps/v1beta2"
	"github.com/emqx/emqx-operator/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestGenerateLoadedPlugins(t *testing.T) {
	plugins := []v1beta2.Plugin{
		{
			Name:   "foo",
			Enable: true,
		},
		{
			Name:   "bar",
			Enable: false,
		},
	}

	emqxBroker := v1beta2.EmqxBroker{
		Spec: v1beta2.EmqxBrokerSpec{
			EmqxTemplate: v1beta2.EmqxBrokerTemplate{
				Plugins: plugins,
			},
		},
	}
	emqxBroker.Default()

	assert.Equal(t,
		util.StringLoadedPlugins(emqxBroker.GetPlugins()),
		"{foo, true}.\n{bar, false}.\n{emqx_management, true}.\n",
	)
}
