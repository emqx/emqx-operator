package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:validation:Optional

type Node struct {
	// Name is the name of the node
	//+kubebuilder:default:="emqx@127.0.0.1"
	//+kubebuilder:validation:Pattern:=*@*
	Name string `json:"name,omitempty"`

	// Cookie for the erlang cluster
	//+kubebuilder:default:="emqxsecretcookie"
	Cookie string `json:"cookie,omitempty"`

	// The directory of data
	// TODO
	//+kubebuilder:default:="./data"
	DataDir string `json:"data_dir,omitempty"`

	//+kubebuilder:default:=off
	//+kubebuilder:validation:Enum=on;off
	Heartbeat Heartbeat `json:"heartbeat,omitempty"`

	//+kubebuilder:default:=4
	//+kubebuilder:validation:Minimum:=0
	//+kubebuilder:validation:Maximum:=1024
	AsyncTthreads int32 `json:"async_tthreads,omitempty"`

	//+kubebuilder:default:=2097152
	//+kubebuilder:validation:Minimum:=1024
	//+kubebuilder:validation:Maximum:Maximum=134217727
	ProcessLimit int64 `json:"process_limit,omitempty"`

	//+kubebuilder:default:=1048576
	//+kubebuilder:validation:Minimum=1024
	//+kubebuilder:validation:Maximum=134217727
	MaxPorts int64 `json:"max_ports,omitempty"`

	//+kubebuilder:default:="8MB"
	// TODO 1KB~2GB
	DistBufferSize resource.Quantity `json:"dist_buffer_size,omitempty"`

	//+kubebuilder:default:=262144
	MaxEtsTables int32 `json:"max_ets_tables,omitempty"`

	//+kubebuilder:default:="15m"
	GlobalGCInterval metav1.Duration `json:"global_gc_interval"`

	//+kubebuilder:default:=1000
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:Maximum=65535
	FullsweepAfter int32 `json:"fullsweep_after"`

	//+kubebuilder:default:=log/crash.dump
	CrashDump string `json:"crash_dump,omitempty"`

	//+kubebuilder:default:=etc/ssl_dist.conf
	SSLDistOptfile string `json:"ssl_dist_optfile"`

	//+kubebuilder:default:=120
	DistNetTicktime int32 `json:"dist_net_ticktime"`

	//+kubebuilder:default:=6369
	//+kubebuilder:validation:Minimum=1024
	//+kubebuilder:validation:Maximum=65535
	DistListenMin int32 `json:"dist_listen_min,omitempty"`

	//+kubebuilder:default:=6369
	//+kubebuilder:validation:Minimum=1024
	//+kubebuilder:validation:Maximum=65535
	DistListenMax int32 `json:"dist_listen_max,omitempty"`
}

type Heartbeat string

const (
	HEARTBEAT_ON  Heartbeat = "on"
	HEARTBEAT_OFF Heartbeat = "off"
)
