package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/emqx/emqx-operator/apis/apps/v1beta1"
	"github.com/emqx/emqx-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Generate(emqx v1beta1.Emqx) []client.Object {
	var resources []client.Object

	sts := generateStatefulSetDef(emqx)

	sa, role, roleBinding, sts := generateRBAC(emqx, sts)
	resources = append(resources, sa, role, roleBinding)

	headlessSvc, svc, sts := generateSvc(emqx, sts)
	resources = append(resources, headlessSvc, svc)

	acl, sts := generateConfigMapForAcl(emqx, sts)
	resources = append(resources, acl)

	plugins, sts := generateConfigMapForPlugins(emqx, sts)
	resources = append(resources, plugins)

	module, sts := generateConfigMapForModules(emqx, sts)
	resources = append(resources, module)

	if emqxEnterprise, ok := emqx.(*v1beta1.EmqxEnterprise); ok {
		if emqxEnterprise.GetLicense() != "" {
			var license *corev1.Secret
			license, sts = generateSecretForLicense(*emqxEnterprise, sts)
			resources = append(resources, license)
		}
	}

	// add logic for the telegraf pod
	if emqx.GetTelegrafTemplate() != nil {
		var cmForTelegraf *corev1.ConfigMap
		cmForTelegraf, sts = generateContainerForTelegraf(emqx, sts)
		resources = append(resources, cmForTelegraf)
	}

	resources = append(resources, sts)

	ownerRef := metav1.NewControllerRef(emqx, v1beta1.VersionKind(emqx.GetKind()))
	for _, resource := range resources {
		addOwnerRefToObject(resource, *ownerRef)
	}

	return resources
}

func generateStatefulSetDef(emqx v1beta1.Emqx) *appsv1.StatefulSet {
	sts := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      emqx.GetName(),
			Namespace: emqx.GetNamespace(),
			Labels:    emqx.GetLabels(),
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: emqx.GetReplicas(),
			Selector: &metav1.LabelSelector{
				MatchLabels: emqx.GetLabels(),
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      emqx.GetLabels(),
					Annotations: emqx.GetAnnotations(),
				},
				Spec: corev1.PodSpec{
					Affinity:     emqx.GetAffinity(),
					Tolerations:  emqx.GetToleRations(),
					NodeSelector: emqx.GetNodeSelector(),
					Containers: []corev1.Container{
						{
							Name:            "emqx",
							Image:           emqx.GetImage(),
							ImagePullPolicy: emqx.GetImagePullPolicy(),
							Resources:       emqx.GetResource(),
							Env:             emqx.GetEnv(),
						},
					},
				},
			},
		},
	}

	return generateVolume(emqx, sts)
}

func generateContainerForTelegraf(emqx v1beta1.Emqx, sts *appsv1.StatefulSet) (*corev1.ConfigMap, *appsv1.StatefulSet) {

	telegrafTemplate := emqx.GetTelegrafTemplate()
	telegrafConfName := fmt.Sprintf("%s-%s", emqx.GetName(), "telegraf-config")
	logName := fmt.Sprintf("%s-%s", emqx.GetName(), "log")

	containerForTelegraf := corev1.Container{
		Name:            "telegraf",
		Image:           telegrafTemplate.Image,
		ImagePullPolicy: telegrafTemplate.ImagePullPolicy,
		Resources:       emqx.GetTelegrafTemplate().Resources,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      telegrafConfName,
				MountPath: "/etc/telegraf/telegraf.conf",
				SubPath:   "telegraf.conf",
				ReadOnly:  true,
			},
			{
				Name:      logName,
				MountPath: "/opt/emqx/log",
			},
		},
	}
	sts.Spec.Template.Spec.Containers = append(sts.Spec.Template.Spec.Containers, containerForTelegraf)

	telegrafConfData := emqx.GetTelegrafTemplate().Conf
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    emqx.GetLabels(),
			Name:      telegrafConfName,
			Namespace: emqx.GetNamespace(),
		},
		Data: map[string]string{"telegraf.conf": *telegrafConfData},
	}

	sts.Spec.Template.Spec.Volumes = append(
		sts.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name: telegrafConfName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: telegrafConfName,
					},
				},
			},
		},
	)

	return cm, sts
}

