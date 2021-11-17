package service

import (
	"github.com/emqx/emqx-operator/api/v1alpha2"
	"github.com/emqx/emqx-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func newSecretForCR(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Secret {
	stringData := map[string]string{"emqx.lic": emqx.GetLicense()}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            emqx.GetSecretName(),
			Namespace:       emqx.GetNamespace(),
			OwnerReferences: ownerRefs,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: stringData,
	}

	return secret

}

func newHeadLessSvcForCR(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Service {
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
			Name:            emqx.GetHeadlessServiceName(),
			Namespace:       emqx.GetNamespace(),
			OwnerReferences: ownerRefs,
		},
		Spec: corev1.ServiceSpec{
			Ports:     emqxPorts,
			Selector:  emqx.GetLabels(),
			ClusterIP: corev1.ClusterIPNone,
		},
	}

	return svc
}

func newConfigMapForAcl(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
	data := map[string]string{"acl.conf": emqx.GetAclConf()}
	cmForAcl := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            emqx.GetAclConfName(),
			Namespace:       emqx.GetNamespace(),
			OwnerReferences: ownerRefs,
		},
		Data: data,
	}

	return cmForAcl
}

func newConfigMapForLoadedMoudles(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
	data := map[string]string{"loaded_modules": emqx.GetLoadedModulesConf()}
	cmForPM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            emqx.GetLoadedModulesConfName(),
			Namespace:       emqx.GetNamespace(),
			OwnerReferences: ownerRefs,
		},
		Data: data,
	}

	return cmForPM
}

func newConfigMapForLoadedPlugins(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
	data := map[string]string{"loaded_plugins": emqx.GetLoadedPluginConf()}
	cmForPG := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            emqx.GetLoadedPluginConfName(),
			Namespace:       emqx.GetNamespace(),
			OwnerReferences: ownerRefs,
		},
		Data: data,
	}

	return cmForPG
}

func newEmqxBrokerStatefulSet(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *appsv1.StatefulSet {
	name := emqx.GetName()
	namespace := emqx.GetNamespace()

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

	env := mergeEnv(emqx)

	// TODO
	labels = map[string]string{}

	volumeMounts := getEmqxBrokerVolumeMounts(emqx)
	volumes := getEmqxBrokerVolumes(emqx)

	storageSpec := emqx.GetStorage()
	// volumeClaimTemplates := getVolumeClaimTemplates(storageSpec.VolumeClaimTemplates, emqx, ownerRefs)

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			Labels:          labels,
			OwnerReferences: ownerRefs,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: emqx.GetHeadlessServiceName(),
			Replicas:    emqx.GetReplicas(),
			Selector: &metav1.LabelSelector{
				MatchLabels: emqx.GetLabels(),
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			// VolumeClaimTemplates: volumeClaimTemplates,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					// TODO merge labels
					Labels: emqx.GetLabels(),
					// TODO
					// Annotations:
				},
				Spec: corev1.PodSpec{
					// TODO initContainers
					// InitContainers

					// TODO
					// Affinity: getAffinity(rc.Spec.Affinity, labels),
					ServiceAccountName: emqx.GetServiceAccountName(),
					Tolerations:        emqx.GetToleRations(),
					NodeSelector:       emqx.GetNodeSelector(),
					Containers: []corev1.Container{
						{
							Name:            EMQX_NAME,
							Image:           emqx.GetImage(),
							ImagePullPolicy: getPullPolicy(emqx.GetImagePullPolicy()),
							Resources:       emqx.GetResource(),
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

func getEmqxBrokerVolumeMounts(emqx v1alpha2.Emqx) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{}
	if emqx.GetLicense() != "" {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				Name:      emqx.GetSecretName(),
				MountPath: EMQX_LIC_DIR,
				SubPath:   EMQX_LIC_SUBPATH,
				ReadOnly:  true,
			},
		)
	}
	if emqx.GetAclConf() != "" {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				Name:      emqx.GetAclConfName(),
				MountPath: EMQX_ACL_CONF_DIR,
				SubPath:   EMQX_ACL_CONF_SUBPATH,
			},
		)
	}
	if emqx.GetLoadedModulesConf() != "" {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				Name:      emqx.GetLoadedModulesConfName(),
				MountPath: EMQX_LOADED_MODULES_DIR,
				SubPath:   EMQX_LOADED_MODULES_SUBPATH,
			},
		)
	}
	if emqx.GetLoadedPluginConf() != "" {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				Name:      emqx.GetLoadedPluginConfName(),
				MountPath: EMQX_LOADED_PLUGINS_DIR,
				SubPath:   EMQX_LOADED_PLUGINS_SUBPATH,
			},
		)
	}
	return volumeMounts
}

