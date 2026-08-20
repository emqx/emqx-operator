package controller

import (
	crd "github.com/emqx/emqx-operator/api/v3beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const BaseConfigFile string = "base.hocon"
const StartupConfigFile string = "emqx.conf"

const configVolumeName = "bootstrap-config"

type emqxConfigResource struct {
	*crd.EMQX
}

func EMQXConfig(instance *crd.EMQX) emqxConfigResource {
	return emqxConfigResource{instance}
}

func (from emqxConfigResource) ConfigMap(baseConfig, startupConfig string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      from.ConfigsNamespacedName().Name,
			Namespace: from.Namespace,
			Labels:    from.DefaultLabelsWith(from.Labels),
		},
		Data: map[string]string{
			BaseConfigFile:    baseConfig,
			StartupConfigFile: startupConfig,
		},
	}
}

func (emqxConfigResource) VolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      configVolumeName,
			MountPath: "/opt/emqx/etc/" + BaseConfigFile,
			SubPath:   BaseConfigFile,
			ReadOnly:  true,
		},
		{
			Name:      configVolumeName,
			MountPath: "/opt/emqx/etc/" + StartupConfigFile,
			SubPath:   StartupConfigFile,
			ReadOnly:  true,
		},
	}
}

func (from emqxConfigResource) Volume() corev1.Volume {
	return corev1.Volume{
		Name: configVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: from.ConfigsNamespacedName().Name,
				},
			},
		},
	}
}
