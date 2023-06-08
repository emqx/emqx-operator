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

package v2alpha2

import (
	"testing"

	// "github.com/gurkankaymak/hocon"
	hocon "github.com/rory-z/go-hocon"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

func TestDefault(t *testing.T) {
	instance := &EMQX{}
	instance.Default()
}

func TestValidateCreate(t *testing.T) {
	instance := &EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "webhook-test",
			Namespace: "default",
		},
		Spec: EMQXSpec{
			Image: "emqx:latest",
		},
	}
	assert.Nil(t, instance.ValidateCreate())

	instance.Spec.BootstrapConfig = "fake"
	assert.Error(t, instance.ValidateCreate(), "failed to parse bootstrap config")

	instance.Spec.BootstrapConfig = "foo = bar"
	assert.Nil(t, instance.ValidateCreate())
}

func TestValidateUpdate(t *testing.T) {
	instance := &EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "webhook-test",
			Namespace: "default",
		},
		Spec: EMQXSpec{
			Image:           "emqx:latest",
			BootstrapConfig: "fake",
		},
	}

	t.Run("should return error if bootstrap config is invalid", func(t *testing.T) {
		old := instance.DeepCopy()
		instance.Spec.BootstrapConfig = "hello world"
		assert.Error(t, instance.ValidateUpdate(old), "failed to parse bootstrap config")
	})

	t.Run("should return error if bootstrap APIKeys is changed", func(t *testing.T) {
		old := instance.DeepCopy()
		instance.Spec.BootstrapAPIKeys = []BootstrapAPIKey{{
			Key:    "test",
			Secret: "test",
		}}
		assert.Error(t, instance.ValidateUpdate(old), "bootstrap APIKeys cannot be updated")
	})

	t.Run("should return error if bootstrap config is changed", func(t *testing.T) {
		old := instance.DeepCopy()
		instance.Spec.BootstrapConfig = "foo = bar"
		assert.Error(t, instance.ValidateUpdate(old), "bootstrap config cannot be updated")
	})

	t.Run("check bootstrap config is map", func(t *testing.T) {
		old := instance.DeepCopy()
		old.Spec.BootstrapConfig = `{a = 1, b = { c = 2, d = 3}}`

		instance.Spec.BootstrapConfig = `{b = { d = 3, c = 2 }, a = 1}`
		assert.Nil(t, instance.ValidateUpdate(old))
	})

	t.Run("should return error if .spec.coreTemplate.metadata is update", func(t *testing.T) {
		old := instance.DeepCopy()
		old.Spec.CoreTemplate.Labels = map[string]string{"foo": "bar"}
		assert.Error(t, instance.ValidateUpdate(old), "coreTemplate.metadata and .spec.replicantTemplate.metadata cannot be updated")
	})

	t.Run("should return error if .spec.replicant.metadata is update", func(t *testing.T) {
		old := instance.DeepCopy()
		old.Spec.ReplicantTemplate.Labels = map[string]string{"foo": "bar"}
		assert.Error(t, instance.ValidateUpdate(old), "coreTemplate.metadata and .spec.replicantTemplate.metadata cannot be updated")
	})
}

func TestValidateDelete(t *testing.T) {
	instance := &EMQX{}
	assert.Nil(t, instance.ValidateDelete())
}

func TestDefaultName(t *testing.T) {
	instance := &EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name: "webhook-test",
		},
	}
	instance.defaultNames()
	assert.Equal(t, "webhook-test", instance.Name)
	assert.Equal(t, "webhook-test-core", instance.Spec.CoreTemplate.Name)
	assert.Equal(t, "webhook-test-replicant", instance.Spec.ReplicantTemplate.Name)
	assert.Equal(t, "webhook-test-dashboard", instance.Spec.DashboardServiceTemplate.Name)
	assert.Equal(t, "webhook-test-listeners", instance.Spec.ListenersServiceTemplate.Name)
}

func TestDefaultLabels(t *testing.T) {
	instance := &EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "webhook-test",
			Namespace: "default",
		},
	}
	instance.defaultLabels()

	assert.Equal(t, map[string]string{
		"apps.emqx.io/managed-by": "emqx-operator",
		"apps.emqx.io/instance":   "webhook-test",
	}, instance.Labels)

	assert.Equal(t, map[string]string{
		"apps.emqx.io/managed-by": "emqx-operator",
		"apps.emqx.io/instance":   "webhook-test",
	}, instance.Spec.DashboardServiceTemplate.Labels)

	assert.Equal(t, map[string]string{
		"apps.emqx.io/managed-by": "emqx-operator",
		"apps.emqx.io/instance":   "webhook-test",
	}, instance.Spec.ListenersServiceTemplate.Labels)

	assert.Equal(t, map[string]string{
		"apps.emqx.io/managed-by": "emqx-operator",
		"apps.emqx.io/instance":   "webhook-test",
		"apps.emqx.io/db-role":    "core",
	}, instance.Spec.CoreTemplate.Labels)

	assert.Equal(t, map[string]string{
		"apps.emqx.io/managed-by": "emqx-operator",
		"apps.emqx.io/instance":   "webhook-test",
		"apps.emqx.io/db-role":    "replicant",
	}, instance.Spec.ReplicantTemplate.Labels)
}

