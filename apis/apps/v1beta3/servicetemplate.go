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
		if _, ok := s.ObjectMeta.Labels[key]; !ok {
			s.ObjectMeta.Labels[key] = value
		}
	}
	if s.ObjectMeta.Annotations == nil {
		s.ObjectMeta.Annotations = map[string]string{}
	}
	for key, value := range emqx.GetAnnotations() {
		if key == "kubectl.kubernetes.io/last-applied-configuration" {
			continue
		}
		if _, ok := s.ObjectMeta.Annotations[key]; !ok {
			s.ObjectMeta.Annotations[key] = value
		}
	}

	s.Spec.Selector = emqx.GetLabels()
	s.MergePorts([]corev1.ServicePort{
		{
			Name:       "http-management-8081",
			Port:       8081,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromInt(8081),
		},
	})
}

func (s *ServiceTemplate) MergePorts(ports []corev1.ServicePort) {
	ports = append(s.Spec.Ports, ports...)

	result := make([]corev1.ServicePort, 0, len(ports))
	temp := map[string]struct{}{}

	for _, item := range ports {
		if _, ok := temp[item.Name]; !ok {
			temp[item.Name] = struct{}{}
			result = append(result, item)
		}
	}

	s.Spec.Ports = result
}
