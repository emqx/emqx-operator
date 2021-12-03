package v1beta1_test

import (
	"testing"

	"github.com/emqx/emqx-operator/api/v1beta1"
)

func TestGenerateACL(t *testing.T) {
	var emqxBroker v1beta1.EmqxBroker
	var emqxEneterprise v1beta1.EmqxEnterprise
	var acl v1beta1.ACL
	var s string

	acl = v1beta1.ACL{Permission: "allow"}
	emqxBroker = v1beta1.EmqxBroker{
		Spec: v1beta1.EmqxBrokerSpec{
			ACL: []v1beta1.ACL{
				acl,
			},
		},
	}

	s = emqxBroker.GetACL()["conf"]
	if s != "{allow, all, pubsub, [\"#\"]}.\n" {
		t.Errorf("unexpected data: %s", s)
	}

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
	s = emqxBroker.GetACL()["conf"]
	if s != "{deny, all, subscribe, [\"$SYS/#\", {eq, \"#\"}]}.\n" {
		t.Errorf("unexpected data: %s", s)
	}

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
	s = emqxEneterprise.GetACL()["conf"]
	if s != "{allow, {'and', [{user, \"admin\"}, {client, \"emqx\"}, {ipaddr, \"127.0.0.1\"}]}, pubsub, [\"$SYS/#\", \"#\"]}.\n" {
		t.Errorf("unexpected data: %s", s)
	}
}
