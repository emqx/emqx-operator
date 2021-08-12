package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

//+kubebuilder:validation:Optional
type AuthAcl struct {

	//+kubebuilder:default:=True
	//+kubebuilder:validation:Enum:=True;False
	AllowAnonymous AllowAnonymous `json:"allow_anonymous,omitempty"`

	//+kubebuilder:default:=allow
	//+kubebuilder:validation:Enum:=allow;deny
	AclNomatch AclNomatch `json:"acl_nomatch,omitempty"`

	//+kubebuilder:default:=etc/acl.conf
	AclFile string `json:"acl_file,omitempty"`

	//+kubebuilder:default:=on
	//+kubebuilder:validation:Enum:=on;off
	EnableAclCache EnableAclCache `json:"enable_acl_cache,omitempty"`

	//+kubebuilder:default:=32
	AclCacheMaxSize int8 `json:"acl_cache_max_size,omitempty"`

	//+kubebuilder:default:="1m"
	AclCacheTtl metav1.Duration `json:"acl_cache_ttl,omitempty"`

	//+kubebuilder:default:=ignore
	//+kubebuilder:validation:Enum:=ignore;disconnect
	AclDenyAction AclDenyAction `json:"acl_deny_action,omitempty"`
}

type AclNomatch string

const (
	ACL_NOMATCH_ALLOW AclNomatch = "allow"
	ACL_NOMATCH_DENY  AclNomatch = "deny"
)

type EnableAclCache string

const (
	ENABLE_ACL_CACHE_ON  EnableAclCache = "on"
	ENABLE_ACL_CACHE_OFF EnableAclCache = "off"
)
