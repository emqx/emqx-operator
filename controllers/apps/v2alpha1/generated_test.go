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
	"strings"
	"testing"

	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

var (
	coreLabels = map[string]string{
		"apps.emqx.io/instance":   "emqx",
		"apps.emqx.io/managed-by": "emqx-operator",
		"apps.emqx.io/db-role":    "core",
	}
	replicantLabels = map[string]string{
		"apps.emqx.io/instance":   "emqx",
		"apps.emqx.io/managed-by": "emqx-operator",
		"apps.emqx.io/db-role":    "replicant",
	}
)

func TestGenerateNodeCookieSecret(t *testing.T) {
	instance := &appsv2alpha1.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
	}

	t.Run("generate node cookie secret", func(t *testing.T) {
		got := generateNodeCookieSecret(instance)
		assert.Equal(t, "emqx-node-cookie", got.Name)
		_, ok := got.StringData["node_cookie"]
		assert.True(t, ok)
	})

	t.Run("generate node cookie when already set node cookie", func(t *testing.T) {
		instance.Spec.BootstrapConfig = "node.cookie = fake"
		got := generateNodeCookieSecret(instance)
		assert.Equal(t, "emqx-node-cookie", got.Name)
		_, ok := got.StringData["node_cookie"]
		assert.True(t, ok)
		assert.Equal(t, "fake", got.StringData["node_cookie"])
	})
}

func TestGenerateBootstrapUserSecret(t *testing.T) {
	instance := &appsv2alpha1.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: appsv2alpha1.EMQXSpec{
			BootstrapAPIKeys: []appsv2alpha1.BootstrapAPIKey{
				{
					Key:    "test_key",
					Secret: "secret",
				},
			},
		},
	}

	got := generateBootstrapUserSecret(instance)
	assert.Equal(t, "emqx-bootstrap-user", got.Name)
	data, ok := got.StringData["bootstrap_user"]
	assert.True(t, ok)

	users := strings.Split(data, "\n")
	var usernames []string
	for _, user := range users {
		usernames = append(usernames, user[:strings.Index(user, ":")])
	}
	assert.ElementsMatch(t, usernames, []string{defUsername, "test_key"})
}

func TestGenerateBootstrapConfigMap(t *testing.T) {
	instance := &appsv2alpha1.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
	}

	got := generateBootstrapConfigMap(instance)
	assert.Equal(t, "emqx-bootstrap-config", got.Name)
	_, ok := got.Data["emqx.conf"]
	assert.True(t, ok)
}

func TestGenerateHeadlessSVC(t *testing.T) {
	instance := &appsv2alpha1.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: appsv2alpha1.EMQXSpec{
			CoreTemplate: appsv2alpha1.EMQXCoreTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Labels: coreLabels,
				},
			},
		},
	}
	expect := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx-headless",
			Namespace: "emqx",
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
			Selector: coreLabels,
		},
	}
	assert.Equal(t, expect, generateHeadlessService(instance))
}

func TestGenerateDashboardService(t *testing.T) {
	instance := &appsv2alpha1.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: appsv2alpha1.EMQXSpec{
			CoreTemplate: appsv2alpha1.EMQXCoreTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Labels: coreLabels,
				},
			},
			DashboardServiceTemplate: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "emqx-dashboard",
					Labels: map[string]string{
						"apps.emqx.io/instance": "emqx",
					},
					Annotations: map[string]string{
						"foo": "bar",
					},
				},
				Spec: corev1.ServiceSpec{
					Selector: coreLabels,
					Ports: []corev1.ServicePort{
						{
							Name:       "dashboard",
							Protocol:   corev1.ProtocolTCP,
							Port:       18083,
							TargetPort: intstr.FromInt(18083),
						},
					},
				},
			},
		},
	}

	expect := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx-dashboard",
			Namespace: "emqx",
			Labels: map[string]string{
				"apps.emqx.io/instance": "emqx",
			},
			Annotations: map[string]string{
				"foo": "bar",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: coreLabels,
			Ports: []corev1.ServicePort{
				{
					Name:       "dashboard",
					Protocol:   corev1.ProtocolTCP,
					Port:       18083,
					TargetPort: intstr.FromInt(18083),
				},
			},
		},
	}

	assert.Equal(t, expect, generateDashboardService(instance))
}

