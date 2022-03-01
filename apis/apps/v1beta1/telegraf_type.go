package v1beta1

import corev1 "k8s.io/api/core/v1"

//+kubebuilder:object:generate=true
type TelegrafTemplate struct {
	//+kubebuilder:validation:Required
	Image string `json:"image,omitempty"`
	//+kubebuilder:validation:Required
	Conf            *string                     `json:"conf,omitempty"`
	Resources       corev1.ResourceRequirements `json:"resources,omitempty"`
	ImagePullPolicy corev1.PullPolicy           `json:"imagePullPolicy,omitempty"`
}
