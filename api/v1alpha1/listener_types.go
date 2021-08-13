package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:validation:Optional
type Listener struct {
	TCP []TCP `json:"listener,omitempty"`

	SSL []SSL `json:"ssl,omitempty"`

	WS []WS `json:"ws,omitempty"`

	WSS []WSS `json:"wss,omitempty"`
}

type ListenerCommon struct {
	Name string `json:"name,omitempty"`

	//+kubebuilder:default:=8
	Acceptors Acceptors `json:"acceptors,omitempty"`

	//+kubebuilder:default:=1024000
	MaxConnections MaxConnections `json:"max_connections,omitempty"`

	//+kubebuilder:default:=1000
	MaxConnRate MaxConnRate `json:"max_conn_rate,omitempty"`

	//+kubebuilder:default:=100
	ActiveN ActiveN `json:"active_n,omitempty"`

	// TODO
	RateLimite RateLimite `json:"rate_limite,omitempty"`

	//+kubebuilder:default:=1024
	Backlog Backlog `json:"backlog,omitempty"`

	//http://erlang.org/doc/man/inet.html
	RecBuf resource.Quantity `json:"rec_buf,omitempty"`

	//http://erlang.org/doc/man/inet.html
	SndBuf resource.Quantity `json:"snd_buf,omitempty"`

	//http://erlang.org/doc/man/inet.html
	Buffer resource.Quantity `json:"buffer,omitempty"`

	//+kubebuilder:default:=True
	//+kubebuilder:validation:Enum:=True;False
	NodeDelay NodeDelay `json:"nodelay,omitempty"`

	//+kubebuilder:default:=True
	//+kubebuilder:validation:Enum=True;False
	ReuseAddr ReuseAddr `json:"reuseaddr,omitempty"`

	//+kubebuilder:default:="allow all"
	// Example "192.168.0.0/24 192.178.0.0/24"
	Access string `json:"access,omitempty"`

	// TODO default miss
	//+kubebuilder:validation:Enum:=on;off
	ProxyProtocol ProxyProtocol `json:"proxy_protocol,omitempty"`

	// TODO default miss
	ProxyProtocolTimeout metav1.Duration `json:"proxy_protocol_timeout,omitempty"`

	//+kubebuilder:default:=cn
	//+kubebuilder:validation:Enum:=cn;dn;crt;pem;md5
	PeerCertAsUsername PeerCertAsUsername `json:"peer_cert_as_username,omitempty"`

	//+kubebuilder:default:=cn
	//+kubebuilder:validation:Enum:=cn;dn;crt;pem;md5
	PeerCertAsClientid PeerCertAsClientid `json:"peer_cert_as_clientid,omitempty"`
}

type TCP struct {
	ListenerCommon ListenerCommon `json:",omitempty"`
}

