package controller

import (
	"testing"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	"github.com/emqx/emqx-operator/test/util"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestGetNewReplicaSet(t *testing.T) {
	instance := &crd.EMQX{
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
		Spec: crd.EMQXSpec{
			Image:         "emqx/emqx:5.1",
			ClusterDomain: "cluster.local",
		},
	}
	instance.Spec.ReplicantTemplate = crd.EMQXReplicantTemplate{
		TemplateObjectMeta: crd.TemplateObjectMeta{
			Labels: map[string]string{
				"repl-label-key": "repl-label-value",
			},
			Annotations: map[string]string{
				"repl-annotation-key": "repl-annotation-value",
			},
		},
		Spec: crd.EMQXReplicantTemplateSpec{
			Replicas: ptr.To(int32(3)),
		},
	}
	instance.Status.ReplicantNodesStatus = crd.ReplicantNodesStatus{
		CollisionCount: ptr.To(int32(0)),
	}

	t.Run("check metadata", func(t *testing.T) {
		emqx := instance.DeepCopy()
		got := newReplicaSet(emqx)

		assert.Equal(t, emqx.Spec.ReplicantTemplate.Annotations, got.Annotations)
		assert.Equal(t, "repl-label-value", got.Labels["repl-label-key"])
		assert.Equal(t, "emqx", got.Labels[crd.LabelInstance])
		assert.Equal(t, "emqx-operator", got.Labels[crd.LabelManagedBy])
		assert.Equal(t, "replicant", got.Labels[crd.LabelMriaRole])
		assert.Equal(t, "emqx-replicant-"+got.Labels[crd.LabelPodTemplateHash], got.Name)
		assert.Equal(t, emqx.Namespace, got.Namespace)
		assert.EqualValues(t, int32(0), got.Spec.MinReadySeconds)
	})

	t.Run("check selector and pod metadata", func(t *testing.T) {
		emqx := instance.DeepCopy()
		got := newReplicaSet(emqx)

		assert.Equal(t, emqx.Spec.ReplicantTemplate.Annotations, got.Spec.Template.Annotations)
		assert.EqualValues(t, map[string]string{
			crd.LabelInstance:        "emqx",
			crd.LabelManagedBy:       "emqx-operator",
			crd.LabelMriaRole:        "replicant",
			crd.LabelPodTemplateHash: got.Labels[crd.LabelPodTemplateHash],
			"repl-label-key":         "repl-label-value",
		}, got.Spec.Template.Labels)

		assert.EqualValues(t, map[string]string{
			crd.LabelInstance:        "emqx",
			crd.LabelManagedBy:       "emqx-operator",
			crd.LabelMriaRole:        "replicant",
			crd.LabelPodTemplateHash: got.Labels[crd.LabelPodTemplateHash],
			"repl-label-key":         "repl-label-value",
		}, got.Spec.Selector.MatchLabels)
	})

	t.Run("check dnsConfig propagation", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Spec.ReplicantTemplate.Spec.DNSConfig = &corev1.PodDNSConfig{
			Nameservers: []string{"1.1.1.1"},
			Options: []corev1.PodDNSConfigOption{
				{Name: "ndots", Value: ptr.To("3")},
			},
		}
		got := newReplicaSet(emqx)
		assert.Equal(t, emqx.Spec.ReplicantTemplate.Spec.DNSConfig, got.Spec.Template.Spec.DNSConfig)
	})

	t.Run("check no bootstrap API keys", func(t *testing.T) {
		emqx := instance.DeepCopy()
		rs := newReplicaSet(emqx)
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
		emqx.Spec.Config.Roots = crd.ConfigRoots{
			"dashboard": apiextv1.JSON{Raw: util.FromYAMLString(`
                listeners:
                    http:
                        bind: 18083
            `)},
		}
		got := newReplicaSet(emqx)
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
		emqx.Spec.Config.Roots = crd.ConfigRoots{
			"dashboard": apiextv1.JSON{Raw: util.FromYAMLString(`
                listeners:
                    http:
                        bind: 0
                    https:
                        bind: 18084
            `)},
		}
		got := newReplicaSet(emqx)
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
		emqx.Spec.Config.Roots = crd.ConfigRoots{
			"dashboard": apiextv1.JSON{Raw: util.FromYAMLString(`
                listeners:
                    http:
                        bind: 18083
                    https:
                        bind: 18084
            `)},
		}
		got := newReplicaSet(emqx)
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
