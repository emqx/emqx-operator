/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v2alpha1

import (
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"

	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	"github.com/emqx/emqx-operator/internal/handler"
	"github.com/rory-z/go-hocon"
	"github.com/sethvargo/go-password/password"
	appsv1 "k8s.io/api/apps/v1"
)

const defUsername = "emqx_operator_controller"

func generateNodeCookieSecret(instance *appsv2alpha1.EMQX) *corev1.Secret {
	var cookie string

	config, _ := hocon.ParseString(instance.Spec.BootstrapConfig)
	cookie = config.GetString("node.cookie")
	if cookie == "" {
		cookie, _ = password.Generate(64, 10, 0, true, true)
	}

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        instance.NameOfNodeCookie(),
			Namespace:   instance.Namespace,
			Labels:      instance.Labels,
			Annotations: instance.Annotations,
		},
		StringData: map[string]string{
			"node_cookie": cookie,
		},
	}
}

func generateBootstrapUserSecret(instance *appsv2alpha1.EMQX) *corev1.Secret {
	bootstrapUsers := ""
	for _, apiKey := range instance.Spec.BootstrapAPIKeys {
		bootstrapUsers += apiKey.Key + ":" + apiKey.Secret + "\n"
	}

	defPassword, _ := password.Generate(64, 10, 0, true, true)
	bootstrapUsers += defUsername + ":" + defPassword

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        instance.NameOfBootStrapUser(),
			Namespace:   instance.Namespace,
			Labels:      instance.Labels,
			Annotations: instance.Annotations,
		},
		StringData: map[string]string{
			"bootstrap_user": bootstrapUsers,
		},
	}
}

func generateBootstrapConfigMap(instance *appsv2alpha1.EMQX) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        instance.NameOfBootStrapConfig(),
			Namespace:   instance.Namespace,
			Labels:      instance.Labels,
			Annotations: instance.Annotations,
		},
		Data: map[string]string{
			"emqx.conf": instance.Spec.BootstrapConfig,
		},
	}
}

func generateHeadlessService(instance *appsv2alpha1.EMQX) *corev1.Service {
	headlessSvc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.NameOfHeadlessService(),
			Namespace: instance.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type:                     corev1.ServiceTypeClusterIP,
			ClusterIP:                corev1.ClusterIPNone,
			SessionAffinity:          corev1.ServiceAffinityNone,
			PublishNotReadyAddresses: true,
			Ports: []corev1.ServicePort{
				{
					Name:       "ekka",
					Port:       4370,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(4370),
				},
			},
			Selector: instance.Spec.CoreTemplate.Labels,
		},
	}
	return headlessSvc
}

func generateDashboardService(instance *appsv2alpha1.EMQX) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.Namespace,
			Name:        instance.Spec.DashboardServiceTemplate.Name,
			Labels:      instance.Spec.DashboardServiceTemplate.Labels,
			Annotations: instance.Spec.DashboardServiceTemplate.Annotations,
		},
		Spec: instance.Spec.DashboardServiceTemplate.Spec,
	}
}

func generateListenerService(instance *appsv2alpha1.EMQX, listenerPorts []corev1.ServicePort) *corev1.Service {
	instance.Spec.ListenersServiceTemplate.Spec.Ports = appsv2alpha1.MergeServicePorts(
		instance.Spec.ListenersServiceTemplate.Spec.Ports,
		listenerPorts,
	)

	if len(instance.Spec.ListenersServiceTemplate.Spec.Ports) == 0 {
		return nil
	}
	instance.Spec.ListenersServiceTemplate.Spec.Selector = instance.Spec.ReplicantTemplate.Labels

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.Namespace,
			Name:        instance.Spec.ListenersServiceTemplate.Name,
			Labels:      instance.Spec.ListenersServiceTemplate.Labels,
			Annotations: instance.Spec.ListenersServiceTemplate.Annotations,
		},
		Spec: instance.Spec.ListenersServiceTemplate.Spec,
	}
}

