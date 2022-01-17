package util_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/emqx/emqx-operator/apis/apps/v1beta1"
	"github.com/emqx/emqx-operator/pkg/util"
	"k8s.io/apimachinery/pkg/runtime"
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
		util.GetLoadedModules(&emqxBroker)["conf"],
		"{foo, true}.\n{bar, false}.\n{emqx_mod_acl_internal, true}.\n",
	)
}

func TestGenerateEmqxEnterpriseLoadedModules(t *testing.T) {
	modules := []v1beta1.EmqxEnterpriseModules{
		{
			Name:    "fake",
			Enable:  true,
			Configs: runtime.RawExtension{Raw: []byte(`{"foo": "bar"}`)},
		},
	}

	emqxEnterprise := v1beta1.EmqxEnterprise{
		Spec: v1beta1.EmqxEnterpriseSpec{
			Modules: modules,
		},
	}
	emqxEnterprise.Default()

	assert.Equal(t,
		util.GetLoadedModules(&emqxEnterprise)["conf"],
		`[{"name":"fake","enable":true,"configs":{"foo":"bar"}},{"name":"internal_cal","enable":true,"configs":{"acl_rule_file":"etc/acl.conf"}},{"name":"retainer","enable":true,"configs":{"expiry_interval":0,"max_payload_size":"1MB","max_retained_messages":0,"storage_type":"ram"}}]`,
	)
}
