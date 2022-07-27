package v1beta3

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestEmqxBrokerModulesString(t *testing.T) {
	modules := &EmqxBrokerModuleList{
		Items: []EmqxBrokerModule{
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

func TestEmqxEnterpriseModulesString(t *testing.T) {
	var modules *EmqxEnterpriseModuleList

	modules = &EmqxEnterpriseModuleList{}
	assert.Equal(t, modules.String(), "")

	modules = &EmqxEnterpriseModuleList{
		Items: []EmqxEnterpriseModule{
			{
				Name:    "internal_acl",
				Enable:  true,
				Configs: runtime.RawExtension{Raw: []byte(`{"acl_rule_file": "/mounted/acl/acl.conf"}`)},
			},
			{
				Name:   "retainer",
				Enable: true,
				Configs: runtime.RawExtension{Raw: []byte(`{
					"expiry_interval": 0,
					"max_payload_size": "1MB",
					"max_retained_messages": 0,
					"storage_type": "ram"
				}`)},
			},
		},
	}
	assert.Equal(t, modules.String(), `[{"name":"internal_acl","enable":true,"configs":{"acl_rule_file":"/mounted/acl/acl.conf"}},{"name":"retainer","enable":true,"configs":{"expiry_interval":0,"max_payload_size":"1MB","max_retained_messages":0,"storage_type":"ram"}}]`)
}
