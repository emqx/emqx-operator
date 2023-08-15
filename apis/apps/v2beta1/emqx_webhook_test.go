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

package v2beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	instance.Spec.Config.Data = "fake"
	assert.Error(t, instance.ValidateCreate(), "failed to parse configuration")

	instance.Spec.Config.Data = "foo = bar"
	assert.Nil(t, instance.ValidateCreate())

	instance.Spec.Config.Data = `sql = "SELECT * FROM "t/#""`
	assert.Nil(t, instance.ValidateCreate())
}

func TestValidateUpdate(t *testing.T) {
	instance := &EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "webhook-test",
			Namespace: "default",
		},
		Spec: EMQXSpec{
			Image: "emqx:latest",
			Config: Config{
				Data: `{a = 1, b = { c = 2, d = 3}}`,
			},
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

	t.Run("should return error if configuration is invalid", func(t *testing.T) {
		newIns := instance.DeepCopy()
		newIns.Spec.Config.Data = "hello world"
		assert.Error(t, newIns.ValidateUpdate(instance), "failed to parse configuration")
	})

	t.Run("should return error if bootstrap APIKeys is changed", func(t *testing.T) {
		newIns := instance.DeepCopy()
		newIns.Spec.BootstrapAPIKeys = []BootstrapAPIKey{{
			Key:    "test",
			Secret: "test",
		}}
		assert.Error(t, newIns.ValidateUpdate(instance), "bootstrap APIKeys cannot be updated")
	})

	t.Run("check configuration is map", func(t *testing.T) {
		newIns := instance.DeepCopy()
		newIns.Spec.Config.Data = `{b = { d = 3, c = 2 }, a = 1}`
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
		LabelsManagedByKey: "emqx-operator",
		LabelsInstanceKey:  "webhook-test",
	}, instance.Labels)

	assert.Equal(t, map[string]string{
		LabelsManagedByKey: "emqx-operator",
		LabelsInstanceKey:  "webhook-test",
	}, instance.Spec.DashboardServiceTemplate.Labels)

	assert.Equal(t, map[string]string{
		LabelsManagedByKey: "emqx-operator",
		LabelsInstanceKey:  "webhook-test",
	}, instance.Spec.ListenersServiceTemplate.Labels)

	assert.Equal(t, map[string]string{
		LabelsManagedByKey: "emqx-operator",
		LabelsInstanceKey:  "webhook-test",
		LabelsComponentKey: "core",
	}, instance.Spec.CoreTemplate.Labels)

	assert.Equal(t, map[string]string{
		LabelsManagedByKey: "emqx-operator",
		LabelsInstanceKey:  "webhook-test",
		LabelsComponentKey: "replicant",
	}, instance.Spec.ReplicantTemplate.Labels)
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
