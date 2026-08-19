/*
Copyright 2026.

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

package controller

import (
	"testing"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	"github.com/emqx/emqx-operator/internal/controller/config"
	"github.com/emqx/emqx-operator/test/util"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAttachRestartConfigTargetHash(t *testing.T) {
	t.Run("uses desired configuration instead of applied hash", func(t *testing.T) {
		instance := &crd.EMQX{
			ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				crd.AnnotationRestartConfigHash: "previously-applied",
			}},
			Spec: crd.EMQXSpec{Config: crd.Config{Roots: crd.ConfigRoots{
				"dashboard": apiextv1.JSON{Raw: util.FromYAMLString(`
					listeners:
					  http:
					    bind: 28083
				`)},
			}}},
		}
		template := &corev1.PodTemplateSpec{}

		attachTemplateConfigTargetHash(instance, template)

		_, restartRoots := config.SplitRoots(instance.Spec.Config.Roots)
		assert.Equal(t, configHash(restartRoots),
			template.Annotations[crd.AnnotationRestartConfigHash])
		assert.NotEqual(t, "previously-applied",
			template.Annotations[crd.AnnotationRestartConfigHash])
	})

	t.Run("removes target annotation when restart roots disappear", func(t *testing.T) {
		instance := &crd.EMQX{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
			crd.AnnotationRestartConfigHash: "previously-applied",
		}}}
		template := &corev1.PodTemplateSpec{}

		attachTemplateConfigTargetHash(instance, template)

		assert.NotContains(t, template.Annotations, crd.AnnotationRestartConfigHash)
	})

	t.Run("does not track configuration before any restart roots", func(t *testing.T) {
		template := &corev1.PodTemplateSpec{}

		attachTemplateConfigTargetHash(&crd.EMQX{}, template)

		assert.NotContains(t, template.Annotations, crd.AnnotationRestartConfigHash)
	})
}

func TestRestartConfigRemovalApplied(t *testing.T) {
	readyCore := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
			crd.LabelMriaRole: crd.RoleCore,
		}},
		Status: corev1.PodStatus{Conditions: []corev1.PodCondition{{
			Type:   corev1.ContainersReady,
			Status: corev1.ConditionTrue,
		}}},
	}
	state := &reconcileState{pods: []*corev1.Pod{readyCore}}

	assert.True(t, restartConfigApplied(state, &crd.EMQX{}, ""))

	readyCore.Annotations = map[string]string{
		crd.AnnotationRestartConfigHash: "previously-applied",
	}
	assert.False(t, restartConfigApplied(state, &crd.EMQX{}, ""))
}
