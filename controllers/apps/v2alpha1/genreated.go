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
	appsv1 "k8s.io/api/apps/v1"
)

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
	instance.Spec.DashboardServiceTemplate.Spec.Ports = mergeServicePorts(
		instance.Spec.DashboardServiceTemplate.Spec.Ports,
		[]corev1.ServicePort{
			{
				Name:       "dashboard",
				Protocol:   corev1.ProtocolTCP,
				Port:       18083,
				TargetPort: intstr.FromInt(18083),
			},
		},
	)

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
	instance.Spec.ListenersServiceTemplate.Spec.Ports = mergeServicePorts(
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
						{
							Name:  "EMQX_CLUSTER__K8S__ADDRESS_TYPE",
							Value: "hostname",
						},
						{
							Name:  "EMQX_CLUSTER__K8S__NAMESPACE",
							Value: instance.Namespace,
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

func mergeServicePorts(ports1, ports2 []corev1.ServicePort) []corev1.ServicePort {
	ports := append(ports1, ports2...)

	result := make([]corev1.ServicePort, 0, len(ports))
	temp := map[string]struct{}{}

	for _, item := range ports {
		if _, ok := temp[item.Name]; !ok {
			temp[item.Name] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}

func generateAnnotationByContainers(containers []corev1.Container) string {
	containerNames := []string{}
	for _, c := range containers {
		containerNames = append(containerNames, c.Name)
	}
	return strings.Join(containerNames, ",")
}
