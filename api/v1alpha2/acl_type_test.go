package v1alpha2_test

import (
	"testing"

	"github.com/emqx/emqx-operator/api/v1alpha2"
)

func TestGenerateACL(t *testing.T) {
	var acl v1alpha2.ACL
	var s string

	acl = v1alpha2.ACL{Permission: "allow"}
	s = v1alpha2.GenerateACL([]v1alpha2.ACL{acl})
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
	s = v1alpha2.GenerateACL([]v1alpha2.ACL{acl})
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
	s = v1alpha2.GenerateACL([]v1alpha2.ACL{acl})
	if s != "{allow, {'and', [{user, \"admin\"}, {client, \"emqx\"}, {ipaddr, \"127.0.0.1\"}]}, pubsub, [\"$SYS/#\", \"#\"]}.\n" {
		t.Errorf("unexpected data: %s", s)
	}
}
