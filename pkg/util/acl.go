package util

import (
	"fmt"
	"strings"
)

type Topics struct {
	Filter []string `json:"filter,omitempty"`
	Equal  []string `json:"equal,omitempty"`
}

type ACL struct {
	//+kubebuilder:validation:Enum=allow;deny
	Permission string `json:"permission"`
	Username   string `json:"username,omitempty"`
	ClientID   string `json:"clientid,omitempty"`
	IPAddress  string `json:ipaddress,omitempty`
	//+kubebuilder:validation:Enum=publish;subscribe
	Action string `json:action,omitempty`
	Topics Topics `json: topics,omitempty`
}

func GenACL(acls []ACL) string {
	var s string
	if acls == nil {
		acls = defaultACL()
	}
	for _, acl := range acls {
		who := getWho(acl)
		action := getAction(acl)
		topics := getTopics(acl)
		s = fmt.Sprintf("%s{%s, %s, %s, %s}.\n", s, acl.Permission, who, action, topics)
	}

	return s
}

func getWho(acl ACL) string {
	var whos []string

	if acl.Username == "" && acl.ClientID == "" && acl.IPAddress == "" {
		return "all"
	}
	if acl.Username != "" {
		username := fmt.Sprintf("{user, \"%s\"}", acl.Username)
		whos = append(whos, username)
	}
	if acl.ClientID != "" {
		clientid := fmt.Sprintf("{client, \"%s\"}", acl.ClientID)
		whos = append(whos, clientid)
	}
	if acl.IPAddress != "" {
		ipaddress := fmt.Sprintf("{ipaddr, \"%s\"}", acl.IPAddress)
		whos = append(whos, ipaddress)
	}

	if len(whos) == 0 {
		return "all"
	} else if len(whos) == 1 {
		return whos[0]
	} else {
		return fmt.Sprintf("{'and', [%s]}", strings.Join(whos, ", "))
	}
}

func getAction(acl ACL) string {
	if acl.Action == "" {
		return "pubsub"
	} else {
		return acl.Action
	}
}

func getTopics(acl ACL) string {
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

func defaultACL() []ACL {
	return []ACL{
		{
			Permission: "allow",
			Username:   "dashboard",
			Action:     "subscribe",
			Topics: Topics{
				Filter: []string{
					"$STS?#",
				},
			},
		},
		{
			Permission: "allow",
			IPAddress:  "127.0.0.1",
			Topics: Topics{
				Filter: []string{
					"$SYS/#",
					"#",
				},
			},
		},
		{
			Permission: "deny",
			Action:     "subscribe",
			Topics: Topics{
				Filter: []string{"$SYS/#"},
				Equal:  []string{"#"},
			},
		},
		{
			Permission: "allow",
		},
	}
}
