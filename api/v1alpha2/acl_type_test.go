package v1alpha2_test

import (
	"testing"

	"github.com/emqx/emqx-operator/api/v1alpha2"
)

func TestGenerateACL(t *testing.T) {
	var emqxBroker v1alpha2.EmqxBroker
	var emqxEneterprise v1alpha2.EmqxEnterprise
	var acl v1alpha2.ACL
	var s string

	acl = v1alpha2.ACL{Permission: "allow"}
	emqxBroker = v1alpha2.EmqxBroker{
		Spec: v1alpha2.EmqxBrokerSpec{
			ACL: []v1alpha2.ACL{
				acl,
			},
		},
	}

	s = emqxBroker.GetACL()["conf"]
	if s != "{allow, all, pubsub, [\"#\"]}.\n" {
		t.Errorf("unexpected data: %s", s)
	}

	acl = v1alpha2.ACL{
		Permission: "deny",
		Action:     "subscribe",
		Topics: v1alpha2.Topics{
			Filter: []string{"$SYS/#"},
			Equal:  []string{"#"},
		},
	}
	emqxBroker = v1alpha2.EmqxBroker{
		Spec: v1alpha2.EmqxBrokerSpec{
			ACL: []v1alpha2.ACL{
				acl,
			},
		},
	}
	s = emqxBroker.GetACL()["conf"]
	if s != "{deny, all, subscribe, [\"$SYS/#\", {eq, \"#\"}]}.\n" {
		t.Errorf("unexpected data: %s", s)
	}

	acl = v1alpha2.ACL{
		Permission: "allow",
		Username:   "admin",
		ClientID:   "emqx",
		IPAddress:  "127.0.0.1",
		Topics: v1alpha2.Topics{
			Filter: []string{
				"$SYS/#",
				"#",
			},
		},
	}
	emqxEneterprise = v1alpha2.EmqxEnterprise{
		Spec: v1alpha2.EmqxEnterpriseSpec{
			ACL: []v1alpha2.ACL{
				acl,
			},
		},
	}
	s = emqxEneterprise.GetACL()["conf"]
	if s != "{allow, {'and', [{user, \"admin\"}, {client, \"emqx\"}, {ipaddr, \"127.0.0.1\"}]}, pubsub, [\"$SYS/#\", \"#\"]}.\n" {
		t.Errorf("unexpected data: %s", s)
	}
}
