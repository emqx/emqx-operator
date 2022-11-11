package v1beta4

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

// +kubebuilder:object:generate=false
type Names struct {
	metav1.Object
}

func (n Names) HeadlessSvc() string {
	return fmt.Sprintf("%s-%s", n.Object.GetName(), "headless")
}

func (n Names) License() string {
	return fmt.Sprintf("%s-%s", n.Object.GetName(), "license")
}

func (n Names) ACL() string {
	return fmt.Sprintf("%s-%s", n.Object.GetName(), "acl")
}

func (n Names) PluginsConfig() string {
	return fmt.Sprintf("%s-%s", n.Object.GetName(), "plugins-config")
}

func (n Names) Data() string {
	return fmt.Sprintf("%s-%s", n.Object.GetName(), "data")
}

func (n Names) BootstrapUser() string {
	return fmt.Sprintf("%s-%s", n.Object.GetName(), "bootstrap-user")
}
