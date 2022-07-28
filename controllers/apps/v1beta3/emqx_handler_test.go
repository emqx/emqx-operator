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
	"testing"

	appsv1beta3 "github.com/emqx/emqx-operator/apis/apps/v1beta3"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var replicas int32 = 3
var storageClassName string = "emqx-storage"
var user, group int64 = 1000, 1000
var fsGroupChangeAlways corev1.PodFSGroupChangePolicy = corev1.FSGroupChangeAlways

func TestGenerateStatefulSetDef(t *testing.T) {
	broker := &appsv1beta3.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "default",
			Annotations: map[string]string{
				"foo": "bar",
			},
		},
		Spec: appsv1beta3.EmqxBrokerSpec{
			Replicas: &replicas,
			ImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "fake-secret"},
			},
			Persistent: corev1.PersistentVolumeClaimSpec{
				StorageClassName: &storageClassName,
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
			},
			NodeName: "node",
			NodeSelector: map[string]string{
				"kubernetes.io/hostname": "node",
			},
			InitContainers: []corev1.Container{
				{Name: "init", Image: "init"},
			},
			ExtraContainers: []corev1.Container{
				{Name: "extra", Image: "extra"},
			},
			Env: []corev1.EnvVar{
				{Name: "EMQX_LOG__TO", Value: "file"},
			},
			EmqxTemplate: appsv1beta3.EmqxBrokerTemplate{
				Image:           "emqx/emqx:4.4.6",
				ImagePullPolicy: corev1.PullAlways,
				EmqxConfig: appsv1beta3.EmqxConfig{
					"log.level": "debug",
				},
				Args: []string{
					"--log.level", "debug",
				},
				ExtraVolumes: []corev1.Volume{
					{Name: "extra", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
				},
				ExtraVolumeMounts: []corev1.VolumeMount{
					{Name: "extra", MountPath: "/extra"},
				},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("100m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("100m"),
					},
				},
			},
		},
	}
	broker.Default()

	expect := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "default",
			Labels: map[string]string{
				"apps.emqx.io/managed-by": "emqx-operator",
				"apps.emqx.io/instance":   "emqx",
			},
			Annotations: map[string]string{
				"foo": "bar",
			},
		},
		Spec: appsv1.StatefulSetSpec{
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "emqx-data",
						Namespace: "default",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						StorageClassName: &storageClassName,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "emqx-log",
						Namespace: "default",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						StorageClassName: &storageClassName,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				},
			},
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"apps.emqx.io/managed-by": "emqx-operator",
					"apps.emqx.io/instance":   "emqx",
				},
			},
			PodManagementPolicy: appsv1.OrderedReadyPodManagement,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"apps.emqx.io/managed-by": "emqx-operator",
						"apps.emqx.io/instance":   "emqx",
					},
					Annotations: map[string]string{
						"foo": "bar",
					},
				},
				Spec: corev1.PodSpec{
					NodeName: "node",
					NodeSelector: map[string]string{
						"kubernetes.io/hostname": "node",
					},
					ImagePullSecrets: []corev1.LocalObjectReference{
						{Name: "fake-secret"},
					},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:           &user,
						RunAsGroup:          &group,
						FSGroup:             &group,
						FSGroupChangePolicy: &fsGroupChangeAlways,
						SupplementalGroups:  []int64{group},
					},
					InitContainers: []corev1.Container{
						{Name: "init", Image: "init"},
					},
					Containers: []corev1.Container{
						{
							Name:            "emqx",
							Image:           "emqx/emqx:4.4.6",
							ImagePullPolicy: corev1.PullAlways,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU: resource.MustParse("100m"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU: resource.MustParse("100m"),
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "EMQX_CLUSTER__DISCOVERY",
									Value: "dns",
								},
								{
									Name:  "EMQX_CLUSTER__DNS__APP",
									Value: "emqx",
								},
								{
									Name:  "EMQX_CLUSTER__DNS__NAME",
									Value: "emqx-headless.default.svc.cluster.local",
								},
								{
									Name:  "EMQX_CLUSTER__DNS__TYPE",
									Value: "srv",
								},
								{
									Name:  "EMQX_DASHBOARD__DEFAULT_USER__LOGIN",
									Value: "admin",
								},
								{
									Name:  "EMQX_DASHBOARD__DEFAULT_USER__PASSWORD",
									Value: "public",
								},
								{
									Name:  "EMQX_LISTENER__SSL__EXTERNAL",
									Value: "8883",
								},
								{
									Name:  "EMQX_LISTENER__TCP__EXTERNAL",
									Value: "1883",
								},
								{
									Name:  "EMQX_LISTENER__WSS__EXTERNAL",
									Value: "8084",
								},

								{
									Name:  "EMQX_LISTENER__WS__EXTERNAL",
									Value: "8083",
								},
								{
									Name:  "EMQX_LOG__LEVEL",
									Value: "debug",
								},
								{
									Name:  "EMQX_LOG__TO",
									Value: "file",
								},
								{
									Name:  "EMQX_MANAGEMENT__DEFAULT_APPLICATION__ID",
									Value: "admin",
								},

								{
									Name:  "EMQX_MANAGEMENT__DEFAULT_APPLICATION__SECRET",
									Value: "public",
								},
								{
									Name:  "EMQX_NAME",
									Value: "emqx",
								},
							},
							Args: []string{
								"--log.level", "debug",
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/status",
										Port: intstr.FromInt(8081),
									},
								},
								InitialDelaySeconds: int32(10),
								PeriodSeconds:       int32(5),
								FailureThreshold:    int32(30),
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "extra", MountPath: "/extra"},
								{Name: "emqx-data", MountPath: "/opt/emqx/data"},
								{Name: "emqx-log", MountPath: "/opt/emqx/log"},
							},
						},
						{
							Name:            "reloader",
							Image:           "emqx/emqx-operator-reloader:0.0.1",
							ImagePullPolicy: corev1.PullAlways,
							Args: []string{
								"-u", "admin",
								"-p", "public",
								"-P", "8081",
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "extra", MountPath: "/extra"},
								{Name: "emqx-data", MountPath: "/opt/emqx/data"},
								{Name: "emqx-log", MountPath: "/opt/emqx/log"},
							},
						},
						{Name: "extra", Image: "extra"},
					},
					Volumes: []corev1.Volume{
						{Name: "extra", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
					},
				},
			},
		},
	}

	assert.Equal(t, expect, generateStatefulSetDef(broker))

	broker.Spec.Persistent = corev1.PersistentVolumeClaimSpec{}
	assert.Nil(t, generateStatefulSetDef(broker).Spec.VolumeClaimTemplates)

	assert.Subset(t,
		[]corev1.Volume{
			{
				Name: "emqx-data",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: "emqx-log",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: "extra",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		},
		generateStatefulSetDef(broker).Spec.Template.Spec.Volumes)
}

