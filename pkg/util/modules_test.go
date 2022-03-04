package util_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/emqx/emqx-operator/apis/apps/v1beta2"
	"github.com/emqx/emqx-operator/pkg/util"
)

func TestGenerateEmqxBrokerLoadedModules(t *testing.T) {
	modules := []v1beta2.EmqxBrokerModules{
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
				Modules: modules,
			},
		},
	}
	emqxBroker.Default()

	assert.Equal(t,
		util.StringEmqxBrokerLoadedModules(emqxBroker.GetModules()),
		"{emqx_mod_acl_internal, true}.\n{foo, true}.\n{bar, false}.\n",
	)
}
