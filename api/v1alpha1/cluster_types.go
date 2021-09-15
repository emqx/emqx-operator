package v1alpha1

import (
	"fmt"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:validation:Optional
// Cluster defines the desired state of Cluster
type Cluster struct {
	// The name of cluster
	//+kubebuilder:default:=emqxcl
	Name string `json:"name,omitempty"`

	// The protocl of cluster about erlang
	//+kubebuilder:default:=inet_tcp
	//+kubebuilder:validation:Enum=inet_tcp;inet6_tcp;inet_tls
	ProtoDist ProtoDist `json:"proto_dist,omitempty"`

	// The mode of cluster discovery
	//+kubebuilder:default:=k8s
	//+kubebuilder:validation:Enum=dns;k8s
	Discovery Discovery `json:"discovery,omitempty"`

	// The mechanism about turning on/off of
	//+kubebuilder:default:=on
	//+kubebuilder:validation:Enum=on;off
	// TODO
	Autoheal Autoheal `json:"autoheal,omitempty"`

	// Defines how long to remove the stub-node from cluster
	//+kubebuilder:default:="5m"
	Autoclean metav1.Duration `json:"autoclean,omitempty"`

	// Only applies to the mode of cluster discovery is K8S
	K8S K8S `json:"k8s,omitempty"`
}

type ProtoDist string

const (
	PROTO_DIST_INET_TCP  ProtoDist = "inet_tcp"
	PROTO_DIST_INET6_TCP ProtoDist = "inet6_tcp"
	//TODO
	PROTO_DIST_INET_TLS ProtoDist = "inet_tls"
)

type Discovery string

const (
	DISCOVERY_DNS Discovery = "dns"
	DISCOVERY_K8S Discovery = "k8s"
)

type Autoheal string

const (
	AUTOHEAL_ON  Autoheal = "on"
	AUTOHEAL_OFF Autoheal = "off"
)

type DNS struct {
	// Example "mycluster.com"
	Name string `json:"name,omitempty"`
	// Example "emqx"
	App string `json:"app,omitempty"`
}

type K8S struct {
	// TODO validation
	// Example "http://10.110.111.204:8080"
	Apiserver string `json:"apiserver,omitempty"`

	// Example "emqx"
	ServiceName string `json:"service_name,omitempty"`

	// The address type is used to extract host from k8s service.
	// Note: Hostname is only supported after v4.0-rc.2
	//+kubebuilder:default:=hostname
	//+kubebuilder:validation:Enum=ip;dns;hostname
	AddressType AddressType `json:"address_type,omitempty"`

	// Example "emqx"
	AppName string `json:"app_name,omitempty"`

	// Example "pod.cluster.local"
	Suffix string `json:"suffix,omitempty"`

	// Example "default"
	Namespace string `json:"namespace,omitempty"`
}

type AddressType string

const ENV_CULSTER_PREFIX = "emqx_cluster"

const (
	ADDRESS_TYPE_IP       AddressType = "ip"
	ADDRESS_TYPE_DNS      AddressType = "dns"
	ADDRESS_TYPE_HOSTNAME AddressType = "hostname"
)

func (c Cluster) ConvertToEnv() []corev1.EnvVar {
	var envVar []corev1.EnvVar
	var clusterMap = make(map[string]string)
	t := reflect.TypeOf(c)
	v := reflect.ValueOf(c)
	m := structToEnvHelper(0, ENV_CULSTER_PREFIX, t, v, clusterMap)
	for k, v := range m {
		// aim to skip the v is null
		if len(v) != 0 {
			envVar = append(envVar, corev1.EnvVar{
				Name:  k,
				Value: v,
			})
		}
	}
	return envVar
}

func structToEnvHelper(i int, k string, t reflect.Type, v reflect.Value, m map[string]string) map[string]string {
	if v.Kind() != reflect.Struct {
		m[strings.ToUpper(k)] = fmt.Sprintf("%v", v)
		return m
	}
	for i := 0; i < t.NumField(); i++ {
		key := fmt.Sprintf("%s__%s", k, strings.Split(t.Field(i).Tag.Get("json"), ",")[0])
		structToEnvHelper(i, key, v.Field(i).Type(), v.Field(i), m)
	}
	return m
}
