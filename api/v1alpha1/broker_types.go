package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

//+kubebuilder:validation:Optional
type Broker struct {

	//+kubebuilder:default:="1m"
	SysInterval metav1.Duration `json:"sys_interval,omitempty"`

	//+kubebuilder:default:="30s"
	SysHeartbeat metav1.Duration `json:"sys_heartbeat,omitempty,omitempty"`

	//+kubebuilder:default:=on
	//+kubebuilder:validation:Enum:=on;off
	EnableSessionRegistry EnableSessionRegistry `json:"enable_session_registry     ,omitempty"`

	//+kubebuilder:default:=quorum
	//+kubebuilder:validation:Enum:=local;leader;quorum;all
	SessionLockingStrategy EnableSessionRegistry `json:"session_locking_strategy    ,omitempty"`

	//+kubebuilder:default:=random
	//+kubebuilder:validation:Enum=random;round_robin;sticky;hash_clientid;hash_topic
	SharedSubscriptionStrategy SharedSubscriptionStrategy `json:"shared_subscription_strategy,omitempty"`

	//+kubebuilder:default:=False
	SharedDispatchAckEnabled bool `json:"shared_dispatch_ack_enabled ,omitempty"`

	//+kubebuilder:default:=off
	//+kubebuilder:validation:Enum:=on;off
	RouteBatchClean RouteBatchClean `json:"route_batch_clean,omitempty"`

	Perf Perf `json:"perf,omitempty"`
}

type EnableSessionRegistry string

const (
	ENABLE_SESSION_REGISTRY_ON  EnableSessionRegistry = "on"
	ENABLE_SESSION_REGISTRY_OFF EnableSessionRegistry = "off"
)

type SessionLockingStrategy string

const (
	SESSION_LOCKING_STRATEGY_LOCAL  SessionLockingStrategy = "local"
	SESSION_LOCKING_STRATEGY_LEADER SessionLockingStrategy = "leader"
	SESSION_LOCKING_STRATEGY_QUORUM SessionLockingStrategy = "quorum"
	SESSION_LOCKING_STRATEGY_ALL    SessionLockingStrategy = "all	"
)

type SharedSubscriptionStrategy string

const (
	SHARED_SUBSCRIPTION_STRATEGY_RANDOM        SharedSubscriptionStrategy = "random"
	SHARED_SUBSCRIPTION_STRATEGY_ROUND_ROBIN   SharedSubscriptionStrategy = "round_robin"
	SHARED_SUBSCRIPTION_STRATEGY_STICKY        SharedSubscriptionStrategy = "sticky"
	SHARED_SUBSCRIPTION_STRATEGY_HASH_CLIENTID SharedSubscriptionStrategy = "hash_clientid"
	SHARED_SUBSCRIPTION_STRATEGY_HASH_TOPIC    SharedSubscriptionStrategy = "hash_topic"
)

type RouteBatchClean string

const (
	ROUTE_BATCH_CLEAN_ON  RouteBatchClean = "on"
	ROUTE_BATCH_CLEAN_OFF RouteBatchClean = "off"
)

type Perf struct {
	//+kubebuilder:default:=key
	//+kubebuilder:validation:Enum:=key;tab;global
	RouteLockType RouteLockType `json:"route_lock_type,omitempty"`

	//+kubebuilder:default:=True
	TrieCompaction bool `json:"trie_compaction,omitempty"`
}

type RouteLockType string

const (
	ROUTE_LOCK_TYPE_KEY    RouteLockType = "key"
	ROUTE_LOCK_TYPE_TAB    RouteLockType = "tab"
	ROUTE_LOCK_TYPE_GLOBAL RouteLockType = "global"
)
