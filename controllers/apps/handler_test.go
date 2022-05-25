package apps_test

import (
	"testing"

	"github.com/emqx/emqx-operator/controllers/apps"
	json "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestIgnoreOtherContainer(t *testing.T) {
	selectEmqxContainer := apps.IgnoreOtherContainers()

	currentObject := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "emqx",
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
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "emqx",
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

	modifiedObject.Spec.Template.Spec.Containers[0] = corev1.Container{Name: "emqx", Args: []string{"--fake"}}
	modified, _ = json.ConfigCompatibleWithStandardLibrary.Marshal(modifiedObject)

	current, modified, err = selectEmqxContainer(current, modified)
	if err != nil {
		t.Error(err)
	}
	assert.NotEqual(t, current, modified, "the current and modified byte sequence should be the not same")
}