type SSL struct {
	ListenerCommon ListenerCommon `json:",omitempty"`

	//+kubebuilder:default:="tlsv1.3,tlsv1.2,tlsv1.1,tlsv1"
	TlsVersions TlsVersions `json:"tls_versions,omitempty"`

	//+kubebuilder:default:="15s"
	HandShakeTimeout HandShakeTimeout `json:"handshake_timeout,omitempty"`

	//+kubebuilder:default:=10
	Depth Depth `json:"depth,omitempty"`

	KeyPassword KeyPassword `json:"key_password,omitempty"`

	//+kubebuilder:default:="etc/certs/key.pem"
	KeyFile KeyFile `json:"keyfile,omitempty"`

	//+kubebuilder:default:="etc/certs/cert.pem"
	CertFile CertFile `json:"certfile,omitempty"`

	//+kubebuilder:default:="etc/certs/cacert.pem"
	CaCertFile CaCertFile `json:"cacertfile,omitempty"`

	//+kubebuilder:default:="etc/certs/dh-params.pem"
	DhFile DhFile `json:"dhfile,omitempty"`

	//+kubebuilder:default:=verify_peer
	//+kubebuilder:validation:Enum:=verify_peer;verify_none
	Verify Verify `json:"verify,omitempty"`

	//+kubebuilder:default:=False
	//+kubebuilder:validation:Enum:=False;True
	FailIfNoPeerCert FailIfNoPeerCert `json:"fail_if_no_peer_cert,omitempty"`

	//+kubebuilder:default:="ECDHE-ECDSA-AES256-GCM-SHA384,ECDHE-RSA-AES256-GCM-SHA384,ECDHE-ECDSA-AES256-SHA384,ECDHE-RSA-AES256-SHA384,ECDHE-ECDSA-DES-CBC3-SHA,ECDH-ECDSA-AES256-GCM-SHA384,ECDH-RSA-AES256-GCM-SHA384,ECDH-ECDSA-AES256-SHA384,ECDH-RSA-AES256-SHA384,DHE-DSS-AES256-GCM-SHA384,DHE-DSS-AES256-SHA256,AES256-GCM-SHA384,AES256-SHA256,ECDHE-ECDSA-AES128-GCM-SHA256,ECDHE-RSA-AES128-GCM-SHA256,ECDHE-ECDSA-AES128-SHA256,ECDHE-RSA-AES128-SHA256,ECDH-ECDSA-AES128-GCM-SHA256,ECDH-RSA-AES128-GCM-SHA256,ECDH-ECDSA-AES128-SHA256,ECDH-RSA-AES128-SHA256,DHE-DSS-AES128-GCM-SHA256,DHE-DSS-AES128-SHA256,AES128-GCM-SHA256,AES128-SHA256,ECDHE-ECDSA-AES256-SHA,ECDHE-RSA-AES256-SHA,DHE-DSS-AES256-SHA,ECDH-ECDSA-AES256-SHA,ECDH-RSA-AES256-SHA,AES256-SHA,ECDHE-ECDSA-AES128-SHA,ECDHE-RSA-AES128-SHA,DHE-DSS-AES128-SHA,ECDH-ECDSA-AES128-SHA,ECDH-RSA-AES128-SHA,AES128-SHA"
	Ciphers string `json:"ciphers,omitempty"`

	//+kubebuilder:default:="PSK-AES128-CBC-SHA,PSK-AES256-CBC-SHA,PSK-3DES-EDE-CBC-SHA,PSK-RC4-SHA"
	PskCiphers PskCiphers `json:"psk_ciphers,omitempty"`

	//+kubebuilder:default:=off
	//+kubebuilder:validation:Enum:=on;off
	SecureRenegotiate SecureRenegotiate `json:"secure_renegotiate,omitempty"`

	//+kubebuilder:default:=on
	//+kubebuilder:validation:Enum:=on;off
	ReuseSessions ReuseSessions `json:"reuse_sessions,omitempty"`

	//+kubebuilder:default:=on
	//+kubebuilder:validation:Enum:=on;off
	HonorCipherOrder HonorCipherOrder `json:"honor_cipher_order,omitempty"`
}

type WS struct {
	ListenerCommon ListenerCommon `json:",omitempty"`

	//+kubebuilder:default:="/mqtt"
	MqttPath MqttPath `json:"mqtt_path,omitempty"`

	//+kubebuilder:default:=True
	//+kubebuilder:validation:Enum:=True;False
	FailIfNoSubprotocol FailIfNoSubprotocol `json:"fail_if_no_subprotocol,omitempty"`

	//+kubebuilder:default:="mqtt, mqtt-v3, mqtt-v3.1.1, mqtt-v5"
	SupportedProtocols SupportedProtocols `json:"supported_protocols,omitempty"`

	ProxyAddressHeader ProxyAddressHeader `json:"proxy_address_header,omitempty"`

	ProxyPortHeader ProxyPortHeader `json:"proxy_port_header,omitempty"`

	//+kubebuilder:default:=False
	//+kubebuilder:validation:Enum:=True;Flase
	Compress Compress `json:"compress,omitempty"`

	DeflateOpts DeflateOpts `json:",omitempty"`

	//+kubebuilder:default:="15s"
	IdleTimeout metav1.Duration `json:"idle_timeout,omitempty"`

	MaxFrameSize MaxFrameSize `json:"max_frame_size,omitempty"`
}