func TestGenerateInitPluginList(t *testing.T) {
	enterprise := &appsv1beta3.EmqxEnterprise{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx-ee",
			Namespace: "default",
			Labels: map[string]string{
				"apps.emqx.io/instance": "emqx-ee",
			},
		},
	}

	emqxRuleEngine := &appsv1beta3.EmqxPlugin{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps.emqx.io/v1beta3",
			Kind:       "EmqxPlugin",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx-ee-rule-engine",
			Namespace: "default",
			Labels: map[string]string{
				"apps.emqx.io/instance": "emqx-ee",
			},
		},
		Spec: appsv1beta3.EmqxPluginSpec{
			PluginName: "emqx_rule_engine",
			Selector: map[string]string{
				"apps.emqx.io/instance": "emqx-ee",
			},
			Config: map[string]string{},
		},
	}

	emqxRetainer := &appsv1beta3.EmqxPlugin{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps.emqx.io/v1beta3",
			Kind:       "EmqxPlugin",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx-ee-retainer",
			Namespace: "default",
			Labels: map[string]string{
				"apps.emqx.io/instance": "emqx-ee",
			},
		},
		Spec: appsv1beta3.EmqxPluginSpec{
			PluginName: "emqx_retainer",
			Selector: map[string]string{
				"apps.emqx.io/instance": "emqx-ee",
			},
			Config: map[string]string{},
		},
	}

	emqxModules := &appsv1beta3.EmqxPlugin{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps.emqx.io/v1beta3",
			Kind:       "EmqxPlugin",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx-ee-modules",
			Namespace: "default",
			Labels: map[string]string{
				"apps.emqx.io/instance": "emqx-ee",
			},
		},
		Spec: appsv1beta3.EmqxPluginSpec{
			PluginName: "emqx_modules",
			Selector: map[string]string{
				"apps.emqx.io/instance": "emqx-ee",
			},
			Config: map[string]string{
				"modules.loaded_file": "/mounted/modules/loaded_modules",
			},
		},
	}

	fakePlugin := &appsv1beta3.EmqxPlugin{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps.emqx.io/v1beta3",
			Kind:       "EmqxPlugin",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx-ee-fake",
			Namespace: "default",
			Labels: map[string]string{
				"apps.emqx.io/instance": "emqx-ee",
			},
		},
		Spec: appsv1beta3.EmqxPluginSpec{
			PluginName: "emqx_fake",
			Selector: map[string]string{
				"apps.emqx.io/instance": "emqx-ee",
			},
			Config: map[string]string{},
		},
	}

	noMatchPlugin := &appsv1beta3.EmqxPlugin{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps.emqx.io/v1beta3",
			Kind:       "EmqxPlugin",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx-ee-no-match",
			Namespace: "default",
			Labels: map[string]string{
				"apps.emqx.io/instance": "no-match",
			},
		},
		Spec: appsv1beta3.EmqxPluginSpec{
			PluginName: "emqx_fake",
			Selector: map[string]string{
				"apps.emqx.io/instance": "no-match",
			},
			Config: map[string]string{},
		},
	}

	extraPluginList := &appsv1beta3.EmqxPluginList{
		Items: []appsv1beta3.EmqxPlugin{
			*noMatchPlugin,
			*fakePlugin,
		},
	}
	expect := []client.Object{
		emqxRuleEngine,
		emqxRetainer,
		emqxModules,
	}
	assert.Equal(t, expect, generateInitPluginList(enterprise, extraPluginList))

	extraPluginList = &appsv1beta3.EmqxPluginList{
		Items: []appsv1beta3.EmqxPlugin{
			*emqxRuleEngine,
		},
	}
	expect = []client.Object{
		emqxRetainer,
		emqxModules,
	}
	assert.Equal(t, expect, generateInitPluginList(enterprise, extraPluginList))

	extraPluginList = &appsv1beta3.EmqxPluginList{
		Items: []appsv1beta3.EmqxPlugin{
			*emqxRuleEngine,
			*emqxRetainer,
		},
	}
	expect = []client.Object{
		emqxModules,
	}
	assert.Equal(t, expect, generateInitPluginList(enterprise, extraPluginList))

	extraPluginList = &appsv1beta3.EmqxPluginList{
		Items: []appsv1beta3.EmqxPlugin{
			*emqxRuleEngine,
			*emqxRetainer,
			*emqxModules,
		},
	}
	expect = []client.Object{}
	assert.Equal(t, expect, generateInitPluginList(enterprise, extraPluginList))
}

