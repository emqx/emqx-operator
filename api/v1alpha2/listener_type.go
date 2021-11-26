package v1alpha2

import (
	"reflect"

	corev1 "k8s.io/api/core/v1"
)

//+kubebuilder:object:generate=true
type Listener struct {
	//+kubebuilder:validation:Enum=NodePort;LoadBalancer;ClusterIP
	Type                     corev1.ServiceType `json:"type,omitempty"`
	LoadBalancerIP           string             `json:"loadBalancerIP,omitempty" protobuf:"bytes,8,opt,name=loadBalancerIP"`
	LoadBalancerSourceRanges []string           `json:"loadBalancerSourceRanges,omitempty" protobuf:"bytes,9,opt,name=loadBalancerSourceRanges"`
	ExternalIPs              []string           `json:"externalIPs,omitempty" protobuf:"bytes,5,rep,name=externalIPs"`
	Ports                    Ports              `json:"ports,omitempty"`
	NodePorts                Ports              `json:"nodePorts,omitempty"`
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

func GenerateListener(listener Listener) Listener {
	if reflect.ValueOf(listener).IsZero() {
		return defaultListener()
	} else {
		if reflect.ValueOf(listener.Type).IsZero() {
			listener.Type = defaultListener().Type
		}
		if reflect.ValueOf(listener.Ports).IsZero() {
			listener.Ports = defaultListener().Ports
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