func TestDefaultBootstrapConfig(t *testing.T) {
	t.Run("empty bootstrap config", func(t *testing.T) {
		instance := &EMQX{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "webhook-test",
				Namespace: "default",
			},
			Spec: EMQXSpec{
				BootstrapConfig: "",
			},
		}
		instance.defaultBootstrapConfig()

		bootstrapConfig, err := hocon.ParseString(instance.Spec.BootstrapConfig)
		assert.Nil(t, err)

		assert.NotNil(t, bootstrapConfig.GetString("node.cookie"))
		assert.Equal(t, "data", bootstrapConfig.GetString("node.data_dir"))
		assert.Equal(t, "etc", bootstrapConfig.GetString("node.etc_dir"))

		assert.Equal(t, "18083", bootstrapConfig.GetString("dashboard.listeners.http.bind"))
		assert.Equal(t, "admin", bootstrapConfig.GetString("dashboard.default_username"))
		assert.Equal(t, "public", bootstrapConfig.GetString("dashboard.default_password"))

		assert.Equal(t, "\"0.0.0.0:1883\"", bootstrapConfig.GetString("listeners.tcp.default.bind"))
		assert.Equal(t, "1024000", bootstrapConfig.GetString("listeners.tcp.default.max_connections"))
	})

	t.Run("already set cookie", func(t *testing.T) {
		instance := &EMQX{
			Spec: EMQXSpec{
				BootstrapConfig: `node.cookie = "6gokwjslds3rcx256bkyrv9hnefft2zz7h4ezhzjmalehjedwlliisxtt7nsbvbq"`,
			},
		}
		instance.defaultBootstrapConfig()

		bootstrapConfig, err := hocon.ParseString(instance.Spec.BootstrapConfig)
		assert.Nil(t, err)
		assert.Equal(t, "\"6gokwjslds3rcx256bkyrv9hnefft2zz7h4ezhzjmalehjedwlliisxtt7nsbvbq\"", bootstrapConfig.GetString("node.cookie"))
	})

	t.Run("already set listener", func(t *testing.T) {
		instance := &EMQX{
			Spec: EMQXSpec{
				BootstrapConfig: `listeners.tcp.default.bind = "0.0.0.0:11883"`,
			},
		}
		instance.defaultBootstrapConfig()

		bootstrapConfig, err := hocon.ParseString(instance.Spec.BootstrapConfig)
		assert.Nil(t, err)
		assert.Equal(t, "\"0.0.0.0:11883\"", bootstrapConfig.GetString("listeners.tcp.default.bind"))
		assert.Equal(t, "1024000", bootstrapConfig.GetString("listeners.tcp.default.max_connections"))
	})

	t.Run("other style set listener", func(t *testing.T) {
		instance := &EMQX{
			Spec: EMQXSpec{
				BootstrapConfig: `
					listeners {
						tcp {
							default {
								bind = "0.0.0.0:11883"
							}
						}
					}
					`,
			},
		}
		instance.defaultBootstrapConfig()

		bootstrapConfig, err := hocon.ParseString(instance.Spec.BootstrapConfig)
		assert.Nil(t, err)
		assert.Equal(t, "\"0.0.0.0:11883\"", bootstrapConfig.GetString("listeners.tcp.default.bind"))
		assert.Equal(t, "1024000", bootstrapConfig.GetString("listeners.tcp.default.max_connections"))
	})

	t.Run("wrong bootstrap config", func(t *testing.T) {
		instance := &EMQX{
			Spec: EMQXSpec{
				BootstrapConfig: `hello world`,
			},
		}
		instance.defaultBootstrapConfig()
		assert.Equal(t, `hello world`, instance.Spec.BootstrapConfig)
	})
}

func TestDefaultReplicas(t *testing.T) {
	instance := &EMQX{}
	instance.defaultReplicas()
	assert.Equal(t, int32(2), *instance.Spec.CoreTemplate.Spec.Replicas)
	assert.Equal(t, int32(3), *instance.Spec.ReplicantTemplate.Spec.Replicas)
}

