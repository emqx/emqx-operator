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

package handler_test

import (
	"testing"

	"github.com/emqx/emqx-operator/pkg/handler"
	json "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIgnoreOtherContainerForSts(t *testing.T) {
	selectEmqxContainer := handler.IgnoreOtherContainers()

	currentObject := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						handler.ManageContainersAnnotation: "emqx,reloader",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "emqx",
						},
						{
							Name: "reloader",
						},
					},
				},
			},
		},
	}
	current, _ := json.ConfigCompatibleWithStandardLibrary.Marshal(currentObject)

	modifiedObject := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						handler.ManageContainersAnnotation: "emqx,reloader",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "emqx",
						},
						{
							Name: "reloader",
						},
						{
							Name: "fake",
						},
					},
				},
			},
		},
	}
	modified, _ := json.ConfigCompatibleWithStandardLibrary.Marshal(modifiedObject)

	current, modified, err := selectEmqxContainer(current, modified)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, current, modified, "the current and modified byte sequence should be the same")

	modifiedObject.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name: "emqx",
			Args: []string{"--fake"},
		},
		{
			Name: "reloader",
		},
	}
	modified, _ = json.ConfigCompatibleWithStandardLibrary.Marshal(modifiedObject)

	current, modified, err = selectEmqxContainer(current, modified)
	if err != nil {
		t.Error(err)
	}
	assert.NotEqual(t, current, modified, "the current and modified byte sequence should be the not same")

	modifiedObject.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name: "emqx",
		},
		{
			Name: "reloader",
			Args: []string{"--fake"},
		},
	}
	modified, _ = json.ConfigCompatibleWithStandardLibrary.Marshal(modifiedObject)

	current, modified, err = selectEmqxContainer(current, modified)
	if err != nil {
		t.Error(err)
	}
	assert.NotEqual(t, current, modified, "the current and modified byte sequence should be the not same")
}

func TestIgnoreOtherContainerForDeploy(t *testing.T) {
	selectEmqxContainer := handler.IgnoreOtherContainers()

	currentObject := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						handler.ManageContainersAnnotation: "emqx,reloader",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "emqx",
						},
						{
							Name: "reloader",
						},
					},
				},
			},
		},
	}
	current, _ := json.ConfigCompatibleWithStandardLibrary.Marshal(currentObject)

	modifiedObject := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						handler.ManageContainersAnnotation: "emqx,reloader",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "emqx",
						},
						{
							Name: "reloader",
						},
						{
							Name: "fake",
						},
					},
				},
			},
		},
	}
	modified, _ := json.ConfigCompatibleWithStandardLibrary.Marshal(modifiedObject)

	current, modified, err := selectEmqxContainer(current, modified)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, current, modified, "the current and modified byte sequence should be the same")

	modifiedObject.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name: "emqx",
			Args: []string{"--fake"},
		},
		{
			Name: "reloader",
		},
	}
	modified, _ = json.ConfigCompatibleWithStandardLibrary.Marshal(modifiedObject)

	current, modified, err = selectEmqxContainer(current, modified)
	if err != nil {
		t.Error(err)
	}
	assert.NotEqual(t, current, modified, "the current and modified byte sequence should be the not same")

	modifiedObject.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name: "emqx",
		},
		{
			Name: "reloader",
			Args: []string{"--fake"},
		},
	}
	modified, _ = json.ConfigCompatibleWithStandardLibrary.Marshal(modifiedObject)

	current, modified, err = selectEmqxContainer(current, modified)
	if err != nil {
		t.Error(err)
	}
	assert.NotEqual(t, current, modified, "the current and modified byte sequence should be the not same")
}
