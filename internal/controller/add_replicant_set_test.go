package controller

import (
	"testing"

	crdv2 "github.com/emqx/emqx-operator/api/v2"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestGetNewReplicaSet(t *testing.T) {
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
	instance.Spec.ReplicantTemplate = &crdv2.EMQXReplicantTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"repl-label-key": "repl-label-value",
			},
			Annotations: map[string]string{
				"repl-annotation-key": "repl-annotation-value",
			},
		},
		Spec: crdv2.EMQXReplicantTemplateSpec{
			Replicas: ptr.To(int32(3)),
		},
	}
	instance.Status.ReplicantNodesStatus = crdv2.ReplicantNodesStatus{
		CollisionCount: ptr.To(int32(0)),
	}

	t.Run("check metadata", func(t *testing.T) {
		emqx := instance.DeepCopy()
		conf, _ := config.EMQXConfigWithDefaults(emqx.Spec.Config.Data)
		got := newReplicaSet(emqx, conf)

		assert.Equal(t, emqx.Spec.ReplicantTemplate.Annotations, got.Annotations)
		assert.Equal(t, "repl-label-value", got.Labels["repl-label-key"])
		assert.Equal(t, "emqx", got.Labels[crdv2.LabelInstance])
		assert.Equal(t, "emqx-operator", got.Labels[crdv2.LabelManagedBy])
		assert.Equal(t, "replicant", got.Labels[crdv2.LabelDBRole])
		assert.Equal(t, "emqx-replicant-"+got.Labels[crdv2.LabelPodTemplateHash], got.Name)
		assert.Equal(t, emqx.Namespace, got.Namespace)
		assert.EqualValues(t, int32(0), got.Spec.MinReadySeconds)
	})

	t.Run("check selector and pod metadata", func(t *testing.T) {
		emqx := instance.DeepCopy()
		conf, _ := config.EMQXConfigWithDefaults(emqx.Spec.Config.Data)
		got := newReplicaSet(emqx, conf)

		assert.Equal(t, emqx.Spec.ReplicantTemplate.ObjectMeta.Annotations, got.Spec.Template.Annotations)
		assert.EqualValues(t, map[string]string{
			crdv2.LabelInstance:        "emqx",
			crdv2.LabelManagedBy:       "emqx-operator",
			crdv2.LabelDBRole:          "replicant",
			crdv2.LabelPodTemplateHash: got.Labels[crdv2.LabelPodTemplateHash],
			"repl-label-key":           "repl-label-value",
		}, got.Spec.Template.Labels)

		assert.EqualValues(t, map[string]string{
			crdv2.LabelInstance:        "emqx",
			crdv2.LabelManagedBy:       "emqx-operator",
			crdv2.LabelDBRole:          "replicant",
			crdv2.LabelPodTemplateHash: got.Labels[crdv2.LabelPodTemplateHash],
			"repl-label-key":           "repl-label-value",
		}, got.Spec.Selector.MatchLabels)
	})

	t.Run("check no bootstrap API keys", func(t *testing.T) {
		emqx := instance.DeepCopy()
		conf, _ := config.EMQXConfigWithDefaults(emqx.Spec.Config.Data)
		rs := newReplicaSet(emqx, conf)
		got := []corev1.EnvVar{}
		for _, env := range rs.Spec.Template.Spec.Containers[0].Env {
			if env.Name == "EMQX_API_KEY__BOOTSTRAP_FILE" {
				got = append(got, env)
			}
		}
		assert.Empty(t, got)
	})

	t.Run("check http port", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Spec.Config.Data = "dashboard.listeners.http.bind = 18083"
		conf, _ := config.EMQXConfigWithDefaults(emqx.Spec.Config.Data)
		got := newReplicaSet(emqx, conf)
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
		got := newReplicaSet(emqx, conf)
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
		got := newReplicaSet(emqx, conf)
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
}
