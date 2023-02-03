package v1beta4

import (
	"fmt"
	"path/filepath"

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

func mergeMap(dst, src map[string]string) map[string]string {
	if dst == nil {
		dst = make(map[string]string)
	}

	for key, value := range src {
		if _, ok := dst[key]; !ok {
			dst[key] = value
		}
	}
	return dst
}

func GetEmqxImage(instance Emqx) string {
	image := instance.GetSpec().GetTemplate().Spec.EmqxContainer.Image
	return fmt.Sprintf(
		"%s:%s",
		filepath.Join(image.Registry, image.Repository),
		image.Prefix+image.Version+image.Suffix,
	)
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
