package service

import (
	"reflect"

	"github.com/emqx/emqx-operator/api/v1alpha2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	EMQX_LISTENER_DEFAULT = map[string]string{
		"mqtt":      "EMQX_LISTENERS__TCP__DEFAULT",
		"mqtts":     "EMQX_LISTENERS__SSL__DEFAULT_NAME",
		"ws":        "EMQX_LISTENERS__WS__DEFAULT_NAME",
		"wss":       "EMQX_LISTENERS__WSS__DEFAULT_NAME",
		"dashboard": "EMQX_DASHBOARD__LISTENER__HTTP_NAME",
		"api":       "EMQX_MANAGEMENT__LISTENER__HTTP_NAME",
	}
)

func NewSecretForCR(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Secret {
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

func NewHeadLessSvcForCR(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Service {
	var emqxPorts []corev1.ServicePort
	if emqx.GetListener() == nil {
		emqxPorts = convertPorts(generateDefaultServicePorts())
	} else {
		listener := emqx.GetListener()
		emqxPorts = mergeServicePorts(listener.Ports)
	}

	// labels = util.MergeLabels(labels, generateSelectorLabels(util.SentinelRoleName, cluster.Name))
	// labels = map[string]string{}

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

func NewListenerSvcForCR(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Service {
	listener := emqx.GetListener()
	listenerSvcPorts := mergeServicePorts(listener.Ports)

	listenerSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            emqx.GetListenerServiceName(),
			Namespace:       emqx.GetNamespace(),
			OwnerReferences: ownerRefs,
		},
		Spec: corev1.ServiceSpec{
			Type:     listener.Type,
			Ports:    listenerSvcPorts,
			Selector: emqx.GetLabels(),
		},
	}
	return listenerSvc
}

func NewConfigMapForAcl(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
	acl := emqx.GetAcl()
	cmForAcl := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            acl["name"],
			Namespace:       emqx.GetNamespace(),
			OwnerReferences: ownerRefs,
		},
		Data: map[string]string{"acl.conf": acl["conf"]},
	}

	return cmForAcl
}

func NewConfigMapForLoadedMoudles(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
	modules := emqx.GetLoadedModules()
	cmForPM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            modules["name"],
			Namespace:       emqx.GetNamespace(),
			OwnerReferences: ownerRefs,
		},
		Data: map[string]string{"loaded_modules": modules["conf"]},
	}

	return cmForPM
}

func NewConfigMapForLoadedPlugins(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
	plugins := emqx.GetLoadedPlugins()
	cmForPG := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            plugins["name"],
			Namespace:       emqx.GetNamespace(),
			OwnerReferences: ownerRefs,
		},
		Data: map[string]string{"loaded_plugins": plugins["conf"]},
	}

	return cmForPG
}

func NewEmqxStatefulSet(emqx v1alpha2.Emqx, labels map[string]string, ownerRefs []metav1.OwnerReference) *appsv1.StatefulSet {
	name := emqx.GetName()
	namespace := emqx.GetNamespace()

	ports := getContainerPorts(emqx)

	env := mergeEnv(emqx)

	// TODO
	labels = map[string]string{}

	var emqxUserGroup int64 = 1000
	var runAsNonRoot bool = true
	var fsGroupChangeAlways corev1.PodFSGroupChangePolicy = "Always"

	securityContext := &corev1.PodSecurityContext{
		FSGroup:             &emqxUserGroup,
		FSGroupChangePolicy: &fsGroupChangeAlways,
		RunAsNonRoot:        &runAsNonRoot,
		RunAsUser:           &emqxUserGroup,
		SupplementalGroups:  []int64{emqxUserGroup},
	}

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
			PodManagementPolicy:  appsv1.ParallelPodManagement,
			VolumeClaimTemplates: getVolumeClaimTemplates(emqx),
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
					SecurityContext:    securityContext,
					Tolerations:        emqx.GetToleRations(),
					NodeSelector:       emqx.GetNodeSelector(),
					Containers: []corev1.Container{
						{
							Name:            EMQX_NAME,
							Image:           emqx.GetImage(),
							ImagePullPolicy: getPullPolicy(emqx.GetImagePullPolicy()),
							Resources:       emqx.GetResource(),
							Env:             env,
							Ports:           ports,
							VolumeMounts:    getEmqxVolumeMounts(emqx),
						},
					},
					Volumes: getEmqxVolumes(emqx),
				},
			},
		},
	}

	return sts
}

