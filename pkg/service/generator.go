package service

import (
	"github.com/emqx/emqx-operator/api/v1alpha1"
	"github.com/emqx/emqx-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func newSecretForCR(emqx *v1alpha1.EmqxBroker, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Secret {
	stringData := map[string]string{"emqx.lic": emqx.Spec.License}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            util.GetEmqxBrokerSecret(emqx),
			Namespace:       emqx.Namespace,
			OwnerReferences: ownerRefs,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: stringData,
	}

	return secret

}

func newHeadLessSvcForCR(emqx *v1alpha1.EmqxBroker, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Service {
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
			Name:            util.GetEmqxBrokerHeadlessSvc(emqx),
			Namespace:       emqx.Namespace,
			OwnerReferences: ownerRefs,
		},
		Spec: corev1.ServiceSpec{
			Ports:     emqxPorts,
			Selector:  emqx.Spec.Labels,
			ClusterIP: corev1.ClusterIPNone,
		},
	}

	return svc
}

func newConfigMapForAcl(emqx *v1alpha1.EmqxBroker, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
	data := map[string]string{"acl.conf": emqx.Spec.LoadedModulesConf}
	cmForAcl := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            util.GetEmqxBrokerConfigMapForAcl(emqx),
			Namespace:       emqx.Namespace,
			OwnerReferences: ownerRefs,
		},
		Data: data,
	}

	return cmForAcl
}

func newConfigMapForLoadedMoudles(emqx *v1alpha1.EmqxBroker, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
	data := map[string]string{"loaded_modules": emqx.Spec.LoadedModulesConf}
	cmForPM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            util.GetEmqxBrokerConfigMapForLM(emqx),
			Namespace:       emqx.Namespace,
			OwnerReferences: ownerRefs,
		},
		Data: data,
	}

	return cmForPM
}

func newConfigMapForLoadedPlugins(emqx *v1alpha1.EmqxBroker, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
	data := map[string]string{"loaded_plugins": emqx.Spec.LoadedPluginConf}
	cmForPG := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            util.GetEmqxBrokerConfigMapForPG(emqx),
			Namespace:       emqx.Namespace,
			OwnerReferences: ownerRefs,
		},
		Data: data,
	}

	return cmForPG
}

func newEmqxBrokerStatefulSet(emqx *v1alpha1.EmqxBroker, labels map[string]string, ownerRefs []metav1.OwnerReference) *appsv1.StatefulSet {
	name := util.GetEmqxBrokerName(emqx)
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

	env := mergeDefaultEnv(emqx)

	// TODO
	labels = map[string]string{}

	volumeMounts := getEmqxBrokerVolumeMounts(emqx)
	volumes := getEmqxBrokerVolumes(emqx)

	storageSpec := emqx.Spec.Storage
	// volumeClaimTemplates := getVolumeClaimTemplates(storageSpec.VolumeClaimTemplates, emqx, ownerRefs)

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			Labels:          labels,
			OwnerReferences: ownerRefs,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: util.GetEmqxBrokerHeadlessSvc(emqx),
			Replicas:    emqx.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: emqx.Spec.Labels,
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			// VolumeClaimTemplates: volumeClaimTemplates,
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
					ServiceAccountName: emqx.Spec.ServiceAccountName,
					Tolerations:        emqx.Spec.ToleRations,
					NodeSelector:       emqx.Spec.NodeSelector,
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

	pvcList := []string{EMQX_LOG_NAME, EMQX_DATA_NAME}
	for _, item := range pvcList {
		pvcTemplate := MakeVolumeClaimTemplate(storageSpec.VolumeClaimTemplate, emqx)
		if pvcTemplate.Name == "" {
			pvcTemplate.Name = util.GetPvcName(item)
		}
		if storageSpec.VolumeClaimTemplate.Spec.AccessModes == nil {
			pvcTemplate.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
		} else {
			pvcTemplate.Spec.AccessModes = storageSpec.VolumeClaimTemplate.Spec.AccessModes
		}
		pvcTemplate.Spec.Resources = storageSpec.VolumeClaimTemplate.Spec.Resources
		pvcTemplate.Spec.Selector = storageSpec.VolumeClaimTemplate.Spec.Selector
		sts.Spec.VolumeClaimTemplates = append(sts.Spec.VolumeClaimTemplates, *pvcTemplate)
	}

	return sts
}

func getEmqxBrokerVolumeMounts(emqx *v1alpha1.EmqxBroker) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{}
	if emqx.Spec.License != "" {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				Name:      util.GetEmqxBrokerSecret(emqx),
				MountPath: EMQX_LIC_DIR,
				SubPath:   EMQX_LIC_SUBPATH,
				ReadOnly:  true,
			},
		)
	}
	if emqx.Spec.AclConf != "" {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				Name:      util.GetEmqxBrokerConfigMapForAcl(emqx),
				MountPath: EMQX_ACL_CONF_DIR,
				SubPath:   EMQX_ACL_CONF_SUBPATH,
			},
		)
	}
	if emqx.Spec.LoadedModulesConf != "" {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				Name:      util.GetEmqxBrokerConfigMapForLM(emqx),
				MountPath: EMQX_LOADED_MODULES_DIR,
				SubPath:   EMQX_LOADED_MODULES_SUBPATH,
			},
		)
	}
	if emqx.Spec.LoadedPluginConf != "" {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				Name:      util.GetEmqxBrokerConfigMapForPG(emqx),
				MountPath: EMQX_LOADED_PLUGINS_DIR,
				SubPath:   EMQX_LOADED_PLUGINS_SUBPATH,
			},
		)
	}
	return volumeMounts
}

