package v1beta3

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type ServiceTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              corev1.ServiceSpec `json:"spec,omitempty"`
}

func (s *ServiceTemplate) Default(emqx Emqx) {
	s.ObjectMeta.Namespace = emqx.GetNamespace()
	if s.ObjectMeta.Name == "" {
		s.ObjectMeta.Name = emqx.GetName()
	}
	if s.ObjectMeta.Labels == nil {
		s.ObjectMeta.Labels = make(map[string]string)
	}
	for key, value := range emqx.GetLabels() {
		s.ObjectMeta.Labels[key] = value
	}

	s.Spec.Selector = emqx.GetLabels()
	if s.Spec.Ports == nil {
		s.Spec.Ports = []corev1.ServicePort{
			{
				Name:       "listener-tcp-external",
				Port:       1883,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(1883),
			},
			{
				Name:       "listener-ssl-external",
				Port:       8883,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(8883),
			},
			{
				Name:       "listener-ws-external",
				Port:       8083,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(8083),
			},
			{
				Name:       "listener-wss-external",
				Port:       8084,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(8084),
			},
		}
	}
}
