package controller

import (
	"fmt"

	crd "github.com/emqx/emqx-operator/api/v3alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const DefaultBootstrapAPIKey string = "emqx-operator-controller"

const boostrapApiKeysVolumeName = "bootstrap-api-keys"

type cookieResource struct {
	*crd.EMQX
}

type bootstrapAPIKeyResource struct {
	*crd.EMQX
}

func BootstrapAPIKey(instance *crd.EMQX) bootstrapAPIKeyResource {
	return bootstrapAPIKeyResource{instance}
}

func Cookie(instance *crd.EMQX) cookieResource {
	return cookieResource{instance}
}

func (from bootstrapAPIKeyResource) Secret(content string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: from.Namespace,
			Name:      from.BootstrapAPIKeyNamespacedName().Name,
			Labels:    from.DefaultLabelsWith(from.Labels),
		},
		StringData: map[string]string{
			"bootstrap_api_key": content,
		},
	}
}

func (from bootstrapAPIKeyResource) VolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      boostrapApiKeysVolumeName,
		MountPath: "/opt/emqx/etc/bootstrap_api_keys",
		SubPath:   "bootstrap_api_key",
		ReadOnly:  true,
	}
}

func (from bootstrapAPIKeyResource) Volume() corev1.Volume {
	return corev1.Volume{
		Name: boostrapApiKeysVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: from.BootstrapAPIKeyNamespacedName().Name,
			},
		},
	}
}

func (from bootstrapAPIKeyResource) EnvVar() corev1.EnvVar {
	return corev1.EnvVar{
		Name:  "EMQX_API_KEY__BOOTSTRAP_FILE",
		Value: fmt.Sprintf(`"%s"`, from.VolumeMount().MountPath),
	}
}

func (from cookieResource) Secret(content string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: from.Namespace,
			Name:      from.NodeCookieNamespacedName().Name,
			Labels:    from.DefaultLabelsWith(from.Labels),
		},
		StringData: map[string]string{
			"node_cookie": content,
		},
	}
}

func (from cookieResource) EnvVar() corev1.EnvVar {
	return corev1.EnvVar{
		Name: "EMQX_NODE__COOKIE",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: from.NodeCookieNamespacedName().Name,
				},
				Key: "node_cookie",
			},
		},
	}
}
