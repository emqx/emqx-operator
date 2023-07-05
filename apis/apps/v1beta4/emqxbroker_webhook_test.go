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

package v1beta4

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

func TestBrokerDefault(t *testing.T) {
	instance := &EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "webhook-test",
			Namespace: "default",
			Labels: map[string]string{
				"foo": "bar",
			},
			Annotations: map[string]string{
				"foo": "bar",
				"kubectl.kubernetes.io/last-applied-configuration": "fake",
			},
		},
		Spec: EmqxBrokerSpec{
			Template: EmqxTemplate{
				Spec: EmqxTemplateSpec{
					EmqxContainer: EmqxContainer{
						Image: EmqxImage{
							Version: "4.4.14",
						},
					},
				},
			},
		},
	}
	instance.Default()

	t.Run("default labels", func(t *testing.T) {
		assert.Equal(t, map[string]string{
			"foo":                     "bar",
			"apps.emqx.io/managed-by": "emqx-operator",
			"apps.emqx.io/instance":   "webhook-test",
		}, instance.Labels)

		assert.Equal(t, map[string]string{
			"foo":                     "bar",
			"apps.emqx.io/managed-by": "emqx-operator",
			"apps.emqx.io/instance":   "webhook-test",
		}, instance.Spec.Template.Labels)
	})

	t.Run("default annotations", func(t *testing.T) {
		assert.Equal(t, map[string]string{
			"foo": "bar",
			"kubectl.kubernetes.io/last-applied-configuration": "fake",
		}, instance.Annotations)

		assert.Equal(t, map[string]string{
			"foo": "bar",
		}, instance.Spec.Template.Annotations)
	})

	t.Run("default emqx image", func(t *testing.T) {
		assert.Equal(t, "emqx/emqx", instance.Spec.Template.Spec.EmqxContainer.Image.Repository)
	})

	t.Run("default emqx acl", func(t *testing.T) {
		assert.ElementsMatch(t, []string{
			`{allow, {user, "dashboard"}, subscribe, ["$SYS/#"]}.`,
			`{allow, {ipaddr, "127.0.0.1"}, pubsub, ["$SYS/#", "#"]}.`,
			`{deny, all, subscribe, ["$SYS/#", {eq, "#"}]}.`,
			`{allow, all}.`,
		}, instance.Spec.Template.Spec.EmqxContainer.EmqxACL)
	})

	t.Run("default emqx config", func(t *testing.T) {
		assert.Equal(t, map[string]string{
			"name":                  "webhook-test",
			"log.to":                "console",
			"cluster.discovery":     "dns",
			"cluster.dns.type":      "srv",
			"cluster.dns.app":       "webhook-test",
			"cluster.dns.name":      "webhook-test-headless.default.svc.cluster.local",
			"listener.tcp.internal": "",
		}, instance.Spec.Template.Spec.EmqxContainer.EmqxConfig)
	})

	t.Run("default service template", func(t *testing.T) {
		assert.Equal(t, ServiceTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "webhook-test",
				Namespace: "default",
				Labels: map[string]string{
					"foo":                     "bar",
					"apps.emqx.io/managed-by": "emqx-operator",
					"apps.emqx.io/instance":   "webhook-test",
				},
				Annotations: map[string]string{
					"foo": "bar",
				},
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"foo":                     "bar",
					"apps.emqx.io/managed-by": "emqx-operator",
					"apps.emqx.io/instance":   "webhook-test",
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
		}, instance.Spec.ServiceTemplate)
	})

	t.Run("default persistence", func(t *testing.T) {
		assert.Nil(t, instance.GetSpec().GetPersistent())

		instance.Spec.Persistent = &corev1.PersistentVolumeClaimTemplate{
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
			},
		}
		instance.Default()
		assert.Equal(t, metav1.ObjectMeta{
			Name:      "webhook-test-data",
			Namespace: "default",
			Labels: map[string]string{
				"foo":                     "bar",
				"apps.emqx.io/managed-by": "emqx-operator",
				"apps.emqx.io/instance":   "webhook-test",
			},
			Annotations: map[string]string{
				"foo": "bar",
			},
		}, instance.Spec.Persistent.ObjectMeta)
	})

	t.Run("default container port", func(t *testing.T) {
		assert.GreaterOrEqual(t, len(instance.Spec.Template.Spec.EmqxContainer.Ports), 1)
		defaultPort := instance.Spec.Template.Spec.EmqxContainer.Ports[0]
		assert.Equal(t, defaultPort.Name, "dashboard-http")
		assert.Equal(t, defaultPort.Protocol, corev1.ProtocolTCP)
		assert.Equal(t, defaultPort.ContainerPort, int32(18083))
	})

	t.Run("merge container port by same port", func(t *testing.T) {
		instance.Spec.Template.Spec.EmqxContainer.Ports = []corev1.ContainerPort{
			{
				Name:          "user-defined-dashboard-http",
				ContainerPort: 18083,
			},
		}
		instance.Default()

		assert.GreaterOrEqual(t, len(instance.Spec.Template.Spec.EmqxContainer.Ports), 1)
		defaultPort := instance.Spec.Template.Spec.EmqxContainer.Ports[0]
		assert.Equal(t, defaultPort.Name, "user-defined-dashboard-http")
		assert.Equal(t, defaultPort.ContainerPort, int32(18083))
	})

	t.Run("merge container port by same name", func(t *testing.T) {
		instance.Spec.Template.Spec.EmqxContainer.Ports = []corev1.ContainerPort{
			{
				Name:          "dashboard-http",
				ContainerPort: 18084,
			},
		}
		instance.Default()

		assert.GreaterOrEqual(t, len(instance.Spec.Template.Spec.EmqxContainer.Ports), 1)
		defaultPort := instance.Spec.Template.Spec.EmqxContainer.Ports[0]
		assert.Equal(t, defaultPort.Name, "dashboard-http")
		assert.Equal(t, defaultPort.ContainerPort, int32(18084))
	})
}

