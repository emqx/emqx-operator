package v1beta3_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
)

func TestEmqxBrokerModulesString(t *testing.T) {
	modules := &v1beta3.EmqxBrokerModuleList{
		Items: []v1beta3.EmqxBrokerModule{
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

	assert.Equal(t, modules.String(),
		"{foo, true}.\n{bar, false}.\n",
	)
}
