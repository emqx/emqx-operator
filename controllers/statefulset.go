package controllers

import (
	"github.com/emqx/emqx-operator/api/v1alpha1"
	pkgerr "github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeStatefulSet(instance *v1alpha1.Emqx) (*v1.StatefulSet, error) {
	spec, err := makeStatefulSetSpec(instance)

	if err != nil {
		return nil, pkgerr.Wrap(err, "make StatefulSet spec")
	}

	boolTrue := true
	statefulset := &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        instance.Name,
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
		Spec: *spec,
	}
	return statefulset, nil

}

func makeStatefulSetSpec(instance *v1alpha1.Emqx) (*v1.StatefulSetSpec, error) {

	ports := generateContainerPorts()

	volumes := generateVolumes(instance)

	volumeMounts := generateVolumeMounts(instance)
	podLabels := generatePodLabels(instance)

	postStartCommand := []string{"sudo", "/bin/sh", "-c", "chown -R 1000:1000 /opt/emqx/log /opt/emqx/data/mnesia"}

	lifecycle := &corev1.Lifecycle{
		PostStart: &corev1.Handler{
			Exec: &corev1.ExecAction{
				Command: postStartCommand,
			},
		},
	}
	// var value int64 = 0
	// var privileged bool = true
	// securityContext := &corev1.SecurityContext{
	// 	RunAsUser: &value,
	// 	// RunAsGroup: &value,
	// 	// FSGroup:    &value,
	// 	Privileged: &privileged,
	// }

	return &v1.StatefulSetSpec{
		ServiceName: instance.Name,
		Replicas:    instance.Spec.Replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: podLabels,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: podLabels,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:      EMQX_NAME,
						Image:     instance.Spec.Image,
						Env:       instance.Spec.Env,
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
	}, nil
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
		{
			Name: EMQX_LOG_NAME,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "pvc-" + EMQX_LOG_NAME,
				},
			},
		},
		{
			Name: EMQX_DATA_NAME,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "pvc-" + EMQX_DATA_NAME,
				},
			},
		},
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
		{
			Name:      EMQX_LOG_NAME,
			MountPath: EMQX_LOG_DIR,
		},
		{
			Name:      EMQX_DATA_NAME,
			MountPath: EMQX_DATA_DIR,
		},
	}
	return volumeMounts
}

func generatePodLabels(instance *v1alpha1.Emqx) map[string]string {
	return map[string]string{
		"app":     EMQX_NAME,
		EMQX_NAME: instance.Name,
	}
}
