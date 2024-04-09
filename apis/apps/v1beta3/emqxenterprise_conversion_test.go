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

package v1beta3

import (
	"testing"

	"github.com/emqx/emqx-operator/apis/apps/v1beta4"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

var v1bete3EmqxEnterprise = &EmqxEnterprise{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "emqx",
		Namespace: "default",
		Labels: map[string]string{
			"foo": "bar",
		},
		Annotations: map[string]string{
			"foo": "bar",
		},
	},
	Spec: EmqxEnterpriseSpec{
		Replicas: ptr.To(int32(3)),
		Env: []corev1.EnvVar{
			{
				Name:  "foo",
				Value: "bar",
			},
		},
		ImagePullSecrets: []corev1.LocalObjectReference{
			{
				Name: "fake-secret",
			},
		},
		NodeName: "fake-node",
		NodeSelector: map[string]string{
			"foo": "bar",
		},
		Affinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchFields: []corev1.NodeSelectorRequirement{
								{
									Key: "foo",
								},
							},
						},
					},
				},
			},
		},
		ToleRations: []corev1.Toleration{
			{
				Key: "foo",
			},
		},
		Persistent: corev1.PersistentVolumeClaimSpec{
			StorageClassName: ptr.To("foo"),
		},
		InitContainers: []corev1.Container{
			{
				Name: "fake-init-container",
			},
		},
		ExtraContainers: []corev1.Container{
			{
				Name: "fake-extra-container",
			},
		},
		EmqxTemplate: EmqxEnterpriseTemplate{
			Image: "emqx/emqx-ee:4.4.8",
			License: License{
				SecretName: "fake-license-secret",
			},
			EmqxConfig: map[string]string{
				"foo": "bar",
			},
			ACL:             []string{"allow, all."},
			ImagePullPolicy: corev1.PullIfNotPresent,
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser:           ptr.To(int64(1000)),
				RunAsGroup:          ptr.To(int64(1000)),
				FSGroup:             ptr.To(int64(1000)),
				FSGroupChangePolicy: &[]corev1.PodFSGroupChangePolicy{corev1.FSGroupChangeAlways}[0],
			},
			ExtraVolumes: []corev1.Volume{
				{
					Name: "fake-extra-volume",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
			ExtraVolumeMounts: []corev1.VolumeMount{
				{
					Name:      "fake-extra-volume-mount",
					MountPath: "/fake-extra-volume-mount",
				},
			},
			Args: []string{"-foo", "bar"},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/status",
						Port: intstr.FromInt(8081),
					},
				},
				InitialDelaySeconds: 10,
				PeriodSeconds:       5,
				FailureThreshold:    12,
			},
			LivenessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/status",
						Port: intstr.FromInt(8081),
					},
				},
				InitialDelaySeconds: 60,
				PeriodSeconds:       30,
				FailureThreshold:    3,
			},
			StartupProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/status",
						Port: intstr.FromInt(8081),
					},
				},
				InitialDelaySeconds: 60,
				PeriodSeconds:       30,
				FailureThreshold:    3,
			},
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("125m"),
					corev1.ResourceMemory: resource.MustParse("64Mi"),
				},
			},
			ServiceTemplate: ServiceTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "emqx",
					Namespace: "default",
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{
						"foo": "bar",
					},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
					Selector: map[string]string{
						"foo": "bar",
					},
					Ports: []corev1.ServicePort{
						{
							Name:       "http-management-8081",
							Port:       8081,
							Protocol:   corev1.ProtocolTCP,
							TargetPort: intstr.FromInt(8081),
						},
					},
				},
			},
		},
	},
}

