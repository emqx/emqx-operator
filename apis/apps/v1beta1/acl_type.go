package v1beta1

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
