package v1beta3_test

import (
	"testing"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	"github.com/stretchr/testify/assert"
)

func TestPluginsDefault(t *testing.T) {
	plugins := &v1beta3.PluginList{
		Items: []v1beta3.Plugin{
			{
				Name:   "foo",
				Enable: true,
			},
			{
				Name:   "bar",
				Enable: false,
			},
		},
	}

	plugins.Default()
	assert.ElementsMatch(t, plugins.Items,
		[]v1beta3.Plugin{
			{
				Name:   "foo",
				Enable: true,
			},
			{
				Name:   "bar",
				Enable: false,
			},
			{
				Name:   "emqx_management",
				Enable: true,
			},
		},
	)
}

func TestPluginsString(t *testing.T) {
	plugins := &v1beta3.PluginList{
		Items: []v1beta3.Plugin{
			{
				Name:   "foo",
				Enable: true,
			},
			{
				Name:   "bar",
				Enable: false,
			},
		},
	}

	assert.Equal(t,
		plugins.String(),
		"{foo, true}.\n{bar, false}.\n",
	)
}
