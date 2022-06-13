package v1beta2

import (
	"fmt"
	"strings"
)

//+kubebuilder:object:generate=true
type Topics struct {
	Filter []string `json:"filter,omitempty"`
	Equal  []string `json:"equal,omitempty"`
}

//+kubebuilder:object:generate=true
type ACL struct {
	//+kubebuilder:validation:Enum=allow;deny
	//+kubebuilder:validation:Required
	Permission string `json:"permission"`
	Username   string `json:"username,omitempty"`
	ClientID   string `json:"clientid,omitempty"`
	IPAddress  string `json:"ipaddress,omitempty"`
	//+kubebuilder:validation:Enum=publish;subscribe
	Action string `json:"action,omitempty"`
	Topics Topics `json:"topics,omitempty"`
}

//+kubebuilder:object:generate=false
type ACLList struct {
	Items []ACL
}

func (list *ACLList) Default() {
	list.Items = []ACL{
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

func (list *ACLList) Strings() []string {
	var s []string

	for _, acl := range list.Items {
		who := getWho(acl)
		action := getAction(acl)
		topics := getTopics(acl)
		s = append(s, fmt.Sprintf("{%s, %s, %s, %s}.\n", acl.Permission, who, action, topics))
	}

	return s
}

func (list *ACLList) String() string {
	var s string

	for _, acl := range list.Items {
		who := getWho(acl)
		action := getAction(acl)
		topics := getTopics(acl)
		s = fmt.Sprintf("%s{%s, %s, %s, %s}.\n", s, acl.Permission, who, action, topics)
	}

	return s
}

func getWho(acl ACL) string {
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
