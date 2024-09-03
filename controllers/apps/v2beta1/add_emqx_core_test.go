package v2beta1

import (
	"testing"

	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestGetNewStatefulSet(t *testing.T) {
	instance := &appsv2beta1.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
			Labels: map[string]string{
				"emqx-label-key": "emqx-label-value",
			},
			Annotations: map[string]string{
				"emqx-annotation-key": "emqx-annotation-value",
			},
		},
		Spec: appsv2beta1.EMQXSpec{
			Image:         "emqx/emqx:5.1",
			ClusterDomain: "cluster.local",
		},
	}
	instance.Spec.CoreTemplate.ObjectMeta = metav1.ObjectMeta{
		Labels: map[string]string{
			"core-label-key": "core-label-value",
		},
		Annotations: map[string]string{
			"core-annotation-key": "core-annotation-value",
		},
	}
	instance.Spec.CoreTemplate.Spec.Replicas = ptr.To(int32(3))
	instance.Status.CoreNodesStatus = &appsv2beta1.EMQXNodesStatus{
		CollisionCount: ptr.To(int32(0)),
	}

	t.Run("check metadata", func(t *testing.T) {
		emqx := instance.DeepCopy()
		got := getNewStatefulSet(emqx)

		assert.Equal(t, emqx.Spec.CoreTemplate.Annotations, got.Annotations)
		assert.Equal(t, "core-label-value", got.Labels["core-label-key"])
		assert.Equal(t, "emqx", got.Labels[appsv2beta1.LabelsInstanceKey])
		assert.Equal(t, "emqx-operator", got.Labels[appsv2beta1.LabelsManagedByKey])
		assert.Equal(t, "core", got.Labels[appsv2beta1.LabelsDBRoleKey])
		assert.Equal(t, "emqx-core-"+got.Labels[appsv2beta1.LabelsPodTemplateHashKey], got.Name)
		assert.Equal(t, emqx.Namespace, got.Namespace)
	})

	t.Run("check selector and pod metadata", func(t *testing.T) {
		emqx := instance.DeepCopy()
		got := getNewStatefulSet(emqx)
		assert.Equal(t, emqx.Spec.CoreTemplate.ObjectMeta.Annotations, got.Spec.Template.Annotations)
		assert.EqualValues(t, map[string]string{
			appsv2beta1.LabelsInstanceKey:        "emqx",
			appsv2beta1.LabelsManagedByKey:       "emqx-operator",
			appsv2beta1.LabelsDBRoleKey:          "core",
			appsv2beta1.LabelsPodTemplateHashKey: got.Labels[appsv2beta1.LabelsPodTemplateHashKey],
			"core-label-key":                     "core-label-value",
		}, got.Spec.Template.Labels)

		assert.EqualValues(t, map[string]string{
			appsv2beta1.LabelsInstanceKey:        "emqx",
			appsv2beta1.LabelsManagedByKey:       "emqx-operator",
			appsv2beta1.LabelsDBRoleKey:          "core",
			appsv2beta1.LabelsPodTemplateHashKey: got.Labels[appsv2beta1.LabelsPodTemplateHashKey],
			"core-label-key":                     "core-label-value",
		}, got.Spec.Selector.MatchLabels)
	})

	t.Run("check http port", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Spec.Config.Data = "dashboard.listeners.http.bind = 18083"
		got := getNewStatefulSet(emqx)

		assert.Contains(t, got.Spec.Template.Spec.Containers[0].Ports,
			corev1.ContainerPort{
				Name:          "dashboard",
				Protocol:      corev1.ProtocolTCP,
				ContainerPort: 18083,
			},
		)

		assert.Contains(t, got.Spec.Template.Spec.Containers[0].Env,
			corev1.EnvVar{
				Name:  "EMQX_DASHBOARD__LISTENERS__HTTP__BIND",
				Value: "18083",
			},
		)
	})

	t.Run("check https port", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Spec.Config.Data = "dashboard.listeners.http.bind = 0 \n dashboard.listeners.https.bind = 18084"
		got := getNewStatefulSet(emqx)

		assert.Contains(t, got.Spec.Template.Spec.Containers[0].Ports,
			corev1.ContainerPort{
				Name:          "dashboard-https",
				Protocol:      corev1.ProtocolTCP,
				ContainerPort: 18084,
			},
		)

		assert.Contains(t, got.Spec.Template.Spec.Containers[0].Env,
			corev1.EnvVar{
				Name:  "EMQX_DASHBOARD__LISTENERS__HTTPS__BIND",
				Value: "18084",
			},
		)
	})

	t.Run("check http and https port", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Spec.Config.Data = "dashboard.listeners.http.bind = 18083 \n dashboard.listeners.https.bind = 18084"
		got := getNewStatefulSet(emqx)

		assert.Contains(t, got.Spec.Template.Spec.Containers[0].Ports,
			corev1.ContainerPort{
				Name:          "dashboard",
				Protocol:      corev1.ProtocolTCP,
				ContainerPort: 18083,
			},
		)

		assert.Contains(t, got.Spec.Template.Spec.Containers[0].Ports,
			corev1.ContainerPort{
				Name:          "dashboard-https",
				Protocol:      corev1.ProtocolTCP,
				ContainerPort: 18084,
			},
		)

		assert.Contains(t, got.Spec.Template.Spec.Containers[0].Env,
			corev1.EnvVar{
				Name:  "EMQX_DASHBOARD__LISTENERS__HTTP__BIND",
				Value: "18083",
			},
		)

		assert.Contains(t, got.Spec.Template.Spec.Containers[0].Env,
			corev1.EnvVar{
				Name:  "EMQX_DASHBOARD__LISTENERS__HTTPS__BIND",
				Value: "18084",
			},
		)
	})

	t.Run("check sts volume claim templates", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Spec.CoreTemplate.Spec.VolumeClaimTemplates = corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("20Mi"),
				},
			},
		}

		fs := corev1.PersistentVolumeFilesystem
		got := generateStatefulSet(emqx)
		assert.Equal(t, []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "emqx-core-data",
					Namespace: "emqx",
					Labels: map[string]string{
						appsv2beta1.LabelsDBRoleKey:    "core",
						appsv2beta1.LabelsInstanceKey:  "emqx",
						appsv2beta1.LabelsManagedByKey: "emqx-operator",
						"core-label-key":               "core-label-value",
					},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("20Mi"),
						},
					},
					VolumeMode: &fs,
				},
			},
		}, got.Spec.VolumeClaimTemplates)
		assert.NotContains(t, got.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: "emqx-core-data",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	})
}