func getEmqxBrokerVolumes(emqx *v1alpha1.EmqxBroker) []corev1.Volume {
	volumes := []corev1.Volume{}
	if emqx.Spec.License != "" {
		volumes = append(volumes,
			corev1.Volume{
				Name: util.GetEmqxBrokerSecret(emqx),
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: util.GetEmqxBrokerSecret(emqx),
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
				Name: util.GetEmqxBrokerConfigMapForAcl(emqx),
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: util.GetEmqxBrokerConfigMapForAcl(emqx),
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
				Name: util.GetEmqxBrokerConfigMapForPG(emqx),
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: util.GetEmqxBrokerConfigMapForPG(emqx),
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
	if emqx.Spec.LoadedModulesConf != "" {
		volumes = append(volumes,
			corev1.Volume{
				Name: util.GetEmqxBrokerConfigMapForLM(emqx),
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: util.GetEmqxBrokerConfigMapForLM(emqx),
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
	return volumes

}

func mergeDefaultEnv(emqx *v1alpha1.EmqxBroker) []corev1.EnvVar {
	defaultEnv := []corev1.EnvVar{
		{
			Name:  "EMQX_NAME",
			Value: util.GetEmqxBrokerName(emqx),
		},
		{
			Name:  "EMQX_CLUSTER__DISCOVERY",
			Value: "k8s",
		},
		{
			Name:  "EMQX_CLUSTER__K8S__APP_NAME",
			Value: util.GetEmqxBrokerName(emqx),
		},
		{
			Name:  "EMQX_CLUSTER__K8S__SERVICE_NAME",
			Value: util.GetEmqxBrokerHeadlessSvc(emqx),
		},
		{
			Name:  "EMQX_CLUSTER__K8S__NAMESPACE",
			Value: emqx.ObjectMeta.Namespace,
		},
		{
			Name:  "EMQX_CLUSTER__K8S__APISERVER",
			Value: "https://kubernetes.default.svc:443",
		},
		{
			Name:  "EMQX_CLUSTER__K8S__ADDRESS_TYPE",
			Value: "hostname",
		},
		{
			Name:  "EMQX_CLUSTER__K8S__SUFFIX",
			Value: "svc.cluster.local",
		},
	}

	// return append(emqx.Spec.Env, defaultEnv...)
	return append(defaultEnv, emqx.Spec.Env...)
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

// func getVolumeClaimTemplates(e v1alpha1.EmbeddedPersistentVolumeClaim, emqx *v1alpha1.EmqxBroker, ownerRefs []metav1.OwnerReference) []corev1.PersistentVolumeClaim {
// 	pvcList := []string{EMQX_LOG_NAME, EMQX_DATA_NAME}
// 	for _, pvc := range pvcList {
// 		pvcTemplate := getVolumeClaimTemplate(emqx.Spec.Storage.VolumeClaimTemplates, emqx, ownerRefs)
// 		if pvcTemplate.Name == "" {
// 			pvcTemplate.Name = util.GetPvcName(pvc)
// 		}
// 		if emqx.Spec.Storage..VolumeClaimTemplate.Spec.AccessModes == nil {
// 			pvcTemplate.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
// 		} else {
// 			pvcTemplate.Spec.AccessModes = storageSpec.VolumeClaimTemplate.Spec.AccessModes
// 		}
// 		pvcTemplate.Spec.Resources = storageSpec.VolumeClaimTemplate.Spec.Resources
// 		pvcTemplate.Spec.Selector = storageSpec.VolumeClaimTemplate.Spec.Selector
// 		statefulsetSpec.VolumeClaimTemplates = append(statefulsetSpec.VolumeClaimTemplates, *pvcTemplate)
// 	}
// 	return pvcs
// }

// func getVolumeClaimTemplate(e v1alpha1.EmbeddedPersistentVolumeClaim, emqx *v1alpha1.EmqxBroker, ownerRefs []metav1.OwnerReference) *corev1.PersistentVolumeClaim {
// 	pvc := corev1.PersistentVolumeClaim{
// 		TypeMeta: metav1.TypeMeta{
// 			APIVersion: e.APIVersion,
// 			Kind:       e.Kind,
// 		},
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:            e.Name,
// 			Namespace:       emqx.Namespace,
// 			Annotations:     emqx.ObjectMeta.Annotations,
// 			OwnerReferences: ownerRefs,
// 		},
// 		Spec:   e.Spec,
// 		Status: e.Status,
// 	}
// 	return &pvc
// }

func MakeVolumeClaimTemplate(e v1alpha1.EmbeddedPersistentVolumeClaim, instance *v1alpha1.EmqxBroker) *corev1.PersistentVolumeClaim {
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