func generateRBAC(emqx v1beta1.Emqx, sts *appsv1.StatefulSet) (*corev1.ServiceAccount, *rbacv1.Role, *rbacv1.RoleBinding, *appsv1.StatefulSet) {
	meta := metav1.ObjectMeta{
		Name:      emqx.GetServiceAccountName(),
		Namespace: emqx.GetNamespace(),
		Labels:    emqx.GetLabels(),
	}

	sa := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: meta,
	}

	role := &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: meta,
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "watch", "list"},
				APIGroups: []string{""},
				Resources: []string{"endpoints"},
			},
		},
	}

	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: meta,
		Subjects: []rbacv1.Subject{
			{
				Kind:      sa.Kind,
				Name:      sa.Name,
				Namespace: sa.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     role.Kind,
			Name:     role.Name,
		},
	}

	sts.Spec.Template.Spec.ServiceAccountName = sa.Name

	return sa, role, roleBinding, sts
}

func generateSvc(emqx v1beta1.Emqx, sts *appsv1.StatefulSet) (*corev1.Service, *corev1.Service, *appsv1.StatefulSet) {
	listener := emqx.GetListener()
	headlessSvc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    emqx.GetLabels(),
			Name:      emqx.GetHeadlessServiceName(),
			Namespace: emqx.GetNamespace(),
		},
		Spec: corev1.ServiceSpec{
			Selector:  emqx.GetLabels(),
			ClusterIP: corev1.ClusterIPNone,
		},
	}
	sts.Spec.ServiceName = headlessSvc.Name

	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    emqx.GetLabels(),
			Name:      emqx.GetName(),
			Namespace: emqx.GetNamespace(),
		},
		Spec: corev1.ServiceSpec{
			Type:                     listener.Type,
			LoadBalancerIP:           listener.LoadBalancerIP,
			LoadBalancerSourceRanges: listener.LoadBalancerSourceRanges,
			ExternalIPs:              listener.ExternalIPs,
			Selector:                 emqx.GetLabels(),
		},
	}

	container := sts.Spec.Template.Spec.Containers[0]

	ports := reflect.ValueOf(listener.Ports)
	nodePorts := reflect.ValueOf(listener.NodePorts)

	for i := 0; i < ports.NumField(); i++ {
		port := int32(ports.Field(i).Int())
		if port != 0 {
			name := strings.ToLower(ports.Type().Field(i).Name)
			nodePort := int32(nodePorts.Field(i).Int())

			var envName string
			switch name {
			default:
				envName = fmt.Sprintf("EMQX_LISTENER__%s__EXTERNAL", strings.ToUpper(name))
			case "mqtt":
				envName = "EMQX_LISTENER__TCP__EXTERNAL"
			case "mqtts":
				envName = "EMQX_LISTENER__SSL__EXTERNAL"
			case "api":
				envName = "EMQX_MANAGEMENT__LISTENER__HTTP"
			case "dashboard":
				envName = "EMQX_DASHBOARD__LISTENER__HTTP"
			}

			container.Env = append(container.Env, corev1.EnvVar{
				Name:  envName,
				Value: fmt.Sprint(port),
			})

			container.Ports = append(container.Ports, corev1.ContainerPort{
				Name:          name,
				ContainerPort: port,
				Protocol:      corev1.ProtocolTCP,
			})

			svc.Spec.Ports = append(svc.Spec.Ports, corev1.ServicePort{
				Name:     name,
				Port:     port,
				NodePort: nodePort,
				Protocol: "TCP",
				TargetPort: intstr.IntOrString{
					Type:   0,
					IntVal: port,
				},
			})
		}
	}
	sts.Spec.Template.Spec.Containers = []corev1.Container{container}
	return headlessSvc, svc, sts
}