func TestDefaultDashboardServiceTemplate(t *testing.T) {
	t.Run("failed to get dashboard listeners", func(t *testing.T) {
		instance := &EMQX{}
		instance.defaultDashboardServiceTemplate()
		assert.Equal(t, int32(18083), instance.Spec.DashboardServiceTemplate.Spec.Ports[0].Port)
	})

	t.Run("set dashboard listeners", func(t *testing.T) {
		instance := &EMQX{
			Spec: EMQXSpec{
				BootstrapConfig: `dashboard.listeners.http.bind = 18084`,
			},
		}
		instance.defaultDashboardServiceTemplate()
		assert.Equal(t, int32(18084), instance.Spec.DashboardServiceTemplate.Spec.Ports[0].Port)
	})

	t.Run("check service selector", func(t *testing.T) {
		instance := &EMQX{
			Spec: EMQXSpec{
				CoreTemplate: EMQXCoreTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
		}
		instance.defaultDashboardServiceTemplate()
		assert.Equal(t, map[string]string{
			"foo": "bar",
		}, instance.Spec.DashboardServiceTemplate.Spec.Selector)
	})

}

func TestDefaultProbeForCoreNode(t *testing.T) {
	t.Run("failed to get dashboard listeners", func(t *testing.T) {
		instance := &EMQX{}
		instance.defaultProbe()

		expectReadinessProbe := &corev1.Probe{
			InitialDelaySeconds: 10,
			PeriodSeconds:       5,
			FailureThreshold:    12,
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/status",
					Port: intstr.FromInt(18083),
				},
			},
		}

		expectLivenessProbe := &corev1.Probe{
			InitialDelaySeconds: 60,
			PeriodSeconds:       30,
			FailureThreshold:    3,
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/status",
					Port: intstr.FromInt(18083),
				},
			},
		}

		assert.Equal(t, expectReadinessProbe, instance.Spec.CoreTemplate.Spec.ReadinessProbe)
		assert.Equal(t, expectLivenessProbe, instance.Spec.CoreTemplate.Spec.LivenessProbe)
	})

	t.Run("set dashboard listeners", func(t *testing.T) {
		instance := &EMQX{
			Spec: EMQXSpec{
				BootstrapConfig: `dashboard.listeners.http.bind = 18084`,
			},
		}
		instance.defaultProbe()

		expectReadinessProbe := &corev1.Probe{
			InitialDelaySeconds: 10,
			PeriodSeconds:       5,
			FailureThreshold:    12,
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/status",
					Port: intstr.FromInt(18084),
				},
			},
		}

		expectLivenessProbe := &corev1.Probe{
			InitialDelaySeconds: 60,
			PeriodSeconds:       30,
			FailureThreshold:    3,
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/status",
					Port: intstr.FromInt(18084),
				},
			},
		}

		assert.Equal(t, expectReadinessProbe, instance.Spec.CoreTemplate.Spec.ReadinessProbe)
		assert.Equal(t, expectLivenessProbe, instance.Spec.CoreTemplate.Spec.LivenessProbe)
	})
}

func TestDefaultAnnotations(t *testing.T) {
	instance := &EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"foo": "bar",
			},
		},
		Spec: EMQXSpec{
			CoreTemplate: EMQXCoreTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"core": "test",
					},
				},
			},
			ReplicantTemplate: EMQXReplicantTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"replicant": "test",
					},
				},
			},
			DashboardServiceTemplate: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"dashboard": "test",
					},
				},
			},
			ListenersServiceTemplate: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"listeners": "test",
					},
				},
			},
		},
	}
	instance.defaultAnnotations()

	assert.Equal(t, map[string]string{
		"foo":  "bar",
		"core": "test",
	}, instance.Spec.CoreTemplate.Annotations)
	assert.Equal(t, map[string]string{
		"foo":       "bar",
		"replicant": "test",
	}, instance.Spec.ReplicantTemplate.Annotations)
	assert.Equal(t, map[string]string{
		"foo":       "bar",
		"dashboard": "test",
	}, instance.Spec.DashboardServiceTemplate.Annotations)
	assert.Equal(t, map[string]string{
		"foo":       "bar",
		"listeners": "test",
	}, instance.Spec.ListenersServiceTemplate.Annotations)
}

func TestDefaultSecurityContext(t *testing.T) {
	instance := &EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "webhook-test",
			Namespace: "default",
		},
	}
	instance.defaultSecurityContext()

	assert.Equal(t, corev1.PodSecurityContext{
		RunAsUser:           pointer.Int64(1000),
		RunAsGroup:          pointer.Int64(1000),
		FSGroup:             pointer.Int64(1000),
		FSGroupChangePolicy: (*corev1.PodFSGroupChangePolicy)(pointer.String("Always")),
		SupplementalGroups:  []int64{1000},
	}, *instance.Spec.CoreTemplate.Spec.PodSecurityContext)

	assert.Equal(t, corev1.PodSecurityContext{
		RunAsUser:           pointer.Int64(1000),
		RunAsGroup:          pointer.Int64(1000),
		FSGroup:             pointer.Int64(1000),
		FSGroupChangePolicy: (*corev1.PodFSGroupChangePolicy)(pointer.String("Always")),
		SupplementalGroups:  []int64{1000},
	}, *instance.Spec.ReplicantTemplate.Spec.PodSecurityContext)
}
