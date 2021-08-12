package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AllowAnonymous bool

type AclDenyAction string

const (
	ACL_DENY_ACTION_IGNORE     AclDenyAction = "ignore"
	ACL_DENY_ACTION_DISCONNECT AclDenyAction = "disconnect"
)

// type MaxPacketSize resource.Quantity

type MaxClientIdLen int64

type MaxTopicLevels int32

type MaxQosAllowed int8

const (
	MAX_QOS_ALLOWED_ZERO MaxQosAllowed = 0
	MAX_QOS_ALLOWED_ONE  MaxQosAllowed = 1
	MAX_QOS_ALLOWED_TWO  MaxQosAllowed = 2
)

type MaxTopicAlias int64

type RetainAvailable bool

type WildcardSubscription bool

type SharedSubscription bool

type IgnoreLoopDeliver bool

type StrictMode bool

type EnableAcl string

type EnableStats string

type ForceGCPolicy string

type ForceShutdownPolicy string

type MaxSubscriptions int64

type MaxInflight int64

type MaxAwaitingRel int64

type MaxMqueueLen int64

type MqueueStoreQos0 bool

type EnableFlappingDetect string

const (
	ENABLE_FLAPPING_DETECT_ON  EnableFlappingDetect = "on"
	ENABLE_FLAPPING_DETECT_OFF EnableFlappingDetect = "off"
)

type MountPoint string

type RetryInterval metav1.Duration

type Acceptors int64

type MaxConnections int64

type MaxConnRate int64

type ActiveN int8

type ListenerZone string

type RateLimite string

type Backlog uint32

type SendTimeout metav1.Duration

type SendTimeoutClose string

type ProxyProtocol string

const (
	PROXY_PROTOCOL_ON  ProxyProtocol = "on"
	PROXY_PROTOCOL_OFF ProxyProtocol = "off"
)

type TuneBuffer string

const (
	TUNE_BUFFER_ON  TuneBuffer = "on"
	TUNE_BUFFER_OFF TuneBuffer = "off"
)

type NodeDelay bool

type ReuseAddr bool

type PeerCertAsUsername string

const (
	PEER_CERT_AS_USERNAME_CN  PeerCertAsUsername = "cn"
	PEER_CERT_AS_USERNAME_DN  PeerCertAsUsername = "dn"
	PEER_CERT_AS_USERNAME_CRT PeerCertAsUsername = "crt"
	PEER_CERT_AS_USERNAME_PEM PeerCertAsUsername = "pem"
	PEER_CERT_AS_USERNAME_md5 PeerCertAsUsername = "md5"
)

type PeerCertAsClientid string

const (
	PEER_CERT_AS_CLIENTID_CN  PeerCertAsClientid = "cn"
	PEER_CERT_AS_CLIENTID_DN  PeerCertAsClientid = "dn"
	PEER_CERT_AS_CLIENTID_CRT PeerCertAsClientid = "crt"
	PEER_CERT_AS_CLIENTID_PEM PeerCertAsClientid = "pem"
	PEER_CERT_AS_CLIENTID_md5 PeerCertAsClientid = "md5"
)

type TlsVersions string

type HandShakeTimeout metav1.Duration

type Depth uint8

type KeyPassword string

type KeyFile string

type CertFile string

type CaCertFile string

type DhFile string

type Verify string

const (
	VERIFY_PEER Verify = "verify_peer"
	VERIFY_NONE Verify = "verify_none"
)

type FailIfNoPeerCert bool

type PskCiphers string

type SecureRenegotiate string

const (
	SECURE_RENEGOTIATE_ON  SecureRenegotiate = "on"
	SECURE_RENEGOTIATE_OFF SecureRenegotiate = "off"
)

type ReuseSessions string

const (
	REUSE_SESSIONS_ON  ReuseSessions = "on"
	REUSE_SESSIONS_OFF ReuseSessions = "off"
)

type HonorCipherOrder string

const (
	HONOR_CIPHER_ORDER_ON  = "on"
	HONOR_CIPHER_ORDER_OFF = "off"
)

type MqttPath string

type FailIfNoSubprotocol bool

type SupportedProtocols string

type ProxyAddressHeader string

type ProxyPortHeader string

type Compress bool

type MaxFrameSize int32