func getVolumeClaimTemplates(emqx v1alpha2.Emqx) []corev1.PersistentVolumeClaim {
	storageSpec := emqx.GetStorage()
	if reflect.ValueOf(storageSpec).IsNil() {
		return []corev1.PersistentVolumeClaim{}
	} else {
		return []corev1.PersistentVolumeClaim{
			genVolumeClaimTemplate(emqx, emqx.GetDataVolumeName()),
			genVolumeClaimTemplate(emqx, emqx.GetLogVolumeName()),
		}
	}
}

func getEmqxVolumeMounts(emqx v1alpha2.Emqx) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{}
	volumeMounts = append(volumeMounts,
		corev1.VolumeMount{
			Name:      emqx.GetDataVolumeName(),
			MountPath: EMQX_DATA_DIR,
		},
	)
	volumeMounts = append(volumeMounts,
		corev1.VolumeMount{
			Name:      emqx.GetLogVolumeName(),
			MountPath: EMQX_LOG_DIR,
		},
	)
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
	acl := emqx.GetAcl()
	volumeMounts = append(volumeMounts,
		corev1.VolumeMount{
			Name:      acl["name"],
			MountPath: acl["mountPath"],
			SubPath:   acl["subPath"],
		},
	)
	modules := emqx.GetLoadedModules()
	volumeMounts = append(volumeMounts,
		corev1.VolumeMount{
			Name:      modules["name"],
			MountPath: modules["mountPath"],
			SubPath:   modules["subPath"],
		},
	)
	plugins := emqx.GetLoadedPlugins()
	volumeMounts = append(volumeMounts,
		corev1.VolumeMount{
			Name:      plugins["name"],
			MountPath: plugins["mountPath"],
			SubPath:   plugins["subPath"],
		},
	)
	return volumeMounts
}