func getEmqxBrokerVolumes(emqx v1alpha2.Emqx) []corev1.Volume {
	volumes := []corev1.Volume{}
	if emqx.GetLicense() != "" {
		volumes = append(volumes,
			corev1.Volume{
				Name: emqx.GetSecretName(),
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: emqx.GetSecretName(),
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

	if emqx.GetAclConf() != "" {
		volumes = append(volumes,
			corev1.Volume{
				Name: emqx.GetAclConfName(),
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: emqx.GetAclConfName(),
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
	if emqx.GetLoadedPluginConf() != "" {
		volumes = append(volumes,
			corev1.Volume{
				Name: emqx.GetLoadedPluginConfName(),
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: emqx.GetLoadedPluginConfName(),
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
	if emqx.GetLoadedModulesConf() != "" {
		volumes = append(volumes,
			corev1.Volume{
				Name: emqx.GetLoadedModulesConfName(),
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: emqx.GetLoadedModulesConfName(),
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

func Contains(Env []corev1.EnvVar, Name string) int {
	for index, value := range Env {
		if value.Name == Name {
			return index
		}
	}
	return -1
}

func mergeEnv(emqx v1alpha2.Emqx) []corev1.EnvVar {
	env := emqx.GetEnv()
	clusterEnv := getDefaultClusterConfig(emqx)
	for index, value := range clusterEnv {
		r := Contains(env, value.Name)
		if r == -1 {
			env = append(env, clusterEnv[index])
		}
	}

	return env
}

func getDefaultClusterConfig(emqx v1alpha2.Emqx) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "EMQX_NAME",
			Value: emqx.GetName(),
		},
		{
			Name:  "EMQX_CLUSTER__DISCOVERY",
			Value: "k8s",
		},
		{
			Name:  "EMQX_CLUSTER__K8S__APP_NAME",
			Value: emqx.GetName(),
		},
		{
			Name:  "EMQX_CLUSTER__K8S__SERVICE_NAME",
			Value: emqx.GetHeadlessServiceName(),
		},
		{
			Name:  "EMQX_CLUSTER__K8S__NAMESPACE",
			Value: emqx.GetNamespace(),
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
		return corev1.PullIfNotPresent
	}
	return specPolicy
}

// func getVolumeClaimTemplates(e v1alpha2.EmbeddedPersistentVolumeClaim, emqx *v1alpha2.EmqxBroker, ownerRefs []metav1.OwnerReference) []corev1.PersistentVolumeClaim {
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

// func getVolumeClaimTemplate(e v1alpha2.EmbeddedPersistentVolumeClaim, emqx *v1alpha2.EmqxBroker, ownerRefs []metav1.OwnerReference) *corev1.PersistentVolumeClaim {
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

func MakeVolumeClaimTemplate(e v1alpha2.EmbeddedPersistentVolumeClaim, emqx v1alpha2.Emqx) *corev1.PersistentVolumeClaim {
	boolTrue := true
	pvc := corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: e.APIVersion,
			Kind:       e.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        e.Name,
			Namespace:   emqx.GetNamespace(),
			Annotations: emqx.GetAnnotations(),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         emqx.GetAPIVersion(),
					BlockOwnerDeletion: &boolTrue,
					Controller:         &boolTrue,
					Kind:               emqx.GetKind(),
					Name:               emqx.GetName(),
					UID:                emqx.GetUID(),
				},
			},
		},
		Spec:   e.Spec,
		Status: e.Status,
	}
	return &pvc
}