func generateConfigMapForAcl(emqx v1beta1.Emqx, sts *appsv1.StatefulSet) (*corev1.ConfigMap, *appsv1.StatefulSet) {
	aclString := util.StringACL(emqx.GetACL())

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    emqx.GetLabels(),
			Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "acl"),
			Namespace: emqx.GetNamespace(),
		},
		Data: map[string]string{"acl.conf": aclString},
	}

	annotations := sts.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["ACL/Base64EncodeConfig"] = base64.StdEncoding.EncodeToString([]byte(aclString))
	sts.Annotations = annotations

	container := sts.Spec.Template.Spec.Containers[0]
	container.VolumeMounts = append(
		container.VolumeMounts,
		corev1.VolumeMount{
			Name:      cm.Name,
			MountPath: "/mounted/acl",
		},
	)
	container.Env = append(
		container.Env,
		corev1.EnvVar{
			Name:  "EMQX_ACL_FILE",
			Value: "/mounted/acl/acl.conf",
		},
	)
	sts.Spec.Template.Spec.Containers = []corev1.Container{container}

	sts.Spec.Template.Spec.Volumes = append(
		sts.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name: cm.Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cm.Name,
					},
				},
			},
		},
	)

	return cm, sts
}

func generateConfigMapForPlugins(emqx v1beta1.Emqx, sts *appsv1.StatefulSet) (*corev1.ConfigMap, *appsv1.StatefulSet) {
	loadedPluginsString := util.StringLoadedPlugins(emqx.GetPlugins())
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    emqx.GetLabels(),
			Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "loaded-plugins"),
			Namespace: emqx.GetNamespace(),
		},
		Data: map[string]string{"loaded_plugins": loadedPluginsString},
	}
	cm.SetGroupVersionKind(schema.GroupVersionKind{Kind: "ConfigMap", Version: "v1"})

	annotations := sts.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["LoadedPlugins/Base64EncodeConfig"] = base64.StdEncoding.EncodeToString([]byte(loadedPluginsString))
	sts.Annotations = annotations

	container := sts.Spec.Template.Spec.Containers[0]
	container.VolumeMounts = append(
		container.VolumeMounts,
		corev1.VolumeMount{
			Name:      cm.Name,
			MountPath: "/mounted/plugins",
		},
	)
	container.Env = append(
		container.Env,
		corev1.EnvVar{
			Name:  "EMQX_PLUGINS__LOADED_FILE",
			Value: "/mounted/plugins/loaded_plugins",
		},
	)
	sts.Spec.Template.Spec.Containers = []corev1.Container{container}

	sts.Spec.Template.Spec.Volumes = append(
		sts.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name: cm.Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cm.Name,
					},
				},
			},
		},
	)
	return cm, sts
}

func generateConfigMapForModules(emqx v1beta1.Emqx, sts *appsv1.StatefulSet) (*corev1.ConfigMap, *appsv1.StatefulSet) {
	var loadedModulesString string
	switch obj := emqx.(type) {
	case *v1beta1.EmqxBroker:
		loadedModulesString = util.StringEmqxBrokerLoadedModules(obj.Spec.Modules)
	case *v1beta1.EmqxEnterprise:
		data, _ := json.Marshal(obj.Spec.Modules)
		loadedModulesString = string(data)
	}
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    emqx.GetLabels(),
			Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "loaded-modules"),
			Namespace: emqx.GetNamespace(),
		},
		Data: map[string]string{"loaded_modules": loadedModulesString},
	}

	annotations := sts.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["LoadedModules/Base64EncodeConfig"] = base64.StdEncoding.EncodeToString([]byte(loadedModulesString))
	sts.Annotations = annotations

	container := sts.Spec.Template.Spec.Containers[0]
	container.VolumeMounts = append(
		container.VolumeMounts,
		corev1.VolumeMount{
			Name:      cm.Name,
			MountPath: "/mounted/modules",
		},
	)
	container.Env = append(
		container.Env,
		corev1.EnvVar{
			Name:  "EMQX_MODULES__LOADED_FILE",
			Value: "/mounted/modules/loaded_modules",
		},
	)
	sts.Spec.Template.Spec.Containers = []corev1.Container{container}

	sts.Spec.Template.Spec.Volumes = append(
		sts.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name: cm.Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cm.Name,
					},
				},
			},
		},
	)
	return cm, sts
}