func TestGenerateListenerService(t *testing.T) {
	var replicas int32 = 3

	listenerPorts := []corev1.ServicePort{
		{
			Name:       "mqtt",
			Protocol:   corev1.ProtocolTCP,
			Port:       1883,
			TargetPort: intstr.FromInt(1883),
		},
	}

	instance := &appsv2alpha1.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: appsv2alpha1.EMQXSpec{
			ReplicantTemplate: appsv2alpha1.EMQXReplicantTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Labels: replicantLabels,
				},
			},
			ListenersServiceTemplate: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "emqx-listeners",
					Labels: map[string]string{
						"apps.emqx.io/instance": "emqx",
					},
					Annotations: map[string]string{
						"foo": "bar",
					},
				},
				Spec: corev1.ServiceSpec{},
			},
		},
	}

	assert.Nil(t, generateListenerService(instance, []corev1.ServicePort{}))

	expect := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx-listeners",
			Namespace: "emqx",
			Labels: map[string]string{
				"apps.emqx.io/instance": "emqx",
			},
			Annotations: map[string]string{
				"foo": "bar",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: replicantLabels,
			Ports:    listenerPorts,
		},
	}

	assert.Equal(t, expect, generateListenerService(instance, listenerPorts))

	instance.Spec.ReplicantTemplate = appsv2alpha1.EMQXReplicantTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Labels: replicantLabels,
		},
		Spec: appsv2alpha1.EMQXReplicantTemplateSpec{
			Replicas: &replicas,
		},
	}
	expect.Spec.Selector = replicantLabels

	assert.Equal(t, expect, generateListenerService(instance, listenerPorts))
}

