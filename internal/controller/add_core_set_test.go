package controller

import (
	"testing"

	crdv2 "github.com/emqx/emqx-operator/api/v2"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestGetNewStatefulSet(t *testing.T) {
	instance := &crdv2.EMQX{
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
		Spec: crdv2.EMQXSpec{
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

	t.Run("check metadata", func(t *testing.T) {
		emqx := instance.DeepCopy()
		conf, _ := config.EMQXConfigWithDefaults(emqx.Spec.Config.Data)
		got := newStatefulSet(emqx, conf)

		assert.Equal(t, emqx.Spec.CoreTemplate.Annotations, got.Annotations)
		assert.Equal(t, "core-label-value", got.Labels["core-label-key"])
		assert.Equal(t, "emqx", got.Labels[crdv2.LabelInstance])
		assert.Equal(t, "emqx-operator", got.Labels[crdv2.LabelManagedBy])
		assert.Equal(t, "core", got.Labels[crdv2.LabelDBRole])
		// Single StatefulSet: name is deterministic, no hash suffix.
		assert.Equal(t, "emqx-core", got.Name)
		assert.Equal(t, emqx.Namespace, got.Namespace)
	})

	t.Run("check selector and pod metadata", func(t *testing.T) {
		emqx := instance.DeepCopy()
		conf, _ := config.EMQXConfigWithDefaults(emqx.Spec.Config.Data)
		got := newStatefulSet(emqx, conf)
		assert.Equal(t, emqx.Spec.CoreTemplate.ObjectMeta.Annotations, got.Spec.Template.Annotations)
		assert.EqualValues(t, map[string]string{
			crdv2.LabelInstance:  "emqx",
			crdv2.LabelManagedBy: "emqx-operator",
			crdv2.LabelDBRole:    "core",
			"core-label-key":     "core-label-value",
		}, got.Spec.Template.Labels)

		assert.EqualValues(t, map[string]string{
			crdv2.LabelInstance:  "emqx",
			crdv2.LabelManagedBy: "emqx-operator",
			crdv2.LabelDBRole:    "core",
			"core-label-key":     "core-label-value",
		}, got.Spec.Selector.MatchLabels)
	})

	t.Run("check update strategy is OnDelete", func(t *testing.T) {
		emqx := instance.DeepCopy()
		conf, _ := config.EMQXConfigWithDefaults(emqx.Spec.Config.Data)
		got := newStatefulSet(emqx, conf)
		assert.Equal(t, appsv1.OnDeleteStatefulSetStrategyType, got.Spec.UpdateStrategy.Type)
		assert.EqualValues(t, int32(0), got.Spec.MinReadySeconds)
	})

	t.Run("check PVC retention policy deletes on scale-down and sts deletion", func(t *testing.T) {
		emqx := instance.DeepCopy()
		conf, _ := config.EMQXConfigWithDefaults(emqx.Spec.Config.Data)
		got := newStatefulSet(emqx, conf)
		assert.NotNil(t, got.Spec.PersistentVolumeClaimRetentionPolicy)
		assert.Equal(t, appsv1.DeletePersistentVolumeClaimRetentionPolicyType, got.Spec.PersistentVolumeClaimRetentionPolicy.WhenScaled)
		assert.Equal(t, appsv1.DeletePersistentVolumeClaimRetentionPolicyType, got.Spec.PersistentVolumeClaimRetentionPolicy.WhenDeleted)
	})

	t.Run("check bootstrap API keys", func(t *testing.T) {
		emqx := instance.DeepCopy()
		conf, _ := config.EMQXConfigWithDefaults(emqx.Spec.Config.Data)
		rs := newStatefulSet(emqx, conf)
		got := []corev1.EnvVar{}
		for _, env := range rs.Spec.Template.Spec.Containers[0].Env {
			if env.Name == "EMQX_API_KEY__BOOTSTRAP_FILE" {
				got = append(got, env)
			}
		}
		assert.NotEmpty(t, got)
	})

	t.Run("check http port", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Spec.Config.Data = "dashboard.listeners.http.bind = 18083"
		conf, _ := config.EMQXConfigWithDefaults(emqx.Spec.Config.Data)
		got := newStatefulSet(emqx, conf)

		assert.Contains(t, got.Spec.Template.Spec.Containers[0].Ports,
			corev1.ContainerPort{
				Name:          "dashboard",
				Protocol:      corev1.ProtocolTCP,
				ContainerPort: 18083,
			},
		)
	})

	t.Run("check https port", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Spec.Config.Data = `
		dashboard.listeners.http.bind = 0
		dashboard.listeners.https.bind = 18084
		`
		conf, _ := config.EMQXConfigWithDefaults(emqx.Spec.Config.Data)
		got := newStatefulSet(emqx, conf)
		assert.Contains(t, got.Spec.Template.Spec.Containers[0].Ports,
			corev1.ContainerPort{
				Name:          "dashboard-https",
				Protocol:      corev1.ProtocolTCP,
				ContainerPort: 18084,
			},
		)
	})

	t.Run("check http and https port", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Spec.Config.Data = `
		dashboard.listeners.http.bind = 18083
		dashboard.listeners.https.bind = 18084
		`
		conf, _ := config.EMQXConfigWithDefaults(emqx.Spec.Config.Data)
		got := newStatefulSet(emqx, conf)
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
	})

	t.Run("check default volume claim templates", func(t *testing.T) {
		emqx := instance.DeepCopy()

		fs := corev1.PersistentVolumeFilesystem
		got := generateStatefulSet(emqx)
		assert.Equal(t, []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "emqx-core-data",
					Namespace: "emqx",
					Labels: map[string]string{
						crdv2.LabelDBRole:    "core",
						crdv2.LabelInstance:  "emqx",
						crdv2.LabelManagedBy: "emqx-operator",
						"core-label-key":     "core-label-value",
					},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("500Mi"),
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

	t.Run("check explicit volume claim templates", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Spec.CoreTemplate.Spec.PersistentVolumeClaimSpec = corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
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
						crdv2.LabelDBRole:    "core",
						crdv2.LabelInstance:  "emqx",
						crdv2.LabelManagedBy: "emqx-operator",
						"core-label-key":     "core-label-value",
					},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
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