func generateStatefulSet(instance *appsv2alpha1.EMQX) *appsv1.StatefulSet {
	emqxContainer := corev1.Container{
		Name:            EMQXContainerName,
		Image:           instance.Spec.Image,
		ImagePullPolicy: corev1.PullPolicy(instance.Spec.ImagePullPolicy),
		Command:         instance.Spec.CoreTemplate.Spec.Command,
		Args:            instance.Spec.CoreTemplate.Spec.Args,
		Ports:           instance.Spec.CoreTemplate.Spec.Ports,
		Env: append([]corev1.EnvVar{
			{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.name",
					},
				},
			},
			{
				Name: "POD_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.namespace",
					},
				},
			},
			{
				Name: "STS_HEADLESS_SERVICE_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.annotations['apps.emqx.io/headless-service-name']",
					},
				},
			},
			{
				Name:  "EMQX_HOST",
				Value: "$(POD_NAME).$(STS_HEADLESS_SERVICE_NAME).$(POD_NAMESPACE).svc.cluster.local",
			},
			{
				Name:  "EMQX_NODE__DB_ROLE",
				Value: "core",
			},
		}, instance.Spec.CoreTemplate.Spec.Env...),
		EnvFrom:         instance.Spec.CoreTemplate.Spec.EnvFrom,
		Resources:       instance.Spec.CoreTemplate.Spec.Resources,
		SecurityContext: instance.Spec.CoreTemplate.Spec.ContainerSecurityContext,
		LivenessProbe:   instance.Spec.CoreTemplate.Spec.LivenessProbe,
		ReadinessProbe:  instance.Spec.CoreTemplate.Spec.ReadinessProbe,
		StartupProbe:    instance.Spec.CoreTemplate.Spec.StartupProbe,
		Lifecycle:       instance.Spec.CoreTemplate.Spec.Lifecycle,
		VolumeMounts: append(instance.Spec.CoreTemplate.Spec.ExtraVolumeMounts, corev1.VolumeMount{
			Name:      instance.NameOfCoreNodeData(),
			MountPath: "/opt/emqx/data",
		}),
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: corev1.TerminationMessageReadFile,
	}

	containers := append(
		[]corev1.Container{emqxContainer},
		instance.Spec.CoreTemplate.Spec.ExtraContainers...,
	)

	podAnnotation := instance.Spec.CoreTemplate.Annotations
	if podAnnotation == nil {
		podAnnotation = make(map[string]string)
	}
	podAnnotation["apps.emqx.io/headless-service-name"] = instance.NameOfHeadlessService()
	podAnnotation[handler.ManageContainersAnnotation] = generateAnnotationByContainers(containers)

	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      instance.Spec.CoreTemplate.Labels,
			Annotations: podAnnotation,
		},
		Spec: corev1.PodSpec{
			ImagePullSecrets:              instance.Spec.ImagePullSecrets,
			SecurityContext:               instance.Spec.CoreTemplate.Spec.PodSecurityContext,
			Affinity:                      instance.Spec.CoreTemplate.Spec.Affinity,
			Tolerations:                   instance.Spec.CoreTemplate.Spec.ToleRations,
			NodeName:                      instance.Spec.CoreTemplate.Spec.NodeName,
			NodeSelector:                  instance.Spec.CoreTemplate.Spec.NodeSelector,
			InitContainers:                instance.Spec.CoreTemplate.Spec.InitContainers,
			Volumes:                       instance.Spec.CoreTemplate.Spec.ExtraVolumes,
			Containers:                    containers,
			DNSPolicy:                     corev1.DNSClusterFirst,
			RestartPolicy:                 corev1.RestartPolicyAlways,
			SchedulerName:                 "default-scheduler",
			TerminationGracePeriodSeconds: pointer.Int64Ptr(30),
		},
	}

	annotations := instance.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")

	sts := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.Namespace,
			Name:        instance.Spec.CoreTemplate.Name,
			Labels:      instance.Spec.CoreTemplate.Labels,
			Annotations: annotations,
		},
		Spec: appsv1.StatefulSetSpec{
			RevisionHistoryLimit: pointer.Int32Ptr(10),
			ServiceName:          instance.NameOfHeadlessService(),
			Replicas:             instance.Spec.CoreTemplate.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: instance.Spec.CoreTemplate.Labels,
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template:            podTemplate,
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
					Partition: pointer.Int32Ptr(0),
				},
			},
		},
	}
	if !reflect.ValueOf(instance.Spec.CoreTemplate.Spec.VolumeClaimTemplates).IsZero() {
		sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instance.NameOfCoreNodeData(),
					Namespace: instance.Namespace,
					Labels:    instance.Spec.CoreTemplate.Labels,
				},
				Spec: instance.Spec.CoreTemplate.Spec.VolumeClaimTemplates,
			},
		}
	} else {
		sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: instance.NameOfCoreNodeData(),
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}

	return sts
}

