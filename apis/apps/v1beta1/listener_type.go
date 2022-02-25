package v1beta1

import (
	"reflect"

	corev1 "k8s.io/api/core/v1"
)

//+kubebuilder:object:generate=true
type Listener struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	//+kubebuilder:validation:Enum=NodePort;LoadBalancer;ClusterIP
	Type                     corev1.ServiceType `json:"type,omitempty"`
	LoadBalancerIP           string             `json:"loadBalancerIP,omitempty" protobuf:"bytes,8,opt,name=loadBalancerIP"`
	LoadBalancerSourceRanges []string           `json:"loadBalancerSourceRanges,omitempty" protobuf:"bytes,9,opt,name=loadBalancerSourceRanges"`
	ExternalIPs              []string           `json:"externalIPs,omitempty" protobuf:"bytes,5,rep,name=externalIPs"`
	Ports                    Ports              `json:"ports,omitempty"`
	NodePorts                Ports              `json:"nodePorts,omitempty"`
	Certificate              Certificate        `json:"certificate,omitempty"`
}

type Ports struct {
	//+kubebuilder:validation:Maximum=65535
	MQTT int32 `json:"mqtt,omitempty"`
	//+kubebuilder:validation:Maximum=65535
	MQTTS int32 `json:"mqtts,omitempty"`
	//+kubebuilder:validation:Maximum=65535
	WS int32 `json:"ws,omitempty"`
	//+kubebuilder:validation:Maximum=65535
	WSS int32 `json:"wss,omitempty"`
	//+kubebuilder:validation:Maximum=65535
	Dashboard int32 `json:"dashboard,omitempty"`
	//+kubebuilder:validation:Maximum=65535
	API int32 `json:"api,omitempty"`
}

//+kubebuilder:object:generate=true
type Certificate struct {
	WSS   CertificateConf `json:"wss,omitempty"`
	MQTTS CertificateConf `json:"mqtts,omitempty"`
}

type CertificateConf struct {
	Data       CertificateData       `json:"data,omitempty"`
	StringData CertificateStringData `json:"stringData,omitempty"`
}

type CertificateData struct {
	CaCert  []byte `json:"ca.crt,omitempty"`
	TLSCert []byte `json:"tls.crt,omitempty"`
	TLSKey  []byte `json:"tls.key,omitempty"`
}

type CertificateStringData struct {
	CaCert  string `json:"ca.crt,omitempty"`
	TLSCert string `json:"tls.crt,omitempty"`
	TLSKey  string `json:"tls.key,omitempty"`
}

func generateListener(listener Listener) Listener {
	if reflect.ValueOf(listener).IsZero() {
		return defaultListener()
	} else {
		if reflect.ValueOf(listener.Type).IsZero() {
			listener.Type = defaultListener().Type
		}
		if reflect.ValueOf(listener.Ports).IsZero() {
			listener.Ports = defaultListener().Ports
		}
		if reflect.ValueOf(listener.Ports.API).IsZero() {
			listener.Ports.API = defaultListener().Ports.API
		}
		return listener
	}
}

func defaultListener() Listener {
	return Listener{
		Type: "ClusterIP",
		Ports: Ports{
			MQTT:      1883,
			MQTTS:     8883,
			WS:        8083,
			WSS:       8084,
			Dashboard: 18083,
			API:       8081,
		},
	}

}
