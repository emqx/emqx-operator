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

	"github.com/banzaicloud/k8s-objectmatcher/patch"
	"github.com/emqx/emqx-operator/internal/handler"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIgnoreOtherContainerForSts(t *testing.T) {
	current := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "default",
		},
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
	assert.Nil(t, patch.DefaultAnnotator.SetLastAppliedAnnotation(current))

	modified := current.DeepCopy()
	modified.Spec.Template.Spec.Containers = append(modified.Spec.Template.Spec.Containers, corev1.Container{Name: "fake"})

	patchResult, err := patch.DefaultPatchMaker.Calculate(current, modified, handler.IgnoreOtherContainers())
	assert.Nil(t, err)
	assert.True(t, patchResult.IsEmpty())

	modified.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name: "emqx",
			Args: []string{"--fake"},
		},
		{
			Name: "reloader",
		},
	}

	patchResult, err = patch.DefaultPatchMaker.Calculate(current, modified, handler.IgnoreOtherContainers())
	assert.Nil(t, err)
	assert.False(t, patchResult.IsEmpty())

	modified.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name: "emqx",
		},
		{
			Name: "reloader",
			Args: []string{"--fake"},
		},
	}

	patchResult, err = patch.DefaultPatchMaker.Calculate(current, modified, handler.IgnoreOtherContainers())
	assert.Nil(t, err)
	assert.False(t, patchResult.IsEmpty())
}

func TestIgnoreOtherContainerForDeploy(t *testing.T) {
	current := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "default",
		},
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
	assert.Nil(t, patch.DefaultAnnotator.SetLastAppliedAnnotation(current))

	modified := current.DeepCopy()
	modified.Spec.Template.Spec.Containers = append(modified.Spec.Template.Spec.Containers, corev1.Container{Name: "fake"})

	patchResult, err := patch.DefaultPatchMaker.Calculate(current, modified, []patch.CalculateOption{
		handler.IgnoreOtherContainers(),
	}...)
	assert.Nil(t, err)
	assert.True(t, patchResult.IsEmpty())

	modified.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name: "emqx",
			Args: []string{"--fake"},
		},
		{
			Name: "reloader",
		},
	}

	patchResult, err = patch.DefaultPatchMaker.Calculate(current, modified, handler.IgnoreOtherContainers())
	assert.Nil(t, err)
	assert.False(t, patchResult.IsEmpty())

	modified.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name: "emqx",
		},
		{
			Name: "reloader",
			Args: []string{"--fake"},
		},
	}

	patchResult, err = patch.DefaultPatchMaker.Calculate(current, modified, handler.IgnoreOtherContainers())
	assert.Nil(t, err)
	assert.False(t, patchResult.IsEmpty())
}