func generateDeployment(instance *appsv2alpha1.EMQX) *appsv1.Deployment {
	annotations := instance.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")

	deploy := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.Namespace,
			Name:        instance.Spec.ReplicantTemplate.Name,
			Labels:      instance.Spec.ReplicantTemplate.Labels,
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			ProgressDeadlineSeconds: pointer.Int32Ptr(600),
			RevisionHistoryLimit:    pointer.Int32Ptr(10),
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{Type: intstr.String, StrVal: "25%"},
					MaxSurge:       &intstr.IntOrString{Type: intstr.String, StrVal: "25%"},
				},
			},
			Replicas: instance.Spec.ReplicantTemplate.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: instance.Spec.ReplicantTemplate.Labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      instance.Spec.ReplicantTemplate.Labels,
					Annotations: instance.Spec.ReplicantTemplate.Annotations,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: instance.Spec.ImagePullSecrets,
					SecurityContext:  instance.Spec.ReplicantTemplate.Spec.PodSecurityContext,
					Affinity:         instance.Spec.ReplicantTemplate.Spec.Affinity,
					Tolerations:      instance.Spec.ReplicantTemplate.Spec.ToleRations,
					NodeName:         instance.Spec.ReplicantTemplate.Spec.NodeName,
					NodeSelector:     instance.Spec.ReplicantTemplate.Spec.NodeSelector,
					InitContainers:   instance.Spec.ReplicantTemplate.Spec.InitContainers,
					Containers: append([]corev1.Container{
						{
							Name:            EMQXContainerName,
							Image:           instance.Spec.Image,
							ImagePullPolicy: instance.Spec.ImagePullPolicy,
							Command:         instance.Spec.ReplicantTemplate.Spec.Command,
							Args:            instance.Spec.ReplicantTemplate.Spec.Args,
							Ports:           instance.Spec.ReplicantTemplate.Spec.Ports,
							Env: append([]corev1.EnvVar{
								{
									Name:  "EMQX_NODE__DB_ROLE",
									Value: "replicant",
								},
								{
									Name: "EMQX_HOST",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											APIVersion: "v1",
											FieldPath:  "status.podIP",
										},
									},
								},
							}, instance.Spec.ReplicantTemplate.Spec.Env...),
							EnvFrom:         instance.Spec.ReplicantTemplate.Spec.EnvFrom,
							Resources:       instance.Spec.ReplicantTemplate.Spec.Resources,
							SecurityContext: instance.Spec.ReplicantTemplate.Spec.ContainerSecurityContext,
							LivenessProbe:   instance.Spec.ReplicantTemplate.Spec.LivenessProbe,
							ReadinessProbe:  instance.Spec.ReplicantTemplate.Spec.ReadinessProbe,
							StartupProbe:    instance.Spec.ReplicantTemplate.Spec.StartupProbe,
							Lifecycle:       instance.Spec.ReplicantTemplate.Spec.Lifecycle,
							VolumeMounts: append(instance.Spec.ReplicantTemplate.Spec.ExtraVolumeMounts, corev1.VolumeMount{
								Name:      instance.NameOfReplicantNodeData(),
								MountPath: "/opt/emqx/data",
							}),
							TerminationMessagePath:   "/dev/termination-log",
							TerminationMessagePolicy: corev1.TerminationMessageReadFile,
						},
					}, instance.Spec.ReplicantTemplate.Spec.ExtraContainers...),
					Volumes: append(instance.Spec.ReplicantTemplate.Spec.ExtraVolumes, corev1.Volume{
						Name: instance.NameOfReplicantNodeData(),
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					}),
					DNSPolicy:                     corev1.DNSClusterFirst,
					RestartPolicy:                 corev1.RestartPolicyAlways,
					SchedulerName:                 "default-scheduler",
					TerminationGracePeriodSeconds: pointer.Int64Ptr(30),
				},
			},
		},
	}
	return deploy
}

func updateStatefulSetForNodeCookie(sts *appsv1.StatefulSet, nodeCookie *corev1.Secret) *appsv1.StatefulSet {
	env := corev1.EnvVar{
		Name: "EMQX_NODE__COOKIE",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: nodeCookie.Name,
				},
				Key: "node_cookie",
			},
		},
	}

	if isNotExistEnv(sts.Spec.Template.Spec.Containers[0].Env, env) {
		sts.Spec.Template.Spec.Containers[0].Env = append(sts.Spec.Template.Spec.Containers[0].Env, env)
	}

	return sts
}