func TestGenerateStatefulSet(t *testing.T) {
	var replicas int32 = 3
	var user, group int64 = 1001, 1001
	var storageClass string = "emqx-storage"

	instance := &appsv2alpha1.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
			Annotations: map[string]string{
				"apps.emqx.io/managed-by":                          "emqx-operator",
				"apps.emqx.io/instance":                            "emqx",
				"kubectl.kubernetes.io/last-applied-configuration": "fake",
			},
		},
		Spec: appsv2alpha1.EMQXSpec{
			Image:           "emqx/emqx:5.0",
			ImagePullPolicy: corev1.PullIfNotPresent,
			ImagePullSecrets: []corev1.LocalObjectReference{
				{
					Name: "fake-secret",
				},
			},
			CoreTemplate: appsv2alpha1.EMQXCoreTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "emqx-core",
					Labels: coreLabels,
					Annotations: map[string]string{
						"foo": "bar",
					},
				},
				Spec: appsv2alpha1.EMQXCoreTemplateSpec{
					EMQXReplicantTemplateSpec: appsv2alpha1.EMQXReplicantTemplateSpec{
						Command: []string{"/bin/sh", "-c"},
						Args:    []string{"hello world"},
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: int32(1883),
							},
						},
						Env: []corev1.EnvVar{
							{
								Name:  "FOO",
								Value: "BAR",
							},
						},
						EnvFrom: []corev1.EnvFromSource{
							{
								ConfigMapRef: &corev1.ConfigMapEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "fake-config",
									},
								},
							},
						},
						PodSecurityContext: &corev1.PodSecurityContext{
							RunAsUser:  &user,
							RunAsGroup: &group,
							FSGroup:    &group,
						},
						ContainerSecurityContext: &corev1.SecurityContext{
							RunAsUser:  &user,
							RunAsGroup: &group,
						},
						Replicas:    &replicas,
						Affinity:    &corev1.Affinity{},
						ToleRations: []corev1.Toleration{},
						NodeName:    "emqx-node",
						NodeSelector: map[string]string{
							"kubernetes.io/hostname": "emqx-node",
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("100m"),
								corev1.ResourceMemory: resource.MustParse("100Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("100m"),
								corev1.ResourceMemory: resource.MustParse("100Mi"),
							},
						},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/status",
									Port: intstr.FromInt(18083),
								},
							},
							InitialDelaySeconds: int32(10),
							PeriodSeconds:       int32(5),
							FailureThreshold:    int32(30),
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/status",
									Port: intstr.FromInt(18083),
								},
							},
							InitialDelaySeconds: int32(10),
							PeriodSeconds:       int32(5),
							FailureThreshold:    int32(30),
						},
						StartupProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/status",
									Port: intstr.FromInt(18083),
								},
							},
							InitialDelaySeconds: int32(10),
							PeriodSeconds:       int32(5),
							FailureThreshold:    int32(30),
						},
						Lifecycle: &corev1.Lifecycle{
							PreStop: &corev1.LifecycleHandler{
								Exec: &corev1.ExecAction{
									Command: []string{"emqx", "ctl", "cluster", "leave"},
								},
							},
						},
						InitContainers: []corev1.Container{
							{
								Name:  "init",
								Image: "hello-world",
							},
						},
						ExtraContainers: []corev1.Container{
							{
								Name:  "extra",
								Image: "busybox",
							},
						},
					},
				},
			},
		},
	}
	instance.Default()

	expect := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx-core",
			Namespace: "emqx",
			Labels:    coreLabels,
			Annotations: map[string]string{
				"apps.emqx.io/managed-by": "emqx-operator",
				"apps.emqx.io/instance":   "emqx",
			},
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: "emqx-headless",
			Replicas:    &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: coreLabels,
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
					Partition: pointer.Int32(0),
				},
			},
			RevisionHistoryLimit: pointer.Int32(10),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: coreLabels,
					Annotations: map[string]string{
						"foo":                                "bar",
						"apps.emqx.io/headless-service-name": "emqx-headless",
						"apps.emqx.io/manage-containers":     "emqx,extra",
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy:                 corev1.RestartPolicyAlways,
					TerminationGracePeriodSeconds: pointer.Int64(30),
					DNSPolicy:                     corev1.DNSClusterFirst,
					SchedulerName:                 "default-scheduler",
					ImagePullSecrets: []corev1.LocalObjectReference{
						{Name: "fake-secret"},
					},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  &user,
						RunAsGroup: &group,
						FSGroup:    &group,
					},
					Affinity:    &corev1.Affinity{},
					Tolerations: []corev1.Toleration{},
					NodeName:    "emqx-node",
					NodeSelector: map[string]string{
						"kubernetes.io/hostname": "emqx-node",
					},
					InitContainers: []corev1.Container{
						{
							Name:  "init",
							Image: "hello-world",
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "emqx",
							Image:           "emqx/emqx:5.0",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"/bin/sh", "-c"},
							Args:            []string{"hello world"},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: int32(1883),
								},
							},
							Env: []corev1.EnvVar{
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
								{
									Name:  "FOO",
									Value: "BAR",
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "fake-config",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("100Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("100Mi"),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:  &user,
								RunAsGroup: &group,
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/status",
										Port: intstr.FromInt(18083),
									},
								},
								InitialDelaySeconds: int32(10),
								PeriodSeconds:       int32(5),
								FailureThreshold:    int32(30),
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/status",
										Port: intstr.FromInt(18083),
									},
								},
								InitialDelaySeconds: int32(10),
								PeriodSeconds:       int32(5),
								FailureThreshold:    int32(30),
							},
							StartupProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/status",
										Port: intstr.FromInt(18083),
									},
								},
								InitialDelaySeconds: int32(10),
								PeriodSeconds:       int32(5),
								FailureThreshold:    int32(30),
							},
							Lifecycle: &corev1.Lifecycle{
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"emqx", "ctl", "cluster", "leave"},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "emqx-core-data",
									MountPath: "/opt/emqx/data",
								},
							},
							TerminationMessagePath:   "/dev/termination-log",
							TerminationMessagePolicy: corev1.TerminationMessageReadFile,
						},
						{
							Name:  "extra",
							Image: "busybox",
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "emqx-core-data",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expect, generateStatefulSet(instance))

	instance.Spec.CoreTemplate.Spec.VolumeClaimTemplates = corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{
			corev1.ReadWriteOnce,
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("20Mi"),
			},
		},
		StorageClassName: &storageClass,
	}

	expect.Spec.Template.Spec.Volumes = nil
	expect.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "emqx-core-data",
				Namespace: "emqx",
				Labels:    coreLabels,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				StorageClassName: &storageClass,
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("20Mi"),
					},
				},
			},
		},
	}

	assert.Equal(t, expect, generateStatefulSet(instance))
}

