package v1alpha2

import corev1 "k8s.io/api/core/v1"

type Listener struct {
	//+kubebuilder:validation:Enum:=NodePort;LoadBalancer;ClusterIP
	Type corev1.ServiceType `json:"type,omitempty"`

	//+optional
	Ports []corev1.ServicePort `json:"ports,omitempty"`
}
