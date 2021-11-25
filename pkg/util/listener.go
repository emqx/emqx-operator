package util

import (
	corev1 "k8s.io/api/core/v1"
)

//+kubebuilder:object:generate=true
type Listener struct {
	//+kubebuilder:validation:Enum:=NodePort;LoadBalancer;ClusterIP
	Type                     corev1.ServiceType `json:"type,omitempty"`
	LoadBalancerIP           string             `json:"loadBalancerIP,omitempty" protobuf:"bytes,8,opt,name=loadBalancerIP"`
	LoadBalancerSourceRanges []string           `json:"loadBalancerSourceRanges,omitempty" protobuf:"bytes,9,opt,name=loadBalancerSourceRanges"`
	ExternalIPs              []string           `json:"externalIPs,omitempty" protobuf:"bytes,5,rep,name=externalIPs"`
	Ports                    ports              `json:"ports,omitempty"`
	NodePorts                ports              `json:"nodePorts,omitempty"`
}

type ports struct {
	MQTT      int32 `json:"mqtt,omitempty"`
	MQTTS     int32 `json:"mqtts,omitempty"`
	WS        int32 `json:"ws,omitempty"`
	WSS       int32 `json:"wss,omitempty"`
	Dashboard int32 `json:"dashboard,omitempty"`
	API       int32 `json:"api,omitempty"`
}

func GenerateListener(listener Listener) Listener {
	if IsNil(listener) {
		return defaultListener()
	} else {
		if IsNil(listener.Type) {
			listener.Type = defaultListener().Type
		}
		if IsNil(listener.Ports) {
			listener.Ports = defaultListener().Ports
		}
		return listener
	}
}

func defaultListener() Listener {
	return Listener{
		Type: "ClusterIP",
		Ports: ports{
			MQTT:      1883,
			MQTTS:     8883,
			WS:        8083,
			WSS:       8084,
			Dashboard: 18083,
			API:       8081,
		},
	}

}