func TestGenerateDeployment(t *testing.T) {
	var replicas int32 = 3
	var user, group int64 = 1001, 1001

	instance := &appsv2alpha1.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
			Annotations: map[string]string{
				"apps.emqx.io/managed-by":                          "emqx-operator",
				"apps.emqx.io/instance":                            "emqx",
				"kubectl.kubernetes.io/last-applied-configuration": "fake",
			},
		},
		Spec: appsv2alpha1.EMQXSpec{
			Image:           "emqx/emqx:5.0",
			ImagePullPolicy: corev1.PullIfNotPresent,
			ImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "fake-secret"},
			},
			CoreTemplate: appsv2alpha1.EMQXCoreTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "emqx-core",
				},
				Spec: appsv2alpha1.EMQXCoreTemplateSpec{
					EMQXReplicantTemplateSpec: appsv2alpha1.EMQXReplicantTemplateSpec{
						Replicas: &replicas,
					},
				},
			},
			ReplicantTemplate: appsv2alpha1.EMQXReplicantTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "emqx-replicant",
					Labels: replicantLabels,
					Annotations: map[string]string{
						"foo": "bar",
					},
				},
				Spec: appsv2alpha1.EMQXReplicantTemplateSpec{
					Replicas: &replicas,
					NodeName: "emqx-node",
					NodeSelector: map[string]string{
						"kubernetes.io/hostname": "emqx-node",
					},
					InitContainers: []corev1.Container{
						{Name: "init", Image: "busybox"},
					},
					Command: []string{"/bin/sh", "-c"},
					Args:    []string{"hello world"},
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: int32(1883),
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  "FOO",
							Value: "BAR",
						},
					},
					EnvFrom: []corev1.EnvFromSource{
						{
							ConfigMapRef: &corev1.ConfigMapEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "fake-config",
								},
							},
						},
					},
					PodSecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  &user,
						RunAsGroup: &group,
						FSGroup:    &group,
					},
					ContainerSecurityContext: &corev1.SecurityContext{
						RunAsUser:  &user,
						RunAsGroup: &group,
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("100Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("100Mi"),
						},
					},
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/status",
								Port: intstr.FromInt(18083),
							},
						},
						InitialDelaySeconds: int32(10),
						PeriodSeconds:       int32(5),
						FailureThreshold:    int32(30),
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/status",
								Port: intstr.FromInt(18083),
							},
						},
						InitialDelaySeconds: int32(10),
						PeriodSeconds:       int32(5),
						FailureThreshold:    int32(30),
					},
					StartupProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/status",
								Port: intstr.FromInt(18083),
							},
						},
						InitialDelaySeconds: int32(10),
						PeriodSeconds:       int32(5),
						FailureThreshold:    int32(30),
					},
					Lifecycle: &corev1.Lifecycle{
						PreStop: &corev1.LifecycleHandler{
							Exec: &corev1.ExecAction{
								Command: []string{"emqx", "ctl", "cluster", "leave"},
							},
						},
					},
					ExtraContainers: []corev1.Container{
						{Name: "extra", Image: "busybox"},
					},
					ExtraVolumeMounts: []corev1.VolumeMount{
						{Name: "extra", MountPath: "/extra"},
					},
					ExtraVolumes: []corev1.Volume{
						{Name: "extra", VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								DefaultMode:          pointer.Int32Ptr(420),
								LocalObjectReference: corev1.LocalObjectReference{Name: "extra"},
							},
						}},
					},
				},
			},
		},
	}

	expect := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx-replicant",
			Namespace: "emqx",
			Labels:    replicantLabels,
			Annotations: map[string]string{
				"apps.emqx.io/managed-by": "emqx-operator",
				"apps.emqx.io/instance":   "emqx",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: replicantLabels,
			},
			RevisionHistoryLimit: pointer.Int32Ptr(10),
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{Type: intstr.String, StrVal: "25%"},
					MaxSurge:       &intstr.IntOrString{Type: intstr.String, StrVal: "25%"},
				},
			},
			ProgressDeadlineSeconds: pointer.Int32Ptr(600),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: replicantLabels,
					Annotations: map[string]string{
						"foo": "bar",
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy:                 corev1.RestartPolicyAlways,
					TerminationGracePeriodSeconds: pointer.Int64Ptr(30),
					DNSPolicy:                     corev1.DNSClusterFirst,
					SchedulerName:                 "default-scheduler",
					ImagePullSecrets: []corev1.LocalObjectReference{
						{Name: "fake-secret"},
					},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  &user,
						RunAsGroup: &group,
						FSGroup:    &group,
					},
					NodeName: "emqx-node",
					NodeSelector: map[string]string{
						"kubernetes.io/hostname": "emqx-node",
					},
					InitContainers: []corev1.Container{
						{Name: "init", Image: "busybox"},
					},
					Containers: []corev1.Container{
						{
							Name:            "emqx",
							Image:           "emqx/emqx:5.0",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"/bin/sh", "-c"},
							Args:            []string{"hello world"},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: int32(1883),
								},
							},
							Env: []corev1.EnvVar{
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
								{
									Name:  "FOO",
									Value: "BAR",
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "fake-config",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("100Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("100Mi"),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:  &user,
								RunAsGroup: &group,
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/status",
										Port: intstr.FromInt(18083),
									},
								},
								InitialDelaySeconds: int32(10),
								PeriodSeconds:       int32(5),
								FailureThreshold:    int32(30),
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/status",
										Port: intstr.FromInt(18083),
									},
								},
								InitialDelaySeconds: int32(10),
								PeriodSeconds:       int32(5),
								FailureThreshold:    int32(30),
							},
							StartupProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/status",
										Port: intstr.FromInt(18083),
									},
								},
								InitialDelaySeconds: int32(10),
								PeriodSeconds:       int32(5),
								FailureThreshold:    int32(30),
							},
							Lifecycle: &corev1.Lifecycle{
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"emqx", "ctl", "cluster", "leave"},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "extra",
									MountPath: "/extra",
								},
								{
									Name:      "emqx-replicant-data",
									MountPath: "/opt/emqx/data",
								},
							},
							TerminationMessagePath:   "/dev/termination-log",
							TerminationMessagePolicy: corev1.TerminationMessageReadFile,
						},
						{Name: "extra", Image: "busybox"},
					},
					Volumes: []corev1.Volume{
						{
							Name: "extra",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									DefaultMode:          pointer.Int32Ptr(420),
									LocalObjectReference: corev1.LocalObjectReference{Name: "extra"},
								},
							},
						},
						{
							Name: "emqx-replicant-data",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	assert.Equal(t, expect, generateDeployment(instance))
}

