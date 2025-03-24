package v2beta1

import (
	"testing"

	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	config "github.com/emqx/emqx-operator/controllers/apps/v2beta1/config"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestGetNewReplicaSet(t *testing.T) {
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
	instance.Spec.ReplicantTemplate = &appsv2beta1.EMQXReplicantTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"repl-label-key": "repl-label-value",
			},
			Annotations: map[string]string{
				"repl-annotation-key": "repl-annotation-value",
			},
		},
		Spec: appsv2beta1.EMQXReplicantTemplateSpec{
			Replicas: ptr.To(int32(3)),
		},
	}
	instance.Status.ReplicantNodesStatus = appsv2beta1.EMQXNodesStatus{
		CollisionCount: ptr.To(int32(0)),
	}

	t.Run("check metadata", func(t *testing.T) {
		emqx := instance.DeepCopy()
		conf, _ := config.EMQXConf(config.MergeDefaults(emqx.Spec.Config.Data))
		got := getNewReplicaSet(emqx, conf)

		assert.Equal(t, emqx.Spec.ReplicantTemplate.Annotations, got.Annotations)
		assert.Equal(t, "repl-label-value", got.Labels["repl-label-key"])
		assert.Equal(t, "emqx", got.Labels[appsv2beta1.LabelsInstanceKey])
		assert.Equal(t, "emqx-operator", got.Labels[appsv2beta1.LabelsManagedByKey])
		assert.Equal(t, "replicant", got.Labels[appsv2beta1.LabelsDBRoleKey])
		assert.Equal(t, "emqx-replicant-"+got.Labels[appsv2beta1.LabelsPodTemplateHashKey], got.Name)
		assert.Equal(t, emqx.Namespace, got.Namespace)
	})

	t.Run("check selector and pod metadata", func(t *testing.T) {
		emqx := instance.DeepCopy()
		conf, _ := config.EMQXConf(config.MergeDefaults(emqx.Spec.Config.Data))
		got := getNewReplicaSet(emqx, conf)
		assert.Equal(t, emqx.Spec.ReplicantTemplate.ObjectMeta.Annotations, got.Spec.Template.Annotations)
		assert.EqualValues(t, map[string]string{
			appsv2beta1.LabelsInstanceKey:        "emqx",
			appsv2beta1.LabelsManagedByKey:       "emqx-operator",
			appsv2beta1.LabelsDBRoleKey:          "replicant",
			appsv2beta1.LabelsPodTemplateHashKey: got.Labels[appsv2beta1.LabelsPodTemplateHashKey],
			"repl-label-key":                     "repl-label-value",
		}, got.Spec.Template.Labels)

		assert.EqualValues(t, map[string]string{
			appsv2beta1.LabelsInstanceKey:        "emqx",
			appsv2beta1.LabelsManagedByKey:       "emqx-operator",
			appsv2beta1.LabelsDBRoleKey:          "replicant",
			appsv2beta1.LabelsPodTemplateHashKey: got.Labels[appsv2beta1.LabelsPodTemplateHashKey],
			"repl-label-key":                     "repl-label-value",
		}, got.Spec.Selector.MatchLabels)
	})

	t.Run("check http port", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Spec.Config.Data = "dashboard.listeners.http.bind = 18083"
		conf, _ := config.EMQXConf(config.MergeDefaults(emqx.Spec.Config.Data))
		got := getNewReplicaSet(emqx, conf)

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
		emqx.Spec.Config.Data = `
		dashboard.listeners.http.bind = 0
		dashboard.listeners.https.bind = 18084
		`
		conf, _ := config.EMQXConf(config.MergeDefaults(emqx.Spec.Config.Data))
		got := getNewReplicaSet(emqx, conf)

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
		emqx.Spec.Config.Data = `
		dashboard.listeners.http.bind = 18083
		dashboard.listeners.https.bind = 18084
		`
		conf, _ := config.EMQXConf(config.MergeDefaults(emqx.Spec.Config.Data))
		got := getNewReplicaSet(emqx, conf)

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
}
