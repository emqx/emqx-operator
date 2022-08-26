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
	"testing"

	// "github.com/gurkankaymak/hocon"
	hocon "github.com/rory-z/go-hocon"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	assert.ErrorContains(t, instance.ValidateCreate(), "failed to parse bootstrap config")

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

	old := instance.DeepCopy()
	old.Spec.BootstrapConfig = "foo = bar"

	assert.EqualError(t, instance.ValidateUpdate(old), "bootstrap config cannot be updated")

	instance.Spec.BootstrapConfig = "foo = bar"
	assert.Nil(t, instance.ValidateUpdate(old))
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
}

func TestDefaultDashboardServiceTemplate(t *testing.T) {
	instance := &EMQX{
		Spec: EMQXSpec{
			BootstrapConfig: `dashboard.listeners.http.bind = 18083`,
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
	assert.Equal(t, int32(18083), instance.Spec.DashboardServiceTemplate.Spec.Ports[0].Port)
}