func updateStatefulSetForBootstrapUser(sts *appsv1.StatefulSet, bootstrapUser *corev1.Secret) *appsv1.StatefulSet {
	volume := corev1.Volume{
		Name: "bootstrap-user",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				DefaultMode: pointer.Int32(420),
				SecretName:  bootstrapUser.Name,
			},
		},
	}

	if isNotExistVolume(sts.Spec.Template.Spec.Volumes, volume) {
		sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, volume)
	}

	volumeMount := corev1.VolumeMount{
		Name:      "bootstrap-user",
		MountPath: "/opt/emqx/data/bootstrap_user",
		SubPath:   "bootstrap_user",
		ReadOnly:  true,
	}
	if isNotExistVolumeMount(sts.Spec.Template.Spec.Containers[0].VolumeMounts, volumeMount) {
		sts.Spec.Template.Spec.Containers[0].VolumeMounts = append(sts.Spec.Template.Spec.Containers[0].VolumeMounts, volumeMount)
	}

	env := corev1.EnvVar{
		Name:  "EMQX_DASHBOARD__BOOTSTRAP_USERS_FILE",
		Value: `"/opt/emqx/data/bootstrap_user"`,
	}

	if isNotExistEnv(sts.Spec.Template.Spec.Containers[0].Env, env) {
		sts.Spec.Template.Spec.Containers[0].Env = append(sts.Spec.Template.Spec.Containers[0].Env, env)
	}

	return sts
}

func updateStatefulSetForBootstrapConfig(sts *appsv1.StatefulSet, bootstrapConfig *corev1.ConfigMap) *appsv1.StatefulSet {
	volume := corev1.Volume{
		Name: "bootstrap-config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				DefaultMode: pointer.Int32(420),
				LocalObjectReference: corev1.LocalObjectReference{
					Name: bootstrapConfig.Name,
				},
			},
		},
	}
	if isNotExistVolume(sts.Spec.Template.Spec.Volumes, volume) {
		sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, volume)
	}

	volumeMount := corev1.VolumeMount{
		Name:      "bootstrap-config",
		MountPath: "/opt/emqx/etc/emqx.conf",
		SubPath:   "emqx.conf",
		ReadOnly:  true,
	}
	if isNotExistVolumeMount(sts.Spec.Template.Spec.Containers[0].VolumeMounts, volumeMount) {
		sts.Spec.Template.Spec.Containers[0].VolumeMounts = append(sts.Spec.Template.Spec.Containers[0].VolumeMounts, volumeMount)
	}

	return sts
}

func updateDeploymentForNodeCookie(deploy *appsv1.Deployment, nodeCookie *corev1.Secret) *appsv1.Deployment {
	env := corev1.EnvVar{
		Name: "EMQX_NODE__COOKIE",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: nodeCookie.Name,
				},
				Key: "node_cookie",
			},
		},
	}

	if isNotExistEnv(deploy.Spec.Template.Spec.Containers[0].Env, env) {
		deploy.Spec.Template.Spec.Containers[0].Env = append(deploy.Spec.Template.Spec.Containers[0].Env, env)
	}

	return deploy
}

func updateDeploymentForBootstrapConfig(deploy *appsv1.Deployment, bootstrapConfig *corev1.ConfigMap) *appsv1.Deployment {
	volume := corev1.Volume{
		Name: "bootstrap-config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				DefaultMode: pointer.Int32(420),
				LocalObjectReference: corev1.LocalObjectReference{
					Name: bootstrapConfig.Name,
				},
			},
		},
	}
	if isNotExistVolume(deploy.Spec.Template.Spec.Volumes, volume) {
		deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, volume)
	}

	volumeMount := corev1.VolumeMount{
		Name:      "bootstrap-config",
		MountPath: "/opt/emqx/etc/emqx.conf",
		SubPath:   "emqx.conf",
		ReadOnly:  true,
	}

	if isNotExistVolumeMount(deploy.Spec.Template.Spec.Containers[0].VolumeMounts, volumeMount) {
		deploy.Spec.Template.Spec.Containers[0].VolumeMounts = append(deploy.Spec.Template.Spec.Containers[0].VolumeMounts, volumeMount)
	}

	return deploy
}

func generateAnnotationByContainers(containers []corev1.Container) string {
	containerNames := []string{}
	for _, c := range containers {
		containerNames = append(containerNames, c.Name)
	}
	return strings.Join(containerNames, ",")
}

func isNotExistEnv(envs []corev1.EnvVar, env corev1.EnvVar) bool {
	for _, e := range envs {
		if e.Name == env.Name {
			return false
		}
	}
	return true
}

func isNotExistVolumeMount(volumeMounts []corev1.VolumeMount, volumeMount corev1.VolumeMount) bool {
	for _, v := range volumeMounts {
		if v.Name == volumeMount.Name {
			return false
		}
	}
	return true
}

func isNotExistVolume(volumes []corev1.Volume, volume corev1.Volume) bool {
	for _, v := range volumes {
		if v.Name == volume.Name {
			return false
		}
	}
	return true
}