func TestGenerateDefaultPluginsConfig(t *testing.T) {
	broker := &appsv1beta3.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "default",
			Labels: map[string]string{
				"apps.emqx.io/instance": "emqx",
			},
		},
	}

	sts := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "emqx"},
						{Name: "reloader"},
					},
				},
			},
		},
	}

	expect := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx-plugins-config",
			Namespace: "default",
			Labels: map[string]string{
				"apps.emqx.io/instance": "emqx",
			},
		},
	}

	expectSts := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "emqx",
							Env: []corev1.EnvVar{
								{
									Name:  "EMQX_PLUGINS__ETC_DIR",
									Value: "/mounted/plugins/etc",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "emqx-plugins-config",
									MountPath: "/mounted/plugins/etc",
								},
							},
						},
						{
							Name: "reloader",
							Env: []corev1.EnvVar{
								{
									Name:  "EMQX_PLUGINS__ETC_DIR",
									Value: "/mounted/plugins/etc",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "emqx-plugins-config",
									MountPath: "/mounted/plugins/etc",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "emqx-plugins-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "emqx-plugins-config",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	pluginsConfig, sts := generateDefaultPluginsConfig(broker, sts)
	assert.Equal(t, expect.ObjectMeta, pluginsConfig.ObjectMeta)
	assert.Equal(t, expectSts, sts)
}

func TestGenerateLoadedPlugins(t *testing.T) {
	borker := &appsv1beta3.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "default",
			Labels: map[string]string{
				"apps.emqx.io/instance": "emqx",
			},
		},
	}

	sts := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "emqx"},
						{Name: "reloader"},
					},
				},
			},
		},
	}

	expect := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx-loaded-plugins",
			Namespace: "default",
			Labels: map[string]string{
				"apps.emqx.io/instance": "emqx",
			},
		},
		Data: map[string]string{
			"loaded_plugins": "emqx_management.\nemqx_dashboard.\nemqx_retainer.\nemqx_rule_engine.\n",
		},
	}

	expectSts := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "emqx",
							Env: []corev1.EnvVar{
								{
									Name:  "EMQX_PLUGINS__LOADED_FILE",
									Value: "/mounted/plugins/data/loaded_plugins",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "emqx-loaded-plugins",
									MountPath: "/mounted/plugins/data",
								},
							},
						},
						{
							Name: "reloader",
							Env: []corev1.EnvVar{
								{
									Name:  "EMQX_PLUGINS__LOADED_FILE",
									Value: "/mounted/plugins/data/loaded_plugins",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "emqx-loaded-plugins",
									MountPath: "/mounted/plugins/data",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "emqx-loaded-plugins",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "emqx-loaded-plugins",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	loadedPlugins, sts := generateLoadedPlugins(borker, sts)
	assert.Equal(t, expect, loadedPlugins)
	assert.Equal(t, expectSts, sts)

	enterprise := &appsv1beta3.EmqxEnterprise{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx-ee",
			Namespace: "default",
			Labels: map[string]string{
				"apps.emqx.io/instance": "emqx-ee",
			},
		},
	}

	loadedPlugins, _ = generateLoadedPlugins(enterprise, sts)
	assert.Equal(t, "emqx_management.\nemqx_dashboard.\nemqx_retainer.\nemqx_rule_engine.\nemqx_modules.\n", loadedPlugins.Data["loaded_plugins"])
}

func TestGenerateSvc(t *testing.T) {
	broker := &appsv1beta3.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "default",
			Annotations: map[string]string{
				"apps.emqx.io/test": "fake",
			},
		},
		Spec: appsv1beta3.EmqxBrokerSpec{
			EmqxTemplate: appsv1beta3.EmqxBrokerTemplate{
				ServiceTemplate: appsv1beta3.ServiceTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake",
						Namespace: "fake",
						Labels: map[string]string{
							"foo": "bar",
						},
						Annotations: map[string]string{
							"foo": "bar",
						},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name:       "mqtt-tcp-1883",
								Port:       1883,
								Protocol:   corev1.ProtocolTCP,
								TargetPort: intstr.FromInt(1883),
							},
						},
					},
				},
			},
		},
	}
	broker.Default()

	expectSvc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake",
			Namespace: "default",
			Labels: map[string]string{
				"apps.emqx.io/instance":   "emqx",
				"apps.emqx.io/managed-by": "emqx-operator",
				"foo":                     "bar",
			},
			Annotations: map[string]string{
				"apps.emqx.io/test": "fake",
				"foo":               "bar",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"apps.emqx.io/instance":   "emqx",
				"apps.emqx.io/managed-by": "emqx-operator",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "mqtt-tcp-1883",
					Port:       1883,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(1883),
				},
				{
					Name:       "http-management-8081",
					Port:       8081,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8081),
				},
			},
		},
	}

	expectHeadlessSvc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx-headless",
			Namespace: "default",
			Labels: map[string]string{
				"apps.emqx.io/instance":   "emqx",
				"apps.emqx.io/managed-by": "emqx-operator",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"apps.emqx.io/instance":   "emqx",
				"apps.emqx.io/managed-by": "emqx-operator",
			},
			ClusterIP: corev1.ClusterIPNone,
			Ports: []corev1.ServicePort{
				{
					Name:       "http-management-8081",
					Port:       8081,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8081),
				},
			},
		},
	}

	headlessSvc, svc := generateSvc(broker)
	assert.Equal(t, expectHeadlessSvc, headlessSvc)
	assert.Equal(t, expectSvc, svc)
}