func generateSecretForLicense(emqx v1beta1.EmqxEnterprise, sts *appsv1.StatefulSet) (*corev1.Secret, *appsv1.StatefulSet) {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    emqx.GetLabels(),
			Name:      fmt.Sprintf("%s-%s", emqx.GetName(), "license"),
			Namespace: emqx.GetNamespace(),
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{"emqx.lic": emqx.GetLicense()},
	}

	container := sts.Spec.Template.Spec.Containers[0]
	container.VolumeMounts = append(
		container.VolumeMounts,
		corev1.VolumeMount{
			Name:      secret.Name,
			MountPath: "/mounted/license",
			ReadOnly:  true,
		},
	)
	container.Env = append(
		container.Env,
		corev1.EnvVar{
			Name:  "EMQX_LICENSE__FILE",
			Value: "/mounted/license/emqx.lic",
		},
	)
	sts.Spec.Template.Spec.Containers = []corev1.Container{container}
	sts.Spec.Template.Spec.Volumes = append(
		sts.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name: secret.Name,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secret.Name,
				},
			},
		},
	)

	return secret, sts
}

func generateVolume(emqx v1beta1.Emqx, sts *appsv1.StatefulSet) *appsv1.StatefulSet {
	container := sts.Spec.Template.Spec.Containers[0]

	dataName := fmt.Sprintf("%s-%s", emqx.GetName(), "data")
	logName := fmt.Sprintf("%s-%s", emqx.GetName(), "log")

	container.VolumeMounts = append(
		container.VolumeMounts,
		corev1.VolumeMount{
			Name:      dataName,
			MountPath: "/opt/emqx/data",
		},
		corev1.VolumeMount{
			Name:      logName,
			MountPath: "/opt/emqx/log",
		},
	)

	if reflect.ValueOf(emqx.GetStorage()).IsNil() {
		sts.Spec.Template.Spec.Volumes = append(
			sts.Spec.Template.Spec.Volumes,
			genreateEmptyDirVolume(dataName),
			genreateEmptyDirVolume(logName),
		)
	} else {
		sts.Spec.VolumeClaimTemplates = append(
			sts.Spec.VolumeClaimTemplates,
			generateVolumeClaimTemplate(emqx, dataName),
			generateVolumeClaimTemplate(emqx, logName),
		)

	}

	container.VolumeMounts = append(container.VolumeMounts, emqx.GetExtraVolumeMounts()...)
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, emqx.GetExtraVolumes()...)

	sts.Spec.Template.Spec.Containers = []corev1.Container{container}
	return sts
}

func genreateEmptyDirVolume(Name string) corev1.Volume {
	return corev1.Volume{
		Name: Name,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
}

func generateVolumeClaimTemplate(emqx v1beta1.Emqx, Name string) corev1.PersistentVolumeClaim {
	template := emqx.GetStorage().VolumeClaimTemplate
	pvc := corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: template.APIVersion,
			Kind:       template.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      Name,
			Namespace: emqx.GetNamespace(),
		},
		Spec:   template.Spec,
		Status: template.Status,
	}
	if pvc.Spec.AccessModes == nil {
		pvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}
	if pvc.Spec.VolumeMode == nil {
		fileSystem := corev1.PersistentVolumeFilesystem
		pvc.Spec.VolumeMode = &fileSystem
	}
	return pvc
}

func addOwnerRefToObject(obj metav1.Object, ownerRef metav1.OwnerReference) {
	obj.SetOwnerReferences(append(obj.GetOwnerReferences(), ownerRef))
}