var v1beta4EmqxEnterprise = &v1beta4.EmqxEnterprise{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "emqx",
		Namespace: "default",
		Labels: map[string]string{
			"foo": "bar",
		},
		Annotations: map[string]string{
			"foo": "bar",
		},
	},
	Spec: v1beta4.EmqxEnterpriseSpec{
		Replicas: ptr.To(int32(3)),
		License: v1beta4.EmqxLicense{
			SecretName: "fake-license-secret",
		},
		Persistent: &corev1.PersistentVolumeClaimTemplate{
			Spec: corev1.PersistentVolumeClaimSpec{
				StorageClassName: ptr.To("foo"),
			},
		},
		Template: v1beta4.EmqxTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"foo": "bar",
				},
				Annotations: map[string]string{
					"foo": "bar",
				},
			},
			Spec: v1beta4.EmqxTemplateSpec{
				NodeName: "fake-node",
				NodeSelector: map[string]string{
					"foo": "bar",
				},
				Affinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{
								{
									MatchFields: []corev1.NodeSelectorRequirement{
										{
											Key: "foo",
										},
									},
								},
							},
						},
					},
				},
				Tolerations: []corev1.Toleration{
					{
						Key: "foo",
					},
				},
				ImagePullSecrets: []corev1.LocalObjectReference{
					{
						Name: "fake-secret",
					},
				},
				InitContainers: []corev1.Container{
					{
						Name: "fake-init-container",
					},
				},
				ExtraContainers: []corev1.Container{
					{
						Name: "fake-extra-container",
					},
				},
				EmqxContainer: v1beta4.EmqxContainer{
					Image: v1beta4.EmqxImage{
						Repository: "emqx/emqx-ee",
						Version:    "4.4.8",
						PullPolicy: corev1.PullIfNotPresent,
					},
					EmqxConfig: map[string]string{
						"foo": "bar",
					},
					EmqxACL: []string{"allow, all."},
					Env: []corev1.EnvVar{
						{
							Name:  "foo",
							Value: "bar",
						},
					},
					Args: []string{"-foo", "bar"},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/status",
								Port: intstr.FromInt(8081),
							},
						},
						InitialDelaySeconds: 10,
						PeriodSeconds:       5,
						FailureThreshold:    12,
					},
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/status",
								Port: intstr.FromInt(8081),
							},
						},
						InitialDelaySeconds: 60,
						PeriodSeconds:       30,
						FailureThreshold:    3,
					},
					StartupProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/status",
								Port: intstr.FromInt(8081),
							},
						},
						InitialDelaySeconds: 60,
						PeriodSeconds:       30,
						FailureThreshold:    3,
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("1Gi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("125m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "fake-extra-volume-mount",
							MountPath: "/fake-extra-volume-mount",
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "fake-extra-volume",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
				PodSecurityContext: &corev1.PodSecurityContext{
					RunAsUser:           ptr.To(int64(1000)),
					RunAsGroup:          ptr.To(int64(1000)),
					FSGroup:             ptr.To(int64(1000)),
					FSGroupChangePolicy: &[]corev1.PodFSGroupChangePolicy{corev1.FSGroupChangeAlways}[0],
				},
			},
		},
		ServiceTemplate: v1beta4.ServiceTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "emqx",
				Namespace: "default",
				Labels: map[string]string{
					"foo": "bar",
				},
				Annotations: map[string]string{
					"foo": "bar",
				},
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeNodePort,
				Selector: map[string]string{
					"foo": "bar",
				},
				Ports: []corev1.ServicePort{
					{
						Name:       "http-management-8081",
						Port:       8081,
						Protocol:   corev1.ProtocolTCP,
						TargetPort: intstr.FromInt(8081),
					},
				},
			},
		},
	},
}