func TestBrokerValidateCreate(t *testing.T) {
	broker := &EmqxBroker{
		Spec: EmqxBrokerSpec{
			Template: EmqxTemplate{
				Spec: EmqxTemplateSpec{
					EmqxContainer: EmqxContainer{
						Image: EmqxImage{
							Version: "4.4.14",
						},
					},
				},
			},
		},
	}

	t.Run("valid image version", func(t *testing.T) {
		instance := broker.DeepCopy()

		instance.Spec.Template.Spec.EmqxContainer.Image.Version = "4.4.14"
		assert.NoError(t, instance.ValidateCreate())
		instance.Spec.Template.Spec.EmqxContainer.Image.Version = "4.4.14"
		assert.NoError(t, instance.ValidateCreate())
		instance.Spec.Template.Spec.EmqxContainer.Image.Version = "4.5.0"
		assert.NoError(t, instance.ValidateCreate())
		instance.Spec.Template.Spec.EmqxContainer.Image.Version = "4.10"
		assert.NoError(t, instance.ValidateCreate())
		instance.Spec.Template.Spec.EmqxContainer.Image.Version = "4.123456789"
		assert.NoError(t, instance.ValidateCreate())

		instance.Spec.Template.Spec.EmqxContainer.Image.Version = "latest"
		assert.ErrorContains(t, instance.ValidateCreate(), "image version can not be latest")
		instance.Spec.Template.Spec.EmqxContainer.Image.Version = ""
		assert.ErrorContains(t, instance.ValidateCreate(), "invalid image version")
		instance.Spec.Template.Spec.EmqxContainer.Image.Version = "fake"
		assert.ErrorContains(t, instance.ValidateCreate(), "invalid image version")
		instance.Spec.Template.Spec.EmqxContainer.Image.Version = "4.4.13"
		assert.ErrorContains(t, instance.ValidateCreate(), "please upgrade to 4.4.14 or later")
		instance.Spec.Template.Spec.EmqxContainer.Image.Version = "4.4"
		assert.ErrorContains(t, instance.ValidateCreate(), "please upgrade to 4.4.14 or later")
		instance.Spec.Template.Spec.EmqxContainer.Image.Version = "4.3"
		assert.ErrorContains(t, instance.ValidateCreate(), "please upgrade to 4.4.14 or later")
		instance.Spec.Template.Spec.EmqxContainer.Image.Version = "4"
		assert.ErrorContains(t, instance.ValidateCreate(), "please upgrade to 4.4.14 or later")
		instance.Spec.Template.Spec.EmqxContainer.Image.Version = "5.0.0"
		assert.ErrorContains(t, instance.ValidateCreate(), "please downgrade to 5.0.0 earlier")
	})
}