func TestGenerateAcl(t *testing.T) {
	broker := &appsv1beta3.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "default",
			Labels: map[string]string{
				"apps.emqx.io/instance": "emqx",
			},
		},
	}

	sts := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "emqx"},
						{Name: "reloader"},
					},
				},
			},
		},
	}

	acl, _ := generateAcl(broker, sts)
	assert.Nil(t, acl)

	broker.Spec.EmqxTemplate.ACL = []string{
		"{allow, all}",
		"{deny, all}",
	}

	expect := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx-acl",
			Namespace: "default",
			Labels: map[string]string{
				"apps.emqx.io/instance": "emqx",
			},
		},
		Data: map[string]string{
			"acl.conf": "{allow, all}\n{deny, all}\n",
		},
	}

	expectSts := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "emqx",
							Env: []corev1.EnvVar{
								{
									Name:  "EMQX_ACL_FILE",
									Value: "/mounted/acl/acl.conf",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "emqx-acl",
									MountPath: "/mounted/acl",
								},
							},
						},
						{
							Name: "reloader",
							Env: []corev1.EnvVar{
								{
									Name:  "EMQX_ACL_FILE",
									Value: "/mounted/acl/acl.conf",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "emqx-acl",
									MountPath: "/mounted/acl",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "emqx-acl",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "emqx-acl",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	acl, sts = generateAcl(broker, sts)
	assert.Equal(t, expect, acl)
	assert.Equal(t, expectSts, sts)
}

func TestGenerateLoadedModules(t *testing.T) {
	broker := &appsv1beta3.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "default",
			Labels: map[string]string{
				"apps.emqx.io/instance": "emqx",
			},
		},
	}

	sts := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "emqx"},
						{Name: "reloader"},
					},
				},
			},
		},
	}

	modules, _ := generateLoadedModules(broker, sts)
	assert.Nil(t, modules)

	broker.Spec.EmqxTemplate.Modules = []appsv1beta3.EmqxBrokerModule{
		{
			Name:   "emqx_module",
			Enable: true,
		},
	}

	expect := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx-loaded-modules",
			Namespace: "default",
			Labels: map[string]string{
				"apps.emqx.io/instance": "emqx",
			},
		},
		Data: map[string]string{
			"loaded_modules": "{emqx_module, true}.\n",
		},
	}

	expectSts := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "emqx",
							Env: []corev1.EnvVar{
								{
									Name:  "EMQX_MODULES__LOADED_FILE",
									Value: "/mounted/modules/loaded_modules",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "emqx-loaded-modules",
									MountPath: "/mounted/modules",
								},
							},
						},
						{
							Name: "reloader",
							Env: []corev1.EnvVar{
								{
									Name:  "EMQX_MODULES__LOADED_FILE",
									Value: "/mounted/modules/loaded_modules",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "emqx-loaded-modules",
									MountPath: "/mounted/modules",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "emqx-loaded-modules",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "emqx-loaded-modules",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	loadedBrokerModules, sts := generateLoadedModules(broker, sts)
	assert.Equal(t, expect, loadedBrokerModules)
	assert.Equal(t, expectSts, sts)

	enterpise := &appsv1beta3.EmqxEnterprise{
		Spec: appsv1beta3.EmqxEnterpriseSpec{
			EmqxTemplate: appsv1beta3.EmqxEnterpriseTemplate{
				Modules: []appsv1beta3.EmqxEnterpriseModule{
					{
						Name:    "internal_acl",
						Enable:  true,
						Configs: runtime.RawExtension{Raw: []byte(`{"acl_rule_file": "/mounted/acl/acl.conf"}`)},
					},
				},
			},
		},
	}

	loadedEnterpriseModules, _ := generateLoadedModules(enterpise, sts)
	assert.Equal(t, `[{"name":"internal_acl","enable":true,"configs":{"acl_rule_file":"/mounted/acl/acl.conf"}}]`, loadedEnterpriseModules.Data["loaded_modules"])
}

func TestGenerateLicense(t *testing.T) {
	broker := &appsv1beta3.EmqxBroker{}
	sts := &appsv1.StatefulSet{}

	license, _ := generateLicense(broker, sts)
	assert.Nil(t, license)

	enterprise := &appsv1beta3.EmqxEnterprise{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx-ee",
			Namespace: "default",
			Labels: map[string]string{
				"apps.emqx.io/instance": "emqx",
			},
		},
	}

	license, _ = generateLicense(enterprise, sts)
	assert.Nil(t, license)

	enterprise.Spec.EmqxTemplate.License.Data = []byte(`-----BEGIN CERTIFICATE-----
	MIIENzCCAx+gAwIBAgIDdMvVMA0GCSqGSIb3DQEBBQUAMIGDMQswCQYDVQQGEwJD
	TjERMA8GA1UECAwIWmhlamlhbmcxETAPBgNVBAcMCEhhbmd6aG91MQwwCgYDVQQK
	DANFTVExDDAKBgNVBAsMA0VNUTESMBAGA1UEAwwJKi5lbXF4LmlvMR4wHAYJKoZI
	hvcNAQkBFg96aGFuZ3doQGVtcXguaW8wHhcNMjAwNjIwMDMwMjUyWhcNNDkwMTAx
	MDMwMjUyWjBjMQswCQYDVQQGEwJDTjEZMBcGA1UECgwQRU1RIFggRXZhbHVhdGlv
	bjEZMBcGA1UEAwwQRU1RIFggRXZhbHVhdGlvbjEeMBwGCSqGSIb3DQEJARYPY29u
	dGFjdEBlbXF4LmlvMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArw+3
	2w9B7Rr3M7IOiMc7OD3Nzv2KUwtK6OSQ07Y7ikDJh0jynWcw6QamTiRWM2Ale8jr
	0XAmKgwUSI42+f4w84nPpAH4k1L0zupaR10VYKIowZqXVEvSyV8G2N7091+6Jcon
	DcaNBqZLRe1DiZXMJlhXnDgq14FPAxffKhCXiCgYtluLDDLKv+w9BaQGZVjxlFe5
	cw32+z/xHU366npHBpafCbxBtWsNvchMVtLBqv9yPmrMqeBROyoJaI3nL78xDgpd
	cRorqo+uQ1HWdcM6InEFET6pwkeuAF8/jJRlT12XGgZKKgFQTCkZi4hv7aywkGBE
	JruPif/wlK0YuPJu6QIDAQABo4HSMIHPMBEGCSsGAQQBg5odAQQEDAIxMDCBlAYJ
	KwYBBAGDmh0CBIGGDIGDZW1xeF9iYWNrZW5kX3JlZGlzLGVtcXhfYmFja2VuZF9t
	eXNxbCxlbXF4X2JhY2tlbmRfcGdzcWwsZW1xeF9iYWNrZW5kX21vbmdvLGVtcXhf
	YmFja2VuZF9jYXNzYSxlbXF4X2JyaWRnZV9rYWZrYSxlbXF4X2JyaWRnZV9yYWJi
	aXQwEAYJKwYBBAGDmh0DBAMMATEwEQYJKwYBBAGDmh0EBAQMAjEwMA0GCSqGSIb3
	DQEBBQUAA4IBAQDHUe6+P2U4jMD23u96vxCeQrhc/rXWvpmU5XB8Q/VGnJTmv3yU
	EPyTFKtEZYVX29z16xoipUE6crlHhETOfezYsm9K0DxF3fNilOLRKkg9VEWcb5hj
	iL3a2tdZ4sq+h/Z1elIXD71JJBAImjr6BljTIdUCfVtNvxlE8M0D/rKSn2jwzsjI
	UrW88THMtlz9sb56kmM3JIOoIJoep6xNEajIBnoChSGjtBYFNFwzdwSTCodYkgPu
	JifqxTKSuwAGSlqxJUwhjWG8ulzL3/pCAYEwlWmd2+nsfotQdiANdaPnez7o0z0s
	EujOCZMbK8qNfSbyo50q5iIXhz2ZIGl+4hdp
	-----END CERTIFICATE-----`)

	sts = &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "emqx"},
						{Name: "reloader"},
					},
				},
			},
		},
	}

	expect := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx-ee-license",
			Namespace: "default",
			Labels: map[string]string{
				"apps.emqx.io/instance": "emqx",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"emqx.lic": enterprise.Spec.EmqxTemplate.License.Data,
		},
	}

	expectSts := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "emqx",
							Env: []corev1.EnvVar{
								{
									Name:  "EMQX_LICENSE__FILE",
									Value: "/mounted/license/emqx.lic",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "emqx-ee-license",
									MountPath: "/mounted/license",
									ReadOnly:  true,
								},
							},
						},
						{
							Name: "reloader",
							Env: []corev1.EnvVar{
								{
									Name:  "EMQX_LICENSE__FILE",
									Value: "/mounted/license/emqx.lic",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "emqx-ee-license",
									MountPath: "/mounted/license",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "emqx-ee-license",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "emqx-ee-license",
								},
							},
						},
					},
				},
			},
		},
	}

	license, sts = generateLicense(enterprise, sts)
	assert.Equal(t, expect, license)
	assert.Equal(t, expectSts, sts)
}