type WSS struct {
	ListenerCommon ListenerCommon `json:",omitempty"`

	//+kubebuilder:default:="tlsv1.3,tlsv1.2,tlsv1.1,tlsv1"
	TlsVersions TlsVersions `json:"tls_versions,omitempty"`

	//+kubebuilder:default:="15s"
	HandShakeTimeout HandShakeTimeout `json:"handshake_timeout,omitempty"`

	//+kubebuilder:default:=10
	Depth Depth `json:"depth,omitempty"`

	KeyPassword KeyPassword `json:"key_password,omitempty"`

	//+kubebuilder:default:="etc/certs/key.pem"
	KeyFile KeyFile `json:"keyfile,omitempty"`

	//+kubebuilder:default:="etc/certs/cert.pem"
	CertFile CertFile `json:"certfile,omitempty"`

	//+kubebuilder:default:="etc/certs/cacert.pem"
	CaCertFile CaCertFile `json:"cacertfile,omitempty"`

	//+kubebuilder:default:="etc/certs/dh-params.pem"
	DhFile DhFile `json:"dhfile,omitempty"`

	//+kubebuilder:default:=verify_peer
	//+kubebuilder:validation:Enum:=verify_peer;verify_none
	Verify Verify `json:"verify,omitempty"`

	//+kubebuilder:default:=False
	//+kubebuilder:validation:Enum:=False;True
	FailIfNoPeerCert FailIfNoPeerCert `json:"fail_if_no_peer_cert,omitempty"`

	//+kubebuilder:default:="ECDHE-ECDSA-AES256-GCM-SHA384,ECDHE-RSA-AES256-GCM-SHA384,ECDHE-ECDSA-AES256-SHA384,ECDHE-RSA-AES256-SHA384,ECDHE-ECDSA-DES-CBC3-SHA,ECDH-ECDSA-AES256-GCM-SHA384,ECDH-RSA-AES256-GCM-SHA384,ECDH-ECDSA-AES256-SHA384,ECDH-RSA-AES256-SHA384,DHE-DSS-AES256-GCM-SHA384,DHE-DSS-AES256-SHA256,AES256-GCM-SHA384,AES256-SHA256,ECDHE-ECDSA-AES128-GCM-SHA256,ECDHE-RSA-AES128-GCM-SHA256,ECDHE-ECDSA-AES128-SHA256,ECDHE-RSA-AES128-SHA256,ECDH-ECDSA-AES128-GCM-SHA256,ECDH-RSA-AES128-GCM-SHA256,ECDH-ECDSA-AES128-SHA256,ECDH-RSA-AES128-SHA256,DHE-DSS-AES128-GCM-SHA256,DHE-DSS-AES128-SHA256,AES128-GCM-SHA256,AES128-SHA256,ECDHE-ECDSA-AES256-SHA,ECDHE-RSA-AES256-SHA,DHE-DSS-AES256-SHA,ECDH-ECDSA-AES256-SHA,ECDH-RSA-AES256-SHA,AES256-SHA,ECDHE-ECDSA-AES128-SHA,ECDHE-RSA-AES128-SHA,DHE-DSS-AES128-SHA,ECDH-ECDSA-AES128-SHA,ECDH-RSA-AES128-SHA,AES128-SHA"
	Ciphers string `json:"ciphers,omitempty"`

	//+kubebuilder:default:="PSK-AES128-CBC-SHA,PSK-AES256-CBC-SHA,PSK-3DES-EDE-CBC-SHA,PSK-RC4-SHA"
	PskCiphers PskCiphers `json:"psk_ciphers,omitempty"`

	//+kubebuilder:default:=off
	//+kubebuilder:validation:Enum:=on;off
	SecureRenegotiate SecureRenegotiate `json:"secure_renegotiate,omitempty"`

	//+kubebuilder:default:=on
	//+kubebuilder:validation:Enum:=on;off
	ReuseSessions ReuseSessions `json:"reuse_sessions,omitempty"`

	//+kubebuilder:default:=on
	//+kubebuilder:validation:Enum:=on;off
	HonorCipherOrder HonorCipherOrder `json:"honor_cipher_order,omitempty"`

	//+kubebuilder:default:="/mqtt"
	MqttPath MqttPath `json:"mqtt_path,omitempty"`

	//+kubebuilder:default:=True
	//+kubebuilder:validation:Enum:=True;False
	FailIfNoSubprotocol FailIfNoSubprotocol `json:"fail_if_no_subprotocol,omitempty"`

	//+kubebuilder:default:="mqtt, mqtt-v3, mqtt-v3.1.1, mqtt-v5"
	SupportedProtocols SupportedProtocols `json:"supported_protocols,omitempty"`

	ProxyAddressHeader ProxyAddressHeader `json:"proxy_address_header,omitempty"`

	ProxyPortHeader ProxyPortHeader `json:"proxy_port_header,omitempty"`

	//+kubebuilder:default:=False
	//+kubebuilder:validation:Enum:=True;Flase
	Compress Compress `json:"compress,omitempty"`

	DeflateOpts DeflateOpts `json:",omitempty"`

	//+kubebuilder:default:="15s"
	IdleTimeout metav1.Duration `json:"idle_timeout,omitempty"`

	MaxFrameSize MaxFrameSize `json:"max_frame_size,omitempty"`
}

