package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:validation:Optional
type RPC struct {

	//+kubebuilder:default:=async
	Mode Mode `json:"mode,omitempty"`

	// Only applies to the mode of async
	//+kubebuilder:default:=256
	AsyncBatchsize int32 `json:"async_batch_size,omitempty"`

	//+kubebuilder:validation:Enum=manual;stateless
	PortDiscovery PortDiscovery `json:"port_discovery,omitempty"`

	//+kubebuilder:default:=5369
	//+kubebuilder:validation:Minimum:=1024
	//+kubebuilder:validation:Maximum:=65535
	TcpServerPort int32 `json:"tcp_server_port,omitempty"`

	// TODO
	//+kubebuilder:validation:Minimum:=1
	//+kubebuilder:validation:Maximum:=256
	TcpClientNum int32 `json:"tcp_client_num,omitempty"`

	//+kubebuilder:default:="5s"
	ConnectTimeout metav1.Duration `json:"connect_timeout,omitempty"`

	//+kubebuilder:default:="5s"
	SendTimeout metav1.Duration `json:"send_timeout,omitempty"`

	//+kubebuilder:default:="5s"
	AuthenticationTimeout metav1.Duration `json:"authentication_timeout,omitempty"`

	//+kubebuilder:default:="15s"
	CallReceiveTimeout metav1.Duration `json:"call_receieve_timeout,omitempty"`

	//+kubebuilder:default:="900s"
	SocketKeepaliveIdle metav1.Duration `json:"socket_keepalive_idel,omitempty"`

	//+kubebuilder:default:="75s"
	SocketkeepaliveInterval metav1.Duration `json:"socket_keepalive_interval,omitempty"`

	//+kubebuilder:default:=9
	SocketKeepaliveCount int8 `json:"socket_keepalive_count,omitempty"`

	//+kubebuilder:default:="1MB"
	SocketSndbuf resource.Quantity `json:"socket_sndbuf,omitempty"`

	//+kubebuilder:default:="1MB"
	SocketRecbuf resource.Quantity `json:"socket_recbuf,omitempty"`

	//+kubebuilder:default:="1MB"
	SocketBuffer resource.Quantity `json:"socket_buffer,omitempty"`
}

type Mode string

const (
	MODE_SYNC  Mode = "SYNC"
	MODE_ASYNC Mode = "ASYNC"
)

type PortDiscovery string

const (
	PORT_DISCOVERY_MANUAL    PortDiscovery = "munal"
	PORT_DISCOVERY_STATELESS PortDiscovery = "stateless"
)
