package v1alpha1

import "k8s.io/apimachinery/pkg/api/resource"

//+kubebuilder:validation:Optional
type Mqtt struct {
	//+kubebuilder:default:="30, 1m, 5m"
	// TODO validation pattern
	FlappingDetectPolicy string `json:"flapping_detect_policy,omitempty"`

	//+kubebuilder:default:="1MB"
	MaxPacketSize resource.Quantity `json:"max_packet_size,omitempty"`

	//+kubebuilder:default:=65535
	MaxClientIdLen MaxClientIdLen `json:"max_clientid_len,omitempty"`

	//+kubebuilder:default:=0
	MaxTopicLevels MaxTopicLevels `json:"max_topic_levels,omitempty"`

	//+kubebuilder:default:=2
	//+kubebuilder:validation:Enum:=0;1;2
	MaxQosAllowed MaxQosAllowed `json:"max_qos_allowed,omitempty"`

	//+kubebuilder:default:=65535
	MaxTopicAlias MaxTopicAlias `json:"max_topic_alias,omitempty"`

	//+kubebuilder:default:=True
	//+kubebuilder:validation:Enum:=True;False
	RetainAvailable RetainAvailable `json:"retain_available,omitempty"`

	//+kubebuilder:default:=True
	//+kubebuilder:validation:Enum:=True;False
	WildcardSubscription WildcardSubscription `json:"wildcard_subscription,omitempty"`

	//+kubebuilder:default:=True
	//+kubebuilder:validation:Enum:=True;False
	SharedSubscription SharedSubscription `json:"shared_subscription,omitempty"`

	//+kubebuilder:default:=False
	//+kubebuilder:validation:Enum:=True;False
	IgnoreLoopDeliver IgnoreLoopDeliver `json:"ignore_loop_deliver,omitempty"`

	//+kubebuilder:default:=False
	//+kubebuilder:validation:Enum:=True;False
	StrictMode StrictMode `json:"strict_mode,omitempty"`
}
