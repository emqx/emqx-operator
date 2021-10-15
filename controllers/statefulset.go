package controllers

import (
	"fmt"

	"github.com/emqx/emqx-operator/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeStatefulSetSpec(instance *v1alpha1.Emqx) *v1.StatefulSetSpec {

	ports := generateContainerPorts()

	volumes := generateVolumes(instance)

	volumeMounts := generateVolumeMounts(instance)

	postStartCommand := []string{"sudo", "/bin/sh", "-c", "chown -R 1000:1000 /opt/emqx/log /opt/emqx/data/mnesia"}

	lifecycle := &corev1.Lifecycle{
		PostStart: &corev1.Handler{
			Exec: &corev1.ExecAction{
				Command: postStartCommand,
			},
		},
	}
	env := mergeClusterConfigToEnv(instance)
	// var value int64 = 0
	// var privileged bool = true
	// securityContext := &corev1.SecurityContext{
	// 	RunAsUser: &value,
	// 	// RunAsGroup: &value,
	// 	// FSGroup:    &value,
	// 	Privileged: &privileged,
	// }

	statefulsetSpec := &v1.StatefulSetSpec{
		ServiceName: instance.Name,
		Replicas:    instance.Spec.Replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: instance.Spec.Labels,
		},
		PodManagementPolicy: v1.ParallelPodManagement,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: instance.Spec.Labels,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:      EMQX_NAME,
						Image:     instance.Spec.Image,
						Env:       env,
						Lifecycle: lifecycle,
						Ports:     ports,

						VolumeMounts: volumeMounts,
						// SecurityContext: securityContext,
					},
				},
				ServiceAccountName: instance.Spec.ServiceAccountName,
				Volumes:            volumes,
			},
		},
	}

	storageSpec := instance.Spec.Storage
	// define the VolumeClaimTemplate to apply the pvc
	if storageSpec == nil {
		volumes = append(volumes, corev1.Volume{
			Name: volumeName(instance.Name),
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	} else {
		pvcList := []string{EMQX_LOG_NAME, EMQX_DATA_NAME}
		for _, item := range pvcList {
			pvcTemplate := MakeVolumeClaimTemplate(storageSpec.VolumeClaimTemplate, instance)
			if pvcTemplate.Name == "" {
				pvcTemplate.Name = volumeName(item)
			}
			if storageSpec.VolumeClaimTemplate.Spec.AccessModes == nil {
				pvcTemplate.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
			} else {
				pvcTemplate.Spec.AccessModes = storageSpec.VolumeClaimTemplate.Spec.AccessModes
			}
			pvcTemplate.Spec.Resources = storageSpec.VolumeClaimTemplate.Spec.Resources
			pvcTemplate.Spec.Selector = storageSpec.VolumeClaimTemplate.Spec.Selector
			statefulsetSpec.VolumeClaimTemplates = append(statefulsetSpec.VolumeClaimTemplates, *pvcTemplate)
		}
	}
	return statefulsetSpec
}

func generateContainerPorts() []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{
			ContainerPort: 1883,
		},
		{
			ContainerPort: 8883,
		},
		{
			ContainerPort: 8081,
		},
		{
			ContainerPort: 8083,
		},
		{
			ContainerPort: 8084,
		},
	}
}

func generateVolumes(instance *v1alpha1.Emqx) []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: EMQX_LIC_NAME,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: EMQX_LIC_NAME,
					Items: []corev1.KeyToPath{
						{
							Key:  "emqx.lic",
							Path: "emqx.lic",
						},
					},
				},
			},
		},
	}
	if instance.Spec.AclConf != "" {
		volumes = append(volumes,
			corev1.Volume{
				Name: instance.Name + "-" + EMQX_ACL_CONF_NAME,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: instance.Name + "-" + EMQX_ACL_CONF_NAME,
						},
						Items: []corev1.KeyToPath{
							{
								Key:  "acl.conf",
								Path: "acl.conf",
							},
						},
					},
				},
			})
	}
	if instance.Spec.LoadedPluginConf != "" {
		volumes = append(volumes,
			corev1.Volume{
				Name: instance.Name + "-" + EMQX_LOADED_MODULES_NAME,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: instance.Name + "-" + EMQX_LOADED_MODULES_NAME,
						},
						Items: []corev1.KeyToPath{
							{
								Key:  "loaded_modules",
								Path: "loaded_modules",
							},
						},
					},
				},
			})
	}
	if instance.Spec.LoadedModulesConf != "" {
		volumes = append(volumes,
			corev1.Volume{
				Name: instance.Name + "-" + EMQX_LOADED_PLUGINS_NAME,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: instance.Name + "-" + EMQX_LOADED_PLUGINS_NAME,
						},
						Items: []corev1.KeyToPath{
							{
								Key:  "loaded_plugins",
								Path: "loaded_plugins",
							},
						},
					},
				},
			})
	}
	return volumes
}

func generateVolumeMounts(instance *v1alpha1.Emqx) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      EMQX_LIC_NAME,
			MountPath: EMQX_LIC_DIR,
			SubPath:   EMQX_LIC_SUBPATH,
			ReadOnly:  true,
		},
	}
	if instance.Spec.AclConf != "" {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				Name:      instance.Name + "-" + EMQX_ACL_CONF_NAME,
				MountPath: EMQX_ACL_CONF_DIR,
				SubPath:   EMQX_ACL_CONF_SUBPATH,
			},
		)
	}
	if instance.Spec.LoadedModulesConf != "" {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				Name:      instance.Name + "-" + EMQX_LOADED_MODULES_NAME,
				MountPath: EMQX_LOADED_MODULES_DIR,
				SubPath:   EMQX_LOADED_MODULES_SUBPATH,
			},
		)
	}
	if instance.Spec.LoadedPluginConf != "" {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				Name:      instance.Name + "-" + EMQX_LOADED_PLUGINS_NAME,
				MountPath: EMQX_LOADED_PLUGINS_DIR,
				SubPath:   EMQX_LOADED_PLUGINS_SUBPATH,
			},
		)
	}
	return volumeMounts
}

func mergeClusterConfigToEnv(instance *v1alpha1.Emqx) []corev1.EnvVar {
	envVar := instance.Spec.Env
	clusterConfig := instance.Spec.Cluster
	envVar = append(envVar, clusterConfig.ConvertToEnv()...)
	return envVar
}

func volumeName(name string) string {
	return fmt.Sprintf("pvc-%s", name)
}

func MakeVolumeClaimTemplate(e v1alpha1.EmbeddedPersistentVolumeClaim, instance *v1alpha1.Emqx) *corev1.PersistentVolumeClaim {
	boolTrue := true
	pvc := corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: e.APIVersion,
			Kind:       e.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        e.Name,
			Namespace:   instance.Namespace,
			Annotations: instance.ObjectMeta.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         instance.APIVersion,
					BlockOwnerDeletion: &boolTrue,
					Controller:         &boolTrue,
					Kind:               instance.Kind,
					Name:               instance.Name,
					UID:                instance.UID,
				},
			},
		},
		Spec:   e.Spec,
		Status: e.Status,
	}
	return &pvc
}
