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

	"github.com/rory-z/go-hocon"

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
	instance.Spec.CoreTemplate.Spec.Replicas = pointer.Int32(1)
	assert.Error(t, instance.ValidateCreate(), "the number of EMQX core nodes must be greater than 1")

	instance.Spec.CoreTemplate.Spec.Replicas = pointer.Int32(5)
	assert.Error(t, instance.ValidateCreate(), "the number of EMQX core nodes must be less than or equal to 4")

	instance.Spec.CoreTemplate.Spec.Replicas = pointer.Int32(2)
	assert.Nil(t, instance.ValidateCreate())

	instance.Spec.BootstrapConfig = "fake"
	assert.Error(t, instance.ValidateCreate(), "failed to parse bootstrap config")

	instance.Spec.BootstrapConfig = "foo = bar"
	assert.Nil(t, instance.ValidateCreate())

	instance.Spec.BootstrapConfig = `sql = "SELECT * FROM "t/#""`
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
			BootstrapConfig: `{a = 1, b = { c = 2, d = 3}}`,
		},
	}
	instance.Spec.CoreTemplate.Spec.Replicas = pointer.Int32(2)

	t.Run("should return error if core nodes is less then 2", func(t *testing.T) {
		newIns := instance.DeepCopy()
		newIns.Spec.CoreTemplate.Spec.Replicas = pointer.Int32(1)
		assert.Error(t, newIns.ValidateUpdate(instance), "the number of EMQX core nodes must be greater than 1")
	})

	t.Run("should return error if core nodes is greater then 4", func(t *testing.T) {
		newIns := instance.DeepCopy()
		newIns.Spec.CoreTemplate.Spec.Replicas = pointer.Int32(5)
		assert.Error(t, newIns.ValidateUpdate(instance), "the number of EMQX core nodes must be less than or equal to 4")
	})

	t.Run("should return error if bootstrap config is invalid", func(t *testing.T) {
		newIns := instance.DeepCopy()
		newIns.Spec.BootstrapConfig = "hello world"
		assert.Error(t, newIns.ValidateUpdate(instance), "failed to parse bootstrap config")
	})

	t.Run("should return error if bootstrap APIKeys is changed", func(t *testing.T) {
		newIns := instance.DeepCopy()
		newIns.Spec.BootstrapAPIKeys = []BootstrapAPIKey{{
			Key:    "test",
			Secret: "test",
		}}
		assert.Error(t, newIns.ValidateUpdate(instance), "bootstrap APIKeys cannot be updated")
	})

	t.Run("should return error if bootstrap config is changed", func(t *testing.T) {
		newIns := instance.DeepCopy()
		newIns.Spec.BootstrapConfig = "foo = bar"
		assert.Error(t, newIns.ValidateUpdate(instance), "bootstrap config cannot be updated")
	})

	t.Run("check bootstrap config is map", func(t *testing.T) {
		newIns := instance.DeepCopy()
		newIns.Spec.BootstrapConfig = `{b = { d = 3, c = 2 }, a = 1}`
		assert.Nil(t, newIns.ValidateUpdate(instance))
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
	instance.Spec.ReplicantTemplate = &EMQXReplicantTemplate{}
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
	instance.Spec.ReplicantTemplate = &EMQXReplicantTemplate{}
	instance.defaultLabels()

	assert.Equal(t, map[string]string{
		ManagerByLabelKey:    "emqx-operator",
		InstanceNameLabelKey: "webhook-test",
	}, instance.Labels)

	assert.Equal(t, map[string]string{
		ManagerByLabelKey:    "emqx-operator",
		InstanceNameLabelKey: "webhook-test",
	}, instance.Spec.DashboardServiceTemplate.Labels)

	assert.Equal(t, map[string]string{
		ManagerByLabelKey:    "emqx-operator",
		InstanceNameLabelKey: "webhook-test",
	}, instance.Spec.ListenersServiceTemplate.Labels)

	assert.Equal(t, map[string]string{
		ManagerByLabelKey:    "emqx-operator",
		InstanceNameLabelKey: "webhook-test",
		DBRoleLabelKey:       "core",
	}, instance.Spec.CoreTemplate.Labels)

	assert.Equal(t, map[string]string{
		ManagerByLabelKey:    "emqx-operator",
		InstanceNameLabelKey: "webhook-test",
		DBRoleLabelKey:       "replicant",
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
		assert.Equal(t, "\"0.0.0.0:8883\"", bootstrapConfig.GetString("listeners.ssl.default.bind"))
		assert.Equal(t, "\"0.0.0.0:8083\"", bootstrapConfig.GetString("listeners.ws.default.bind"))
		assert.Equal(t, "\"0.0.0.0:8084\"", bootstrapConfig.GetString("listeners.wss.default.bind"))
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

func TestDefaultContainerPort(t *testing.T) {
	instance := &EMQX{}
	t.Run("set default container port to core template", func(t *testing.T) {
		instance.defaultContainerPort()
		assert.Equal(t, len(instance.Spec.CoreTemplate.Spec.Ports), 1)
		defaultPort := instance.Spec.CoreTemplate.Spec.Ports[0]
		assert.Equal(t, int32(18083), defaultPort.ContainerPort)
		assert.Equal(t, "dashboard-http", defaultPort.Name)
		assert.Equal(t, corev1.ProtocolTCP, defaultPort.Protocol)
	})

	t.Run("set default container port to replica template", func(t *testing.T) {
		instance.Spec.ReplicantTemplate = &EMQXReplicantTemplate{}
		instance.defaultContainerPort()

		assert.Equal(t, len(instance.Spec.ReplicantTemplate.Spec.Ports), 1)
		defaultPort := instance.Spec.ReplicantTemplate.Spec.Ports[0]
		assert.Equal(t, int32(18083), defaultPort.ContainerPort)
		assert.Equal(t, "dashboard-http", defaultPort.Name)
		assert.Equal(t, corev1.ProtocolTCP, defaultPort.Protocol)
	})

	t.Run("merge container port by same name", func(t *testing.T) {
		instance.Spec.CoreTemplate.Spec.Ports = []corev1.ContainerPort{
			{
				Name:          "dashboard-http",
				ContainerPort: 18084,
			},
			{
				Name:          "other-port",
				ContainerPort: 1883,
			},
		}
		instance.defaultContainerPort()
		assert.Equal(t, len(instance.Spec.CoreTemplate.Spec.Ports), 2)
		index := -1
		ports := instance.Spec.CoreTemplate.Spec.Ports
		for index = range ports {
			if ports[index].Name == "dashboard-http" {
				break
			}
		}
		assert.NotEqual(t, index, -1, "missing container port named as dashboard-http")
		assert.NotEqual(t, index, len(ports), "missing container port named as dashboard-http")
		assert.Equal(t, ports[index].ContainerPort, int32(18084))
	})

	t.Run("merge container port by same port", func(t *testing.T) {
		instance.Spec.CoreTemplate.Spec.Ports = []corev1.ContainerPort{
			{
				Name:          "user-defined-dashboard-http",
				ContainerPort: 18083,
			},
		}
		instance.defaultContainerPort()
		assert.Equal(t, len(instance.Spec.CoreTemplate.Spec.Ports), 1)
		port := instance.Spec.CoreTemplate.Spec.Ports[0]
		assert.Equal(t, port.Name, "user-defined-dashboard-http")
		assert.Equal(t, port.ContainerPort, int32(18083))
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
			CoreTemplate: EMQXCoreTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"core": "test",
					},
				},
			},
			ReplicantTemplate: &EMQXReplicantTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"replicant": "test",
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