func TestUpdateStatefulSetForNodeCookie(t *testing.T) {
	nodeCookie := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "emqx-node-cookie",
		},
		StringData: map[string]string{
			"node_cookie": "fake",
		},
	}

	sts := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "emqx"},
					},
				},
			},
		},
	}

	got := updateStatefulSetForNodeCookie(sts, nodeCookie)

	assert.Equal(t, []corev1.EnvVar{{
		Name: "EMQX_NODE__COOKIE",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "emqx-node-cookie",
				},
				Key: "node_cookie",
			},
		},
	}}, got.Spec.Template.Spec.Containers[0].Env)
}

func TestUpdateStatefulSetForBootstrapUser(t *testing.T) {
	bootstrapUser := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "emqx-bootstrap-user",
		},
	}

	sts := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "emqx"},
					},
				},
			},
		},
	}

	got := updateStatefulSetForBootstrapUser(sts, bootstrapUser)

	assert.Equal(t, []corev1.VolumeMount{{
		Name:      "bootstrap-user",
		MountPath: "/opt/emqx/data/bootstrap_user",
		SubPath:   "bootstrap_user",
		ReadOnly:  true,
	}}, got.Spec.Template.Spec.Containers[0].VolumeMounts)

	assert.Equal(t, []corev1.Volume{{
		Name: "bootstrap-user",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				DefaultMode: pointer.Int32Ptr(420),
				SecretName:  "emqx-bootstrap-user",
			},
		},
	}}, got.Spec.Template.Spec.Volumes)

	assert.Equal(t, []corev1.EnvVar{{
		Name:  "EMQX_DASHBOARD__BOOTSTRAP_USERS_FILE",
		Value: `"/opt/emqx/data/bootstrap_user"`,
	}}, got.Spec.Template.Spec.Containers[0].Env)
}

