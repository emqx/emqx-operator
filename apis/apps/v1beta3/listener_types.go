package v1beta3

import (
	corev1 "k8s.io/api/core/v1"
)

type CertData struct {
	CaCert  []byte `json:"ca.crt,omitempty"`
	TLSCert []byte `json:"tls.crt,omitempty"`
	TLSKey  []byte `json:"tls.key,omitempty"`
}

type CertStringData struct {
	CaCert  string `json:"ca.crt,omitempty"`
	TLSCert string `json:"tls.crt,omitempty"`
	TLSKey  string `json:"tls.key,omitempty"`
}

type CertConf struct {
	Data       CertData       `json:"data,omitempty"`
	StringData CertStringData `json:"stringData,omitempty"`
}

type Cert struct {
	Data       CertData       `json:"data,omitempty"`
	StringData CertStringData `json:"stringData,omitempty"`
}

type ListenerPort struct {
	Port     int32    `json:"port,omitempty"`
	NodePort int32    `json:"nodePort,omitempty"`
	Cert     CertConf `json:"cert,omitempty"`
}

type Listener struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	//+kubebuilder:validation:Enum=NodePort;LoadBalancer;ClusterIP
	Type                     corev1.ServiceType `json:"type,omitempty"`
	LoadBalancerIP           string             `json:"loadBalancerIP,omitempty" protobuf:"bytes,8,opt,name=loadBalancerIP"`
	LoadBalancerSourceRanges []string           `json:"loadBalancerSourceRanges,omitempty" protobuf:"bytes,9,opt,name=loadBalancerSourceRanges"`
	ExternalIPs              []string           `json:"externalIPs,omitempty" protobuf:"bytes,5,rep,name=externalIPs"`
	API                      ListenerPort       `json:"api,omitempty"`
	Dashboard                ListenerPort       `json:"dashboard,omitempty"`
	MQTT                     ListenerPort       `json:"mqtt,omitempty"`
	MQTTS                    ListenerPort       `json:"mqtts,omitempty"`
	WS                       ListenerPort       `json:"ws,omitempty"`
	WSS                      ListenerPort       `json:"wss,omitempty"`
}

func (l *Listener) Default() {
	if l.Dashboard.Port == 0 && l.MQTT.Port == 0 && l.MQTTS.Port == 0 && l.WS.Port == 0 && l.WSS.Port == 0 {
		l.Dashboard.Port = int32(18083)
		l.MQTT.Port = int32(1883)
		l.MQTTS.Port = int32(8883)
		l.WS.Port = int32(8083)
		l.WSS.Port = int32(8084)
	}
	if l.API.Port == 0 {
		l.API.Port = int32(8081)
	}
	if l.Type == "" {
		l.Type = corev1.ServiceTypeClusterIP
	}
}
