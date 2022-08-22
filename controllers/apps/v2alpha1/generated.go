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

package apps

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	"github.com/emqx/emqx-operator/pkg/handler"
	"github.com/sethvargo/go-password/password"
	appsv1 "k8s.io/api/apps/v1"
)

func generateBootstrapUserSecret(instance *appsv2alpha1.EMQX) *corev1.Secret {
	username := "emqx_operator_controller"
	password, _ := password.Generate(64, 10, 10, false, false)

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-bootstrap-user", instance.Name),
			Namespace:   instance.Namespace,
			Labels:      instance.Labels,
			Annotations: instance.Annotations,
		},
		StringData: map[string]string{
			"bootstrap_user": fmt.Sprintf("%s:%s", username, password),
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
			Name:        fmt.Sprintf("%s-bootstrap-config", instance.Name),
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
			Name:      fmt.Sprintf("%s-headless", instance.Name),
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
	instance.Spec.DashboardServiceTemplate.Spec.Selector = instance.Spec.CoreTemplate.Labels

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-dashboard", instance.Name),
			Namespace:   instance.Namespace,
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

	if isExistReplicant(instance) {
		instance.Spec.ListenersServiceTemplate.Spec.Selector = instance.Spec.ReplicantTemplate.Labels
	} else {
		instance.Spec.ListenersServiceTemplate.Spec.Selector = instance.Spec.CoreTemplate.Labels
	}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-listeners", instance.Name),
			Namespace:   instance.Namespace,
			Labels:      instance.Spec.ListenersServiceTemplate.Labels,
			Annotations: instance.Spec.ListenersServiceTemplate.Annotations,
		},
		Spec: instance.Spec.ListenersServiceTemplate.Spec,
	}
}

func generateStatefulSet(instance *appsv2alpha1.EMQX) *appsv1.StatefulSet {
	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      instance.Spec.CoreTemplate.Labels,
			Annotations: instance.Spec.CoreTemplate.Annotations,
		},
		Spec: corev1.PodSpec{
			ImagePullSecrets: instance.Spec.ImagePullSecrets,
			SecurityContext:  instance.Spec.SecurityContext,
			Affinity:         instance.Spec.CoreTemplate.Spec.Affinity,
			Tolerations:      instance.Spec.CoreTemplate.Spec.ToleRations,
			NodeName:         instance.Spec.CoreTemplate.Spec.NodeName,
			NodeSelector:     instance.Spec.CoreTemplate.Spec.NodeSelector,
			InitContainers:   instance.Spec.CoreTemplate.Spec.InitContainers,
			Containers: append([]corev1.Container{
				{
					Name:            EMQXContainerName,
					Image:           instance.Spec.Image,
					ImagePullPolicy: corev1.PullPolicy(instance.Spec.ImagePullPolicy),
					Env: []corev1.EnvVar{
						{
							Name:  "EMQX_NODE__DB_ROLE",
							Value: "core",
						},
						{
							Name:  "EMQX_CLUSTER__DISCOVERY_STRATEGY",
							Value: "dns",
						},
						{
							Name:  "EMQX_CLUSTER__DNS__NAME",
							Value: fmt.Sprintf("%s-headless.%s.svc.cluster.local", instance.Name, instance.Namespace),
						},
						{
							Name:  "EMQX_CLUSTER__DNS__RECORD_TYPE",
							Value: "srv",
						},
					},
					Args:            instance.Spec.CoreTemplate.Spec.Args,
					Resources:       instance.Spec.CoreTemplate.Spec.Resources,
					ReadinessProbe:  instance.Spec.CoreTemplate.Spec.ReadinessProbe,
					LivenessProbe:   instance.Spec.CoreTemplate.Spec.LivenessProbe,
					StartupProbe:    instance.Spec.CoreTemplate.Spec.StartupProbe,
					SecurityContext: instance.Spec.CoreTemplate.Spec.SecurityContext,
					VolumeMounts: append(instance.Spec.CoreTemplate.Spec.ExtraVolumeMounts, corev1.VolumeMount{
						Name:      fmt.Sprintf("%s-core-data", instance.Name),
						MountPath: "/opt/emqx/data",
					}),
				},
			}, instance.Spec.CoreTemplate.Spec.ExtraContainers...),
			Volumes: instance.Spec.CoreTemplate.Spec.ExtraVolumes,
		},
	}
	podAnnotation := podTemplate.ObjectMeta.DeepCopy().Annotations
	if podAnnotation == nil {
		podAnnotation = make(map[string]string)
	}
	podAnnotation[handler.ManageContainersAnnotation] = generateAnnotationByContainers(podTemplate.Spec.Containers)
	podTemplate.Annotations = podAnnotation

	annotations := instance.GetAnnotations()
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
			Name:        fmt.Sprintf("%s-core", instance.Name),
			Namespace:   instance.GetNamespace(),
			Labels:      instance.Spec.CoreTemplate.Labels,
			Annotations: annotations,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: fmt.Sprintf("%s-headless", instance.Name),
			Replicas:    instance.Spec.CoreTemplate.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: instance.Spec.CoreTemplate.Labels,
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template:            podTemplate,
		},
	}
	if !reflect.ValueOf(instance.Spec.CoreTemplate.Spec.Persistent).IsZero() {
		sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-core-data", instance.Name),
					Namespace: instance.GetNamespace(),
					Labels:    instance.Spec.CoreTemplate.Labels,
				},
				Spec: instance.Spec.CoreTemplate.Spec.Persistent,
			},
		}
	} else {
		sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: fmt.Sprintf("%s-core-data", instance.Name),
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}

	return sts
}

