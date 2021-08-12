package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:validation:Optional
type Zone struct {
	ZoneExternal ZoneExternal `json:"external,omitempty"`
	ZoneInternal ZoneInternal `json:"internal,omitempty"`
}

type ZoneExternal struct {
	ZoneCommon ZoneCommon `json:",omitempty"`

	//+kubebuilder:default:="15s"
	IdleTimeout metav1.Duration `json:"idle_timeout,omitempty"`

	//+kubebuilder:default:=on
	//+kubebuilder:validation:Enum:=on;off
	EnableBan string `json:"enable_ban,omitempty"`

	// TODO
	MaxPacketSize resource.Quantity `json:"max_packet_size,omitempty"`

	// TODO
	MaxClientIdLen MaxClientIdLen `json:"max_clientid_len,omitempty"`

	// TODO
	MaxTopicLevels MaxTopicLevels `json:"max_topic_levels,omitempty"`

	// TODO
	//+kubebuilder:validation:Enum:=0;1;2
	MaxQosAllowed MaxQosAllowed `json:"max_qos_allowed,omitempty"`

	// TODO
	MaxTopicAlias MaxTopicAlias `json:"max_topic_alias,omitempty"`

	// TODO
	RetainAvailable RetainAvailable `json:"retain_available,omitempty"`

	// TODO
	ServerKeepalive int64 `json:"server_keepalive,omitempty"`

	//+kubebuilder:default:="0.75"
	//https://github.com/kubernetes-sigs/controller-tools/issues/245
	// TODO
	KeepaliveBackoff resource.Quantity `json:"keepalive_backoff,omitempty"`

	//+kubebuilder:default:=off
	//+kubebuilder:validation:Enum:=on;off
	UpgradeQos UpgradeQos `json:"upgrade_qos,omitempty"`

	//+kubebuilder:default:="15s"
	RetryInterval RetryInterval `json:"retry_interval,omitempty"`

	//+kubebuilder:default:="300s"
	AwaitRelTimeout metav1.Duration `json:"await_rel_timeout,omitempty"`

	//+kubebuilder:default:="2h"
	SessionExpiryInterval metav1.Duration `json:"session_expiry_interval,omitempty"`

	//+kubebuilder:default:=none
	MqueuePriorities string `json:"mqueue_priorities,omitempty"`

	//+kubebuilder:default:=highest
	MqueueDefaultPriority string `json:"mqueue_default_priority,omitempty"`

	//+kubebuilder:default:=False
	//+kubebuilder:validation:Enum:=True;False
	UseUsernameAsClientid bool `json:"use_username_as_clientid,omitempty"`
}

type ZoneInternal struct {
	ZoneCommon ZoneCommon `json:",omitempty"`

	//+kubebuilder:default:=True
	//+kubebuilder:validation:Enum:=True;False
	AllowAnonymous AllowAnonymous `json:"allow_anonymous,omitempty"`

	//+kubebuilder:default:=True
	//+kubebuilder:validation:Enum:=True;False
	BypassAuthPlugins bool `json:"bypass_auth_plugins,omitempty"`
}

type ZoneCommon struct {

	//+kubebuilder:default:=on
	//+kubebuilder:validation:Enum:=on;off
	EnableAcl EnableAcl `json:"enable_acl,omitempty"`

	//+kubebuilder:default:=on
	//+kubebuilder:validation:Enum:=on;off
	EnableStats EnableStats `json:"enable_stats,omitempty"`

	//+kubebuilder:default:=ignore
	//+kubebuilder:validation:Enum:=ignore;disconnect
	AclDenyAction AclDenyAction `json:"acl_deny_action,omitempty"`

	//+kubebuilder:default:="16000|16MB"
	// TODO
	ForceGCPolicy ForceGCPolicy `json:"force_gc_policy,omitempty"`

	// TODO
	ForceShutdownPolicy ForceShutdownPolicy `json:"force_shutdown_policy,omitempty"`

	// TODO
	WildcardSubscription WildcardSubscription `json:"wildcard_subscription,omitempty"`

	// TODO
	SharedSubscription SharedSubscription `json:"shared_subscription,omitempty"`

	//+kubebuilder:default:=0
	MaxSubscriptions MaxSubscriptions `json:"max_subscriptions,omitempty"`

	//+kubebuilder:default:=32
	MaxInflight MaxInflight `json:"max_inflight,omitempty"`

	//+kubebuilder:default:=100
	MaxAwaitingRel MaxAwaitingRel `json:"max_awaiting_rel,omitempty"`

	//+kubebuilder:default:=1000
	MaxMqueueLen MaxMqueueLen `json:"max_mqueue_len,omitempty"`

	//+kubebuilder:default:=True
	//+kubebuilder:validation:Enum:=True;False
	MqueueStoreQos0 MqueueStoreQos0 `json:"mqueue_store_qos0,omitempty"`

	//+kubebuilder:default:=off
	//+kubebuilder:validation:Enum:=on;off
	EnableFlappingDetect EnableFlappingDetect `json:"enable_flapping_detect,omitempty"`

	// TODO
	MountPoint MountPoint `json:"mount_point,omitempty"`

	//+kubebuilder:default:=False
	//+kubebuilder:validation:Enum:=True;False
	IgnoreLoopDeliver IgnoreLoopDeliver `json:"ignore_loop_deliver,omitempty"`

	//+kubebuilder:default:=False
	//+kubebuilder:validation:Enum:=True;False
	StrictMode StrictMode `json:"strict_mode,omitempty"`
}

type UpgradeQos string

const (
	UPGRADE_QOS_ON  UpgradeQos = "on"
	UPGRADE_QOS_OFF UpgradeQos = "off"
)
