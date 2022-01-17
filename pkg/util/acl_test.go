package util_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/emqx/emqx-operator/apis/apps/v1beta1"
	"github.com/emqx/emqx-operator/pkg/util"
)

func TestGenerateACL(t *testing.T) {
	var emqxBroker v1beta1.EmqxBroker
	var emqxEneterprise v1beta1.EmqxEnterprise
	var acl v1beta1.ACL

	acl = v1beta1.ACL{Permission: "allow"}
	emqxBroker = v1beta1.EmqxBroker{
		Spec: v1beta1.EmqxBrokerSpec{
			ACL: []v1beta1.ACL{
				acl,
			},
		},
	}
	emqxBroker.Default()

	assert.Equal(t,
		util.GetACL(&emqxBroker)["conf"],
		"{allow, all, pubsub, [\"#\"]}.\n",
	)

	acl = v1beta1.ACL{
		Permission: "deny",
		Action:     "subscribe",
		Topics: v1beta1.Topics{
			Filter: []string{"$SYS/#"},
			Equal:  []string{"#"},
		},
	}
	emqxBroker = v1beta1.EmqxBroker{
		Spec: v1beta1.EmqxBrokerSpec{
			ACL: []v1beta1.ACL{
				acl,
			},
		},
	}
	emqxBroker.Default()

	assert.Equal(t,
		util.GetACL(&emqxBroker)["conf"],
		"{deny, all, subscribe, [\"$SYS/#\", {eq, \"#\"}]}.\n",
	)

	acl = v1beta1.ACL{
		Permission: "allow",
		Username:   "admin",
		ClientID:   "emqx",
		IPAddress:  "127.0.0.1",
		Topics: v1beta1.Topics{
			Filter: []string{
				"$SYS/#",
				"#",
			},
		},
	}
	emqxEneterprise = v1beta1.EmqxEnterprise{
		Spec: v1beta1.EmqxEnterpriseSpec{
			ACL: []v1beta1.ACL{
				acl,
			},
		},
	}
	emqxEneterprise.Default()

	assert.Equal(t,
		util.GetACL(&emqxEneterprise)["conf"],
		"{allow, {'and', [{user, \"admin\"}, {client, \"emqx\"}, {ipaddr, \"127.0.0.1\"}]}, pubsub, [\"$SYS/#\", \"#\"]}.\n",
	)
}