func TestBrokerValidateUpdate(t *testing.T) {
	broker := &EmqxBroker{
		Spec: EmqxBrokerSpec{
			Template: EmqxTemplate{
				Spec: EmqxTemplateSpec{
					EmqxContainer: EmqxContainer{
						Image: EmqxImage{
							Version: "4.4.14",
						},
						EmqxConfig: map[string]string{
							"name":                  "emqx",
							"cluster.discovery":     "dns",
							"cluster.dns.type":      "srv",
							"cluster.dns.app":       "emqx",
							"cluster.dns.name":      "emqx-headless.default.svc.cluster.local",
							"listener.tcp.internal": "0.0.0.0:1883",
						},
						BootstrapAPIKeys: []BootstrapAPIKey{
							{
								Key:    "test",
								Secret: "test",
							},
						},
					},
				},
			},
		},
	}

	t.Run("valid image version", func(t *testing.T) {
		old := broker.DeepCopy()
		newIns := broker.DeepCopy()

		newIns.Spec.Template.Spec.EmqxContainer.Image.Version = "4.4.14"
		assert.NoError(t, newIns.ValidateUpdate(old))
		newIns.Spec.Template.Spec.EmqxContainer.Image.Version = "4.4.14"
		assert.NoError(t, newIns.ValidateUpdate(old))
		newIns.Spec.Template.Spec.EmqxContainer.Image.Version = "4.5.0"
		assert.NoError(t, newIns.ValidateUpdate(old))
		newIns.Spec.Template.Spec.EmqxContainer.Image.Version = "4.10"
		assert.NoError(t, newIns.ValidateUpdate(old))
		newIns.Spec.Template.Spec.EmqxContainer.Image.Version = "4.123456789"
		assert.NoError(t, newIns.ValidateUpdate(old))

		newIns.Spec.Template.Spec.EmqxContainer.Image.Version = "latest"
		assert.ErrorContains(t, newIns.ValidateUpdate(old), "image version can not be latest")
		newIns.Spec.Template.Spec.EmqxContainer.Image.Version = ""
		assert.ErrorContains(t, newIns.ValidateUpdate(old), "invalid image version")
		newIns.Spec.Template.Spec.EmqxContainer.Image.Version = "fake"
		assert.ErrorContains(t, newIns.ValidateUpdate(old), "invalid image version")
		newIns.Spec.Template.Spec.EmqxContainer.Image.Version = "4.4.13"
		assert.ErrorContains(t, newIns.ValidateUpdate(old), "please upgrade to 4.4.14 or later")
		newIns.Spec.Template.Spec.EmqxContainer.Image.Version = "4.4"
		assert.ErrorContains(t, newIns.ValidateUpdate(old), "please upgrade to 4.4.14 or later")
		newIns.Spec.Template.Spec.EmqxContainer.Image.Version = "4.3"
		assert.ErrorContains(t, newIns.ValidateUpdate(old), "please upgrade to 4.4.14 or later")
		newIns.Spec.Template.Spec.EmqxContainer.Image.Version = "4"
		assert.ErrorContains(t, newIns.ValidateUpdate(old), "please upgrade to 4.4.14 or later")
		newIns.Spec.Template.Spec.EmqxContainer.Image.Version = "5.0.0"
		assert.ErrorContains(t, newIns.ValidateUpdate(old), "please downgrade to 5.0.0 earlier")
	})

	t.Run("valid volume template can not update", func(t *testing.T) {
		old := broker.DeepCopy()
		newIns := broker.DeepCopy()

		assert.Nil(t, newIns.ValidateUpdate(old))

		old.Spec.Persistent = &corev1.PersistentVolumeClaimTemplate{
			Spec: corev1.PersistentVolumeClaimSpec{
				StorageClassName: pointer.String("fake"),
			},
		}
		assert.Error(t, newIns.ValidateUpdate(old))
	})

	t.Run("should return error if bootstrap APIKeys is changed", func(t *testing.T) {
		old := broker.DeepCopy()
		broker.Spec.Template.Spec.EmqxContainer.BootstrapAPIKeys = []BootstrapAPIKey{{
			Key:    "change_key",
			Secret: "test",
		}}
		assert.Error(t, broker.ValidateUpdate(old), "bootstrap APIKeys cannot be updated")
	})

	t.Run("valid emqxConfig can not update", func(t *testing.T) {
		old := broker.DeepCopy()
		newIns := broker.DeepCopy()

		assert.Nil(t, newIns.ValidateUpdate(old))

		newIns.Spec.Template.Spec.EmqxContainer.EmqxConfig["name"] = "emqx-test"
		assert.Error(t, newIns.ValidateUpdate(old))

		newIns.Spec.Template.Spec.EmqxContainer.EmqxConfig["name"] = "emqx"
		newIns.Spec.Template.Spec.EmqxContainer.EmqxConfig["cluster.dns.app"] = "emqx-test"
		assert.Error(t, newIns.ValidateUpdate(old))

		newIns.Spec.Template.Spec.EmqxContainer.EmqxConfig["name"] = "emqx"
		newIns.Spec.Template.Spec.EmqxContainer.EmqxConfig["cluster.dns.app"] = "emqx"
		newIns.Spec.Template.Spec.EmqxContainer.EmqxConfig["listener.tcp.internal"] = "0.0.0.0:1884"
		assert.Nil(t, newIns.ValidateUpdate(old))

		delete(newIns.Spec.Template.Spec.EmqxContainer.EmqxConfig, "name")
		delete(newIns.Spec.Template.Spec.EmqxContainer.EmqxConfig, "cluster.dns.app")
		newIns.Spec.Template.Spec.EmqxContainer.EmqxConfig["listener.tcp.internal"] = "0.0.0.0:1885"
		assert.Nil(t, newIns.ValidateUpdate(old))
	})
}