func TestEnterpriseConversionTo(t *testing.T) {
	emqx := &v1beta4.EmqxEnterprise{}
	err := v1bete3EmqxEnterprise.ConvertTo(emqx)
	assert.Nil(t, err)

	assert.Equal(t, v1bete3EmqxEnterprise.ObjectMeta, emqx.ObjectMeta)

	assert.Equal(t, v1bete3EmqxEnterprise.Spec.Replicas, emqx.Spec.Replicas)
	assert.Equal(t, v1bete3EmqxEnterprise.Spec.Persistent, emqx.Spec.Persistent.Spec)
	assert.ObjectsAreEqualValues(v1bete3EmqxEnterprise.Spec.EmqxTemplate.ServiceTemplate, emqx.Spec.ServiceTemplate)

	assert.Equal(t, v1bete3EmqxBroker.Labels, emqx.Spec.Template.Labels)
	assert.Equal(t, v1bete3EmqxBroker.Annotations, emqx.Spec.Template.Annotations)
	assert.Equal(t, "emqx/emqx-ee", emqx.Spec.Template.Spec.EmqxContainer.Image.Repository)
	assert.Equal(t, "4.4.8", emqx.Spec.Template.Spec.EmqxContainer.Image.Version)
	assert.Equal(t, v1bete3EmqxEnterprise.Spec.EmqxTemplate.ImagePullPolicy, emqx.Spec.Template.Spec.EmqxContainer.Image.PullPolicy)
	assert.ObjectsAreEqualValues(v1bete3EmqxEnterprise.Spec.EmqxTemplate.License, emqx.Spec.License)
	assert.ObjectsAreEqualValues(v1bete3EmqxEnterprise.Spec.EmqxTemplate.EmqxConfig, emqx.Spec.Template.Spec.EmqxContainer.EmqxConfig)
	assert.Equal(t, v1bete3EmqxEnterprise.Spec.EmqxTemplate.ACL, emqx.Spec.Template.Spec.EmqxContainer.EmqxACL)
	assert.Equal(t, v1bete3EmqxEnterprise.Spec.EmqxTemplate.Args, emqx.Spec.Template.Spec.EmqxContainer.Args)
	assert.Equal(t, v1bete3EmqxEnterprise.Spec.EmqxTemplate.Resources, emqx.Spec.Template.Spec.EmqxContainer.Resources)
	assert.Equal(t, v1bete3EmqxEnterprise.Spec.EmqxTemplate.ReadinessProbe, emqx.Spec.Template.Spec.EmqxContainer.ReadinessProbe)
	assert.Equal(t, v1bete3EmqxEnterprise.Spec.EmqxTemplate.LivenessProbe, emqx.Spec.Template.Spec.EmqxContainer.LivenessProbe)
	assert.Equal(t, v1bete3EmqxEnterprise.Spec.EmqxTemplate.StartupProbe, emqx.Spec.Template.Spec.EmqxContainer.StartupProbe)
	assert.Equal(t, v1bete3EmqxEnterprise.Spec.EmqxTemplate.ExtraVolumeMounts, emqx.Spec.Template.Spec.EmqxContainer.VolumeMounts)
	assert.Equal(t, v1bete3EmqxEnterprise.Spec.EmqxTemplate.ExtraVolumes, emqx.Spec.Template.Spec.Volumes)
	assert.Equal(t, v1bete3EmqxEnterprise.Spec.EmqxTemplate.SecurityContext, emqx.Spec.Template.Spec.PodSecurityContext)
	assert.Equal(t, v1bete3EmqxEnterprise.Spec.InitContainers, emqx.Spec.Template.Spec.InitContainers)
	assert.Equal(t, v1bete3EmqxEnterprise.Spec.ExtraContainers, emqx.Spec.Template.Spec.ExtraContainers)
	assert.Equal(t, v1bete3EmqxEnterprise.Spec.ImagePullSecrets, emqx.Spec.Template.Spec.ImagePullSecrets)
	assert.Equal(t, v1bete3EmqxEnterprise.Spec.Env, emqx.Spec.Template.Spec.EmqxContainer.Env)
	assert.Equal(t, v1bete3EmqxEnterprise.Spec.ToleRations, emqx.Spec.Template.Spec.Tolerations)
	assert.Equal(t, v1bete3EmqxEnterprise.Spec.NodeName, emqx.Spec.Template.Spec.NodeName)
	assert.Equal(t, v1bete3EmqxEnterprise.Spec.NodeSelector, emqx.Spec.Template.Spec.NodeSelector)
	assert.Equal(t, v1bete3EmqxEnterprise.Spec.Affinity, emqx.Spec.Template.Spec.Affinity)
}