func generateDeployment(instance *appsv2alpha1.EMQX) *appsv1.Deployment {
	coreNodesSuffix := fmt.Sprintf("%s.%s.svc.cluster.local", fmt.Sprintf("%s-headless", instance.Name), instance.Namespace)

	coreNodes := []string{}
	for i := int32(0); i < *instance.Spec.CoreTemplate.Spec.Replicas; i++ {
		coreNodes = append(coreNodes,
			fmt.Sprintf(
				"%s@%s-%d.%s",
				EMQXContainerName, fmt.Sprintf("%s-core", instance.Name), i, coreNodesSuffix,
			),
		)
	}
	coreNodesStr, _ := json.Marshal(coreNodes)

	annotations := instance.GetAnnotations()
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
			Name:        fmt.Sprintf("%s-replicant", instance.Name),
			Namespace:   instance.GetNamespace(),
			Labels:      instance.Spec.ReplicantTemplate.Labels,
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
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
					SecurityContext:  instance.Spec.SecurityContext,
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
							Env: []corev1.EnvVar{
								{
									Name:  "EMQX_NODE__DB_ROLE",
									Value: "replicant",
								},
								{
									Name: "EMQX_HOST",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "EMQX_CLUSTER__DISCOVERY_STRATEGY",
									Value: "static",
								},
								{
									Name:  "EMQX_CLUSTER__STATIC__SEEDS",
									Value: string(coreNodesStr),
								},
							},
							Args:            instance.Spec.ReplicantTemplate.Spec.Args,
							Resources:       instance.Spec.ReplicantTemplate.Spec.Resources,
							ReadinessProbe:  instance.Spec.ReplicantTemplate.Spec.ReadinessProbe,
							LivenessProbe:   instance.Spec.ReplicantTemplate.Spec.LivenessProbe,
							StartupProbe:    instance.Spec.ReplicantTemplate.Spec.StartupProbe,
							SecurityContext: instance.Spec.ReplicantTemplate.Spec.SecurityContext,
							VolumeMounts: append(instance.Spec.ReplicantTemplate.Spec.ExtraVolumeMounts, corev1.VolumeMount{
								Name:      fmt.Sprintf("%s-replicant-data", instance.Name),
								MountPath: "/opt/emqx/data",
							}),
						},
					}, instance.Spec.ReplicantTemplate.Spec.ExtraContainers...),
					Volumes: append(instance.Spec.ReplicantTemplate.Spec.ExtraVolumes, corev1.Volume{
						Name: fmt.Sprintf("%s-replicant-data", instance.Name),
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					}),
				},
			},
		},
	}
	return deploy
}

func updateStatefulSetForBootstrapUser(sts *appsv1.StatefulSet, bootstrapUser *corev1.Secret) *appsv1.StatefulSet {
	isNotExistEnv := func() bool {
		for _, v := range sts.Spec.Template.Spec.Containers[0].Env {
			if v.Name == "EMQX_DASHBOARD__BOOTSTRAP_USERS_FILE" {
				return false
			}
		}
		return true
	}

	if isNotExistEnv() {
		sts.Spec.Template.Spec.Containers[0].Env = append(sts.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
			Name:  "EMQX_DASHBOARD__BOOTSTRAP_USERS_FILE",
			Value: "/opt/emqx/data/bootstrap_user",
		})
	}

	volume := corev1.Volume{
		Name: "bootstrap-user",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: bootstrapUser.Name,
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

	return sts
}

func updateStatefulSetForBootstrapConfig(sts *appsv1.StatefulSet, bootstrapConfig *corev1.ConfigMap) *appsv1.StatefulSet {
	volume := corev1.Volume{
		Name: "bootstrap-config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
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
func updateDeploymentForBootstrapConfig(deploy *appsv1.Deployment, bootstrapConfig *corev1.ConfigMap) *appsv1.Deployment {
	volume := corev1.Volume{
		Name: "bootstrap-config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
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
