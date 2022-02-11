package util

import (
	"fmt"
	"strings"

	"github.com/emqx/emqx-operator/apis/apps/v1beta1"
)

func StringACL(acls []v1beta1.ACL) string {
	var s string

	for _, acl := range acls {
		who := getWho(acl)
		action := getAction(acl)
		topics := getTopics(acl)
		s = fmt.Sprintf("%s{%s, %s, %s, %s}.\n", s, acl.Permission, who, action, topics)
	}

	return s
}

func getWho(acl v1beta1.ACL) string {
	var who []string

	if acl.Username == "" && acl.ClientID == "" && acl.IPAddress == "" {
		return "all"
	}
	if acl.Username != "" {
		username := fmt.Sprintf("{user, \"%s\"}", acl.Username)
		who = append(who, username)
	}
	if acl.ClientID != "" {
		clientid := fmt.Sprintf("{client, \"%s\"}", acl.ClientID)
		who = append(who, clientid)
	}
	if acl.IPAddress != "" {
		ipaddress := fmt.Sprintf("{ipaddr, \"%s\"}", acl.IPAddress)
		who = append(who, ipaddress)
	}

	if len(who) == 0 {
		return "all"
	} else if len(who) == 1 {
		return who[0]
	} else {
		return fmt.Sprintf("{'and', [%s]}", strings.Join(who, ", "))
	}
}

func getAction(acl v1beta1.ACL) string {
	if acl.Action == "" {
		return "pubsub"
	} else {
		return acl.Action
	}
}

func getTopics(acl v1beta1.ACL) string {
	var list []string
	if acl.Topics.Filter != nil {
		for _, topic := range acl.Topics.Filter {
			list = append(list, fmt.Sprintf("\"%s\"", topic))
		}
	}
	if acl.Topics.Equal != nil {
		for _, topic := range acl.Topics.Equal {
			list = append(list, fmt.Sprintf("{eq, \"%s\"}", topic))
		}
	}

	if len(list) == 0 {
		return `["#"]`
	} else {
		return fmt.Sprintf("[%s]", strings.Join(list, ", "))
	}
}
