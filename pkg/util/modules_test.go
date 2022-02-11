package util_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/emqx/emqx-operator/apis/apps/v1beta1"
	"github.com/emqx/emqx-operator/pkg/util"
)

func TestGenerateEmqxBrokerLoadedModules(t *testing.T) {
	modules := []v1beta1.EmqxBrokerModules{
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
			Modules: modules,
		},
	}
	emqxBroker.Default()

	assert.Equal(t,
		util.StringEmqxBrokerLoadedModules(emqxBroker.GetModules()),
		"{foo, true}.\n{bar, false}.\n{emqx_mod_acl_internal, true}.\n",
	)
}