func TestUpdateStatefulSetForBootstrapConfig(t *testing.T) {
	bootstrapConfig := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "emqx-bootstrap-config",
		},
	}

	sts := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "emqx"},
					},
				},
			},
		},
	}

	got := updateStatefulSetForBootstrapConfig(sts, bootstrapConfig)

	assert.Equal(t, []corev1.Volume{{
		Name: "bootstrap-config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				DefaultMode: pointer.Int32Ptr(420),
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "emqx-bootstrap-config",
				},
			},
		},
	}}, got.Spec.Template.Spec.Volumes)

	assert.Equal(t, []corev1.VolumeMount{{
		Name:      "bootstrap-config",
		MountPath: "/opt/emqx/etc/emqx.conf",
		SubPath:   "emqx.conf",
		ReadOnly:  true,
	}}, got.Spec.Template.Spec.Containers[0].VolumeMounts)
}

func TestUpdateDeploymentForNodeCookie(t *testing.T) {
	nodeCookie := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "emqx-node-cookie",
		},
		StringData: map[string]string{
			"node_cookie": "fake",
		},
	}

	deploy := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "emqx"},
					},
				},
			},
		},
	}

	got := updateDeploymentForNodeCookie(deploy, nodeCookie)

	assert.Equal(t, []corev1.EnvVar{{
		Name: "EMQX_NODE__COOKIE",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "emqx-node-cookie",
				},
				Key: "node_cookie",
			},
		},
	}}, got.Spec.Template.Spec.Containers[0].Env)
}

func TestUpdateDeploymentForBootstrapConfig(t *testing.T) {
	bootstrapConfig := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "emqx-bootstrap-config",
		},
	}

	deploy := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "emqx"},
					},
				},
			},
		},
	}

	got := updateDeploymentForBootstrapConfig(deploy, bootstrapConfig)

	assert.Equal(t, []corev1.Volume{{
		Name: "bootstrap-config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				DefaultMode: pointer.Int32Ptr(420),
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "emqx-bootstrap-config",
				},
			},
		},
	}}, got.Spec.Template.Spec.Volumes)

	assert.Equal(t, []corev1.VolumeMount{{
		Name:      "bootstrap-config",
		MountPath: "/opt/emqx/etc/emqx.conf",
		SubPath:   "emqx.conf",
		ReadOnly:  true,
	}}, got.Spec.Template.Spec.Containers[0].VolumeMounts)
}

func TestIsNotExistVolumeMount(t *testing.T) {
	volumeMounts := []corev1.VolumeMount{
		{Name: "exist"},
	}

	assert.True(t, isNotExistVolumeMount(volumeMounts, corev1.VolumeMount{Name: "not-exist"}))
	assert.False(t, isNotExistVolumeMount(volumeMounts, corev1.VolumeMount{Name: "exist"}))
}

func TestIsNotExistVolume(t *testing.T) {
	volumes := []corev1.Volume{
		{Name: "exist"},
	}

	assert.True(t, isNotExistVolume(volumes, corev1.Volume{Name: "not-exist"}))
	assert.False(t, isNotExistVolume(volumes, corev1.Volume{Name: "exist"}))
}
