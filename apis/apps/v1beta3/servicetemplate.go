package v1beta3

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		if _, ok := s.ObjectMeta.Annotations[key]; !ok {
			s.ObjectMeta.Annotations[key] = value
		}
	}

	s.Spec.Selector = emqx.GetLabels()
}
