package service

import (
	"github.com/emqx/emqx-operator/api/v1alpha1"
	"github.com/emqx/emqx-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// func generateEmqxConfigMap(emqx *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
// 	name := util.GetEmqxName(emqx)
// 	namespace := emqx.Namespace

// }

func newSecretForCR(emqx *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Secret {
	stringData := map[string]string{"emqx.lic": emqx.Spec.License}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            util.GetEmqxSecret(emqx),
			Namespace:       emqx.Namespace,
			OwnerReferences: ownerRefs,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: stringData,
	}

	return secret

}

func newHeadLessSvcForCR(emqx *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Service {
	emqxPorts := []corev1.ServicePort{
		{
			Name:     SERVICE_TCP_NAME,
			Port:     SERVICE_TCP_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: SERVICE_TCP_PORT,
			},
		},
		{
			Name:     SERVICE_TCPS_NAME,
			Port:     SERVICE_TCPS_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: SERVICE_TCPS_PORT,
			},
		},
		{
			Name:     SERVICE_WS_NAME,
			Port:     SERVICE_WS_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: SERVICE_WS_PORT,
			},
		},
		{
			Name:     SERVICE_WSS_NAME,
			Port:     SERVICE_WSS_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: SERVICE_WSS_PORT,
			},
		},
		{
			Name:     SERVICE_DASHBOARD_NAME,
			Port:     SERVICE_DASHBOARD_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: SERVICE_DASHBOARD_PORT,
			},
		},
	}
	// labels = util.MergeLabels(labels, generateSelectorLabels(util.SentinelRoleName, cluster.Name))
	labels = map[string]string{}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            util.GetEmqxHeadlessSvc(emqx),
			Namespace:       emqx.Namespace,
			OwnerReferences: ownerRefs,
		},
		Spec: corev1.ServiceSpec{
			Ports:     emqxPorts,
			Selector:  labels,
			ClusterIP: corev1.ClusterIPNone,
		},
	}

	return svc
}

func newConfigMapForLoadedMoudles(emqx *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
	data := map[string]string{"loaded_modules": emqx.Spec.LoadedModulesConf}
	cmForPM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            util.GetEmqxConfigMapForLM(emqx),
			Namespace:       emqx.Namespace,
			OwnerReferences: ownerRefs,
		},
		Data: data,
	}

	return cmForPM
}

func newConfigMapForLoadedPlugins(emqx *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
	data := map[string]string{"loaded_plugins": emqx.Spec.LoadedPluginConf}
	cmForPG := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            util.GetEmqxConfigMapForPG(emqx),
			Namespace:       emqx.Namespace,
			OwnerReferences: ownerRefs,
		},
		Data: data,
	}

	return cmForPG
}

func newEmqxStatefulSet(emqx *v1alpha1.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *appsv1.StatefulSet {
	name := util.GetEmqxName(emqx)
	namespace := emqx.Namespace

	ports := getContainerPorts()

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

	env := mergeClusterConfigToEnv(emqx)

	// TODO
	labels = map[string]string{}

	volumeMounts := getEmqxVolumeMounts(emqx)
	volumes := getEmqxVolumes(emqx)

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			Labels:          labels,
			OwnerReferences: ownerRefs,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: util.GetEmqxHeadlessSvc(emqx),
			Replicas:    emqx.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: emqx.Spec.Labels,
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					// TODO merge labels
					Labels: emqx.Spec.Labels,
					// TODO
					// Annotations: rc.Spec.Sentinel.Annotations,
				},
				Spec: corev1.PodSpec{
					// TODO initContainers
					// InitContainers

					// TODO
					// Affinity: getAffinity(rc.Spec.Affinity, labels),
					Tolerations:  emqx.Spec.ToleRations,
					NodeSelector: emqx.Spec.NodeSelector,
					Containers: []corev1.Container{
						{
							Name:            EMQX_NAME,
							Image:           emqx.Spec.Image,
							ImagePullPolicy: getPullPolicy(emqx.Spec.ImagePullPolicy),
							Resources:       emqx.Spec.Resources,
							Env:             env,
							Lifecycle:       lifecycle,
							Ports:           ports,
							VolumeMounts:    volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}
	if emqx.Spec.Storage.PersistentVolumeClaim != nil {
		sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			*emqx.Spec.Storage.PersistentVolumeClaim,
		}
	}
	return sts
}

func getEmqxVolumeMounts(emqx *v1alpha1.Emqx) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{}
	if emqx.Spec.License != "" {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				Name:      EMQX_LIC_NAME,
				MountPath: EMQX_LIC_DIR,
				SubPath:   EMQX_LIC_SUBPATH,
				ReadOnly:  true,
			},
		)
	}
	if emqx.Spec.AclConf != "" {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				Name:      EMQX_ACL_CONF_NAME,
				MountPath: EMQX_ACL_CONF_DIR,
				SubPath:   EMQX_ACL_CONF_SUBPATH,
			},
		)
	}
	if emqx.Spec.LoadedModulesConf != "" {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				Name:      EMQX_LOADED_MODULES_NAME,
				MountPath: EMQX_LOADED_MODULES_DIR,
				SubPath:   EMQX_LOADED_MODULES_SUBPATH,
			},
		)
	}
	if emqx.Spec.LoadedPluginConf != "" {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				Name:      EMQX_LOADED_PLUGINS_NAME,
				MountPath: EMQX_LOADED_PLUGINS_DIR,
				SubPath:   EMQX_LOADED_PLUGINS_SUBPATH,
			},
		)
	}
	return volumeMounts
}

func getEmqxVolumes(emqx *v1alpha1.Emqx) []corev1.Volume {
	volumes := []corev1.Volume{}
	if emqx.Spec.License != "" {
		volumes = append(volumes,
			corev1.Volume{
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
			})
	}

	if emqx.Spec.AclConf != "" {
		volumes = append(volumes,
			corev1.Volume{
				Name: EMQX_ACL_CONF_NAME,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: EMQX_ACL_CONF_NAME,
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
	if emqx.Spec.LoadedPluginConf != "" {
		volumes = append(volumes,
			corev1.Volume{
				Name: EMQX_LOADED_MODULES_NAME,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: EMQX_LOADED_MODULES_NAME,
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
	if emqx.Spec.LoadedModulesConf != "" {
		volumes = append(volumes,
			corev1.Volume{
				Name: EMQX_LOADED_PLUGINS_NAME,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: EMQX_LOADED_PLUGINS_NAME,
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

func mergeClusterConfigToEnv(emqx *v1alpha1.Emqx) []corev1.EnvVar {
	envVar := emqx.Spec.Env
	clusterConfig := emqx.Spec.Cluster
	envVar = append(envVar, clusterConfig.ConvertToEnv()...)
	return envVar
}

func getContainerPorts() []corev1.ContainerPort {
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

// TODO
// func getAffinity(affinity *corev1.Affinity, labels map[string]string) *corev1.Affinity {
// 	if affinity != nil {
// 		return affinity
// 	}

// 	// Return a SOFT anti-affinity
// 	return &corev1.Affinity{
// 		PodAntiAffinity: &corev1.PodAntiAffinity{
// 			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
// 				{
// 					Weight: 100,
// 					PodAffinityTerm: corev1.PodAffinityTerm{
// 						TopologyKey: util.HostnameTopologyKey,
// 						LabelSelector: &metav1.LabelSelector{
// 							MatchLabels: labels,
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}
// }
func getPullPolicy(specPolicy corev1.PullPolicy) corev1.PullPolicy {
	if specPolicy == "" {
		return corev1.PullAlways
	}
	return specPolicy
}