func TestGenerateAnnotationByContainers(t *testing.T) {
	assert.Equal(t, "emqx,reloader",
		generateAnnotationByContainers([]corev1.Container{
			{Name: "emqx"}, {Name: "reloader"},
		}))

}

func TestUpdateEmqxStatus(t *testing.T) {
	broker := &appsv1beta3.EmqxBroker{
		Spec: appsv1beta3.EmqxBrokerSpec{
			Replicas: &replicas,
		},
	}

	emqxNodes := []appsv1beta3.EmqxNode{
		{
			Node:       "node0",
			NodeStatus: "Running",
			OTPRelease: "",
			Version:    "",
			Uptime:     "",
		},
	}

	re := updateEmqxStatus(broker, emqxNodes)
	status := re.GetStatus()
	assert.Equal(t, int32(3), status.Replicas)
	assert.Equal(t, int32(1), status.ReadyReplicas)
	assert.Equal(t, emqxNodes, status.EmqxNodes)
	assert.False(t, status.IsRunning())

	emqxNodes = append(emqxNodes, []appsv1beta3.EmqxNode{
		{
			Node:       "node1",
			NodeStatus: "Running",
			OTPRelease: "",
			Version:    "",
			Uptime:     "",
		},
		{
			Node:       "node2",
			NodeStatus: "Running",
			OTPRelease: "",
			Version:    "",
			Uptime:     "",
		},
	}...)

	re = updateEmqxStatus(broker, emqxNodes)
	status = re.GetStatus()
	assert.Equal(t, int32(3), status.Replicas)
	assert.Equal(t, int32(3), status.ReadyReplicas)
	assert.Equal(t, emqxNodes, status.EmqxNodes)
	assert.True(t, status.IsRunning())
}