type DeflateOpts struct {

	//+kubebuilder:default:=default
	//+kubebuilder:validation:Enum:=none;default;best_compression;best_speed
	Level Level `json:"level,omitempty"`

	//	//+kubebuilder:validation:Enum:=1;2;3;4;5;6;7;8;9
	MemLevel MemLevel `json:"mem_level,omitempty"`

	//+kubebuilder:validation:Enum:=default;filtered;huffman_only;rle
	Strategy Strategy `json:"strategy,omitempty"`

	//+kubebuilder:validation:Enum:=takeover;no_takeover
	ServerContextTakeover ServerContextTakeover `json:"server_context_takeover,omitempty"`

	//+kubebuilder:validation:Enum:=takeover;no_takeover
	ClientContextTakeover ClientContextTakeover `json:"client_context_takeover,omitempty"`

	//+kubebuilder:validation:Enum:=8;9;10;11;12;13;14;15
	ServerMaxWindowBits ServerMaxWindowBits `json:"server_max_window_bits,omitempty"`

	//+kubebuilder:validation:Enum:=8;9;10;11;12;13;14;15
	ClientMaxWindowBits ClientMaxWindowBits `json:"client_max_window_bits,omitempty"`
}

type Level string

const (
	LEVEL_NONE             Level = "none"
	LEVEL_DEFAULT          Level = "default"
	LEVEL_BEST_COMPRESSION Level = "best_compression"
	LEVEL_BEST_SPEED       Level = "best_speed"
)

type MemLevel uint8

const (
	MEM_LEVEL_1 MemLevel = 1
	MEM_LEVEL_2 MemLevel = 2
	MEM_LEVEL_3 MemLevel = 3
	MEM_LEVEL_4 MemLevel = 4
	MEM_LEVEL_5 MemLevel = 5
	MEM_LEVEL_6 MemLevel = 6
	MEM_LEVEL_7 MemLevel = 7
	MEM_LEVEL_8 MemLevel = 8
	MEM_LEVEL_9 MemLevel = 9
)

type Strategy string

const (
	STRATEGY_DEFAULT      Strategy = "default"
	STRATEGY_FILTERED     Strategy = "filtered"
	STRATEGY_HUFFMAN_ONLY Strategy = "huffman_only"
	STRATEGY_RLE          Strategy = "rle"
)

type ServerContextTakeover string

const (
	SERVER_CONTEXT_TAKEOVER_TAKEOVER    ServerContextTakeover = "takeover"
	SERVER_CONTEXT_TAKEOVER_NO_TAKEOVER ServerContextTakeover = "no_takeover"
)

type ClientContextTakeover string

const (
	CLIENT_CONTEXT_TAKEOVER_TAKEOVER    ClientContextTakeover = "takeover"
	CLIENT_CONTEXT_TAKEOVER_NO_TAKEOVER ClientContextTakeover = "no_takeover"
)

type ServerMaxWindowBits uint8

const (
	SERVER_MAX_WINDOW_BITS_8  ServerMaxWindowBits = 8
	SERVER_MAX_WINDOW_BITS_9  ServerMaxWindowBits = 9
	SERVER_MAX_WINDOW_BITS_10 ServerMaxWindowBits = 10
	SERVER_MAX_WINDOW_BITS_11 ServerMaxWindowBits = 11
	SERVER_MAX_WINDOW_BITS_12 ServerMaxWindowBits = 12
	SERVER_MAX_WINDOW_BITS_13 ServerMaxWindowBits = 13
	SERVER_MAX_WINDOW_BITS_14 ServerMaxWindowBits = 14
	SERVER_MAX_WINDOW_BITS_15 ServerMaxWindowBits = 15
)

type ClientMaxWindowBits uint8

const (
	CLIENT_MAX_WINDOW_BITS_8  ClientMaxWindowBits = 8
	CLIENT_MAX_WINDOW_BITS_9  ClientMaxWindowBits = 9
	CLIENT_MAX_WINDOW_BITS_10 ClientMaxWindowBits = 10
	CLIENT_MAX_WINDOW_BITS_11 ClientMaxWindowBits = 11
	CLIENT_MAX_WINDOW_BITS_12 ClientMaxWindowBits = 12
	CLIENT_MAX_WINDOW_BITS_13 ClientMaxWindowBits = 13
	CLIENT_MAX_WINDOW_BITS_14 ClientMaxWindowBits = 14
	CLIENT_MAX_WINDOW_BITS_15 ClientMaxWindowBits = 15
)
