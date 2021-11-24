package util_test

import (
	"testing"

	"github.com/emqx/emqx-operator/pkg/util"
)

func TestGenACL(t *testing.T) {
	var acl util.ACL
	var s string

	acl = util.ACL{Permission: "allow"}
	s = util.GenACL([]util.ACL{acl})
	if s != "{allow, all, pubsub, [\"#\"]}.\n" {
		t.Errorf("unexpected data: %s", s)
	}

	acl = util.ACL{
		Permission: "deny",
		Action:     "subscribe",
		Topics: util.Topics{
			Filter: []string{"$SYS/#"},
			Equal:  []string{"#"},
		},
	}
	s = util.GenACL([]util.ACL{acl})
	if s != "{deny, all, subscribe, [\"$SYS/#\", {eq, \"#\"}]}.\n" {
		t.Errorf("unexpected data: %s", s)
	}

	acl = util.ACL{
		Permission: "allow",
		Username:   "admin",
		ClientID:   "emqx",
		IPAddress:  "127.0.0.1",
		Topics: util.Topics{
			Filter: []string{
				"$SYS/#",
				"#",
			},
		},
	}
	s = util.GenACL([]util.ACL{acl})
	if s != "{allow, {'and', [{user, \"admin\"}, {client, \"emqx\"}, {ipaddr, \"127.0.0.1\"}]}, pubsub, [\"$SYS/#\", \"#\"]}.\n" {
		t.Errorf("unexpected data: %s", s)
	}
}
