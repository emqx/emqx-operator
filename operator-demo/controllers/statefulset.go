package controllers

import (
	"Emqx/api/v1alpha1"

	pkgerr "github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeStatefulSet(instance *v1alpha1.Broker) (*v1.StatefulSet, error) {
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

func makeStatefulSetSpec(instance *v1alpha1.Broker) (*v1.StatefulSetSpec, error) {
	env := []corev1.EnvFromSource{
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: emqxenvName,
				},
			},
		},
	}

	ports := []corev1.ContainerPort{
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

	volumes := []corev1.Volume{
		{
			Name: emqxlicName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: emqxlicName,
					Items: []corev1.KeyToPath{
						{
							Key:  "emqx.lic",
							Path: "emqx.lic",
						},
					},
				},
			},
		},
		// {
		// 	Name: emqxloadmodulesName,
		// 	VolumeSource: corev1.VolumeSource{
		// 		ConfigMap: &corev1.ConfigMapVolumeSource{
		// 			LocalObjectReference: corev1.LocalObjectReference{
		// 				Name: emqxloadmodulesName,
		// 			},
		// 			Items: []corev1.KeyToPath{
		// 				{
		// 					Key:  "loaded-modules",
		// 					Path: "loaded_modules",
		// 				},
		// 			},
		// 		},
		// 	},
		// },
		{
			Name: emqxlogName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "pvc-" + emqxlogName,
					// ReadOnly:  true,
				},
			},
		},
		{
			Name: emqxdataName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "pvc-" + emqxdataName,
					// ReadOnly:  true,
				},
			},
		},
	}
	VolumeMounts := []corev1.VolumeMount{
		{
			Name:      emqxlicName,
			MountPath: emqxlicDir,
			SubPath:   emqxlicSubPath,
			ReadOnly:  true,
		},
		{
			Name:      emqxlogName,
			MountPath: emqxlogDir,
		},
		{
			Name:      emqxdataName,
			MountPath: emqxdataDir,
		},
		// {
		// 	Name:      emqxloadmodulesName,
		// 	MountPath: emqxloadmodulesDir,
		// 	SubPath:   emqxloadmodulesSubpath,
		// 	ReadOnly:  true,
		// },
	}
	podLabels := map[string]string{}
	podLabels["app"] = emqxName
	podLabels[emqxName] = instance.Name

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
						Name:         emqxName,
						Image:        instance.Spec.Image,
						Lifecycle:    lifecycle,
						Ports:        ports,
						VolumeMounts: VolumeMounts,
						EnvFrom:      env,
						// SecurityContext: securityContext,
					},
				},
				ServiceAccountName: instance.Spec.ServiceAccountName,
				Volumes:            volumes,
			},
		},
	}, nil
}
