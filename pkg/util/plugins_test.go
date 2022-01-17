package util_test

import (
	"testing"

	"github.com/emqx/emqx-operator/apis/apps/v1beta1"
	"github.com/emqx/emqx-operator/pkg/util"
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
	emqxBroker.Default()

	assert.Equal(t,
		util.GetLoadedPlugins(&emqxBroker)["conf"],
		"{foo, true}.\n{bar, false}.\n{emqx_management, true}.\n",
	)
}
