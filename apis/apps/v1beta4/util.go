package v1beta4

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func MergeServicePorts(ports1, ports2 []corev1.ServicePort) []corev1.ServicePort {
	ports := append(ports1, ports2...)

	result := make([]corev1.ServicePort, 0, len(ports))
	tempName := map[string]struct{}{}
	tempPort := map[int32]struct{}{}

	for _, item := range ports {
		_, nameOK := tempName[item.Name]
		_, portOK := tempPort[item.Port]

		if !nameOK && !portOK {
			tempName[item.Name] = struct{}{}
			tempPort[item.Port] = struct{}{}
			result = append(result, item)
		}
	}

	return result
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