func getEmqxVolumes(emqx v1alpha2.Emqx) []corev1.Volume {
	volumes := []corev1.Volume{}
	storageSpec := emqx.GetStorage()
	if reflect.ValueOf(storageSpec).IsNil() {
		volumes = append(volumes,
			corev1.Volume{
				Name: emqx.GetDataVolumeName(),
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		)
		volumes = append(volumes,
			corev1.Volume{
				Name: emqx.GetLogVolumeName(),
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		)
	}
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
			},
		)
	}

	acl := emqx.GetAcl()
	volumes = append(volumes,
		corev1.Volume{
			Name: acl["name"],
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: acl["name"],
					},
					Items: []corev1.KeyToPath{
						{
							Key:  acl["subPath"],
							Path: acl["subPath"],
						},
					},
				},
			},
		},
	)

	plugins := emqx.GetLoadedPlugins()
	volumes = append(volumes,
		corev1.Volume{
			Name: plugins["name"],
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: plugins["name"],
					},
					Items: []corev1.KeyToPath{
						{
							Key:  plugins["subPath"],
							Path: plugins["subPath"],
						},
					},
				},
			},
		},
	)

	modules := emqx.GetLoadedModules()
	volumes = append(volumes,
		corev1.Volume{
			Name: modules["name"],
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: modules["name"],
					},
					Items: []corev1.KeyToPath{
						{
							Key:  modules["subPath"],
							Path: modules["subPath"],
						},
					},
				},
			},
		},
	)
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
	if emqx.GetListener() != nil {
		listener := emqx.GetListener()
		ports := mergeServicePorts(listener.Ports)
		for _, port := range ports {
			env = append(env,
				corev1.EnvVar{

					Name:  EMQX_LISTENER_DEFAULT[port.Name],
					Value: string(port.Port),
				})
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

func getContainerPorts(emqx v1alpha2.Emqx) []corev1.ContainerPort {
	var ports []corev1.ServicePort
	containerPorts := []corev1.ContainerPort{}
	if emqx.GetListener() != nil {
		listener := emqx.GetListener()
		ports = mergeServicePorts(listener.Ports)
	} else {
		ports = convertPorts(generateDefaultServicePorts())
	}
	for _, port := range ports {
		containerPorts = append(containerPorts,
			corev1.ContainerPort{
				ContainerPort: port.Port,
			})
	}
	return containerPorts
	// return []corev1.ContainerPort{
	// 	{
	// 		ContainerPort: 1883,
	// 	},
	// 	{
	// 		ContainerPort: 8883,
	// 	},
	// 	{
	// 		ContainerPort: 8081,
	// 	},
	// 	{
	// 		ContainerPort: 8083,
	// 	},
	// 	{
	// 		ContainerPort: 8084,
	// 	},
	// }
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

func genVolumeClaimTemplate(emqx v1alpha2.Emqx, Name string) corev1.PersistentVolumeClaim {
	template := emqx.GetStorage().VolumeClaimTemplate
	boolTrue := true
	pvc := corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: template.APIVersion,
			Kind:       template.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        Name,
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
		Spec:   template.Spec,
		Status: template.Status,
	}
	if pvc.Spec.AccessModes == nil {
		pvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}
	return pvc
}

func mergeServicePorts(ports []corev1.ServicePort) []corev1.ServicePort {

	mergePorts := func(port corev1.ServicePort, ports []*corev1.ServicePort) []*corev1.ServicePort {
		for _, item := range ports {
			if item.Name == port.Name {
				item.Port = port.Port
				item.Protocol = port.Protocol
				item.TargetPort = intstr.IntOrString{
					Type:   0,
					IntVal: port.TargetPort.IntVal,
				}
			}
		}
		// TODO Bug to fix
		return ports
	}

	servicePorts := generateDefaultServicePorts()
	for _, port := range ports {
		mergePorts(port, servicePorts)
	}
	return convertPorts(servicePorts)
}

func generateDefaultServicePorts() []*corev1.ServicePort {
	defaultPorts := []*corev1.ServicePort{
		{
			Name:     EMQX_LISTENERS__TCP__DEFAULT_NAME,
			Port:     EMQX_LISTENERS__TCP__DEFAULT_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: EMQX_LISTENERS__TCP__DEFAULT_PORT,
			},
		},
		{
			Name:     EMQX_LISTENERS__SSL__DEFAULT_NAME,
			Port:     EMQX_LISTENERS__SSL__DEFAULT_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: EMQX_LISTENERS__SSL__DEFAULT_PORT,
			},
		},
		{
			Name:     EMQX_LISTENERS__WS__DEFAULT_NAME,
			Port:     EMQX_LISTENERS__WS__DEFAULT_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: EMQX_LISTENERS__WS__DEFAULT_PORT,
			},
		},
		{
			Name:     EMQX_LISTENERS__WSS__DEFAULT_NAME,
			Port:     EMQX_LISTENERS__WSS__DEFAULT_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: EMQX_LISTENERS__WSS__DEFAULT_PORT,
			},
		},
		{
			Name:     EMQX_DASHBOARD__LISTENER__HTTP_NAME,
			Port:     EMQX_DASHBOARD__LISTENER__HTTP_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: EMQX_DASHBOARD__LISTENER__HTTP_PORT,
			},
		},
		{
			Name:     EMQX_MANAGEMENT__LISTENER__HTTP_NAME,
			Port:     EMQX_MANAGEMENT__LISTENER__HTTP_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: EMQX_MANAGEMENT__LISTENER__HTTP_PORT,
			},
		},
	}
	return defaultPorts
}

func convertPorts(ports []*corev1.ServicePort) []corev1.ServicePort {
	var res []corev1.ServicePort
	for _, port := range ports {
		res = append(res, *port)
	}
	return res
}