func TestEnterpriseConversionFrom(t *testing.T) {
	emqx := &EmqxEnterprise{}
	err := emqx.ConvertFrom(v1beta4EmqxEnterprise)
	assert.Nil(t, err)
	assert.Equal(t, v1beta4EmqxEnterprise.ObjectMeta, emqx.ObjectMeta)

	assert.Equal(t, v1beta4EmqxEnterprise.Spec.Replicas, emqx.Spec.Replicas)
	assert.Equal(t, v1beta4EmqxEnterprise.Spec.Persistent.Spec, emqx.Spec.Persistent)
	assert.ObjectsAreEqualValues(v1beta4EmqxEnterprise.Spec.ServiceTemplate, emqx.Spec.EmqxTemplate.ServiceTemplate)

	assert.Equal(t, "emqx/emqx-ee:4.4.8", emqx.Spec.EmqxTemplate.Image)
	assert.Equal(t, v1beta4EmqxEnterprise.Spec.Template.Spec.EmqxContainer.Image.PullPolicy, emqx.Spec.EmqxTemplate.ImagePullPolicy)

	assert.ObjectsAreEqualValues(v1beta4EmqxEnterprise.Spec.License, emqx.Spec.EmqxTemplate.License)
	assert.ObjectsAreEqualValues(v1beta4EmqxEnterprise.Spec.Template.Spec.EmqxContainer.EmqxConfig, emqx.Spec.EmqxTemplate.EmqxConfig)
	assert.Equal(t, v1beta4EmqxEnterprise.Spec.Template.Spec.EmqxContainer.EmqxACL, emqx.Spec.EmqxTemplate.ACL)
	assert.Equal(t, v1beta4EmqxEnterprise.Spec.Template.Spec.EmqxContainer.Args, emqx.Spec.EmqxTemplate.Args)
	assert.Equal(t, v1beta4EmqxEnterprise.Spec.Template.Spec.EmqxContainer.Resources, emqx.Spec.EmqxTemplate.Resources)
	assert.Equal(t, v1beta4EmqxEnterprise.Spec.Template.Spec.EmqxContainer.ReadinessProbe, emqx.Spec.EmqxTemplate.ReadinessProbe)
	assert.Equal(t, v1beta4EmqxEnterprise.Spec.Template.Spec.EmqxContainer.LivenessProbe, emqx.Spec.EmqxTemplate.LivenessProbe)
	assert.Equal(t, v1beta4EmqxEnterprise.Spec.Template.Spec.EmqxContainer.StartupProbe, emqx.Spec.EmqxTemplate.StartupProbe)
	assert.Equal(t, v1beta4EmqxEnterprise.Spec.Template.Spec.EmqxContainer.VolumeMounts, emqx.Spec.EmqxTemplate.ExtraVolumeMounts)
	assert.Equal(t, v1beta4EmqxEnterprise.Spec.Template.Spec.Volumes, emqx.Spec.EmqxTemplate.ExtraVolumes)
	assert.Equal(t, v1beta4EmqxEnterprise.Spec.Template.Spec.PodSecurityContext, emqx.Spec.EmqxTemplate.SecurityContext)
	assert.Equal(t, v1beta4EmqxEnterprise.Spec.Template.Spec.InitContainers, emqx.Spec.InitContainers)
	assert.Equal(t, v1beta4EmqxEnterprise.Spec.Template.Spec.ExtraContainers, emqx.Spec.ExtraContainers)
	assert.Equal(t, v1beta4EmqxEnterprise.Spec.Template.Spec.ImagePullSecrets, emqx.Spec.ImagePullSecrets)
	assert.Equal(t, v1beta4EmqxEnterprise.Spec.Template.Spec.EmqxContainer.Env, emqx.Spec.Env)
	assert.Equal(t, v1beta4EmqxEnterprise.Spec.Template.Spec.Tolerations, emqx.Spec.ToleRations)
	assert.Equal(t, v1beta4EmqxEnterprise.Spec.Template.Spec.NodeName, emqx.Spec.NodeName)
	assert.Equal(t, v1beta4EmqxEnterprise.Spec.Template.Spec.NodeSelector, emqx.Spec.NodeSelector)
	assert.Equal(t, v1beta4EmqxEnterprise.Spec.Template.Spec.Affinity, emqx.Spec.Affinity)

	assert.Equal(t, v1beta4EmqxEnterprise.Spec.Replicas, emqx.Spec.Replicas)
	assert.Equal(t, v1beta4EmqxEnterprise.Spec.Persistent.Spec, emqx.Spec.Persistent)
	assert.ObjectsAreEqualValues(v1beta4EmqxEnterprise.Spec.ServiceTemplate, emqx.Spec.EmqxTemplate.ServiceTemplate)

}
