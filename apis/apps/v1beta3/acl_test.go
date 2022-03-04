package v1beta3_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
)

func TestACLDefault(t *testing.T) {
	acls := &v1beta3.ACLList{}
	acls.Default()

	assert.ElementsMatch(t, acls.Items,
		[]v1beta3.ACL{
			{
				Permission: "allow",
				Username:   "dashboard",
				Action:     "subscribe",
				Topics: v1beta3.Topics{
					Filter: []string{
						"$STS?#",
					},
				},
			},
			{
				Permission: "allow",
				IPAddress:  "127.0.0.1",
				Topics: v1beta3.Topics{
					Filter: []string{
						"$SYS/#",
						"#",
					},
				},
			},
			{
				Permission: "deny",
				Action:     "subscribe",
				Topics: v1beta3.Topics{
					Filter: []string{"$SYS/#"},
					Equal:  []string{"#"},
				},
			},
			{
				Permission: "allow",
			},
		},
	)
}

func TestACLString(t *testing.T) {
	var acl v1beta3.ACL
	var acls *v1beta3.ACLList

	acl = v1beta3.ACL{Permission: "allow"}
	acls = &v1beta3.ACLList{
		Items: []v1beta3.ACL{acl},
	}

	assert.Equal(t,
		acls.String(),
		"{allow, all, pubsub, [\"#\"]}.\n",
	)

	acl = v1beta3.ACL{
		Permission: "deny",
		Action:     "subscribe",
		Topics: v1beta3.Topics{
			Filter: []string{"$SYS/#"},
			Equal:  []string{"#"},
		},
	}
	acls = &v1beta3.ACLList{
		Items: []v1beta3.ACL{acl},
	}

	assert.Equal(t,
		acls.String(),
		"{deny, all, subscribe, [\"$SYS/#\", {eq, \"#\"}]}.\n",
	)

	acl = v1beta3.ACL{
		Permission: "allow",
		Username:   "admin",
		ClientID:   "emqx",
		IPAddress:  "127.0.0.1",
		Topics: v1beta3.Topics{
			Filter: []string{
				"$SYS/#",
				"#",
			},
		},
	}
	acls = &v1beta3.ACLList{
		Items: []v1beta3.ACL{acl},
	}

	assert.Equal(t,
		acls.String(),
		"{allow, {'and', [{user, \"admin\"}, {client, \"emqx\"}, {ipaddr, \"127.0.0.1\"}]}, pubsub, [\"$SYS/#\", \"#\"]}.\n",
	)
}
