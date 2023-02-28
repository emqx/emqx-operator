package v2alpha1

import (
	"testing"

	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var replicantLabels = map[string]string{
	"apps.emqx.io/instance":   "emqx",
	"apps.emqx.io/managed-by": "emqx-operator",
	"apps.emqx.io/db-role":    "replicant",
}

func TestGenerateD(t *testing.T) {
	instance := &appsv2alpha1.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: appsv2alpha1.EMQXSpec{
			Image: "emqx/emqx:5.0",
			ReplicantTemplate: appsv2alpha1.EMQXReplicantTemplate{
				Spec: appsv2alpha1.EMQXReplicantTemplateSpec{
					Replicas: &[]int32{3}[0],
				},
			},
		},
	}
	instance.Default()
	assert.Nil(t, instance.ValidateCreate())

	t.Run("check metadata", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Annotations = map[string]string{
			"kubectl.kubernetes.io/last-applied-configuration": "fake",
		}

		got := generateDeployment(emqx)
		assert.Equal(t, replicantLabels, got.Labels)
		assert.NotContains(t, "kubectl.kubernetes.io/last-applied-configuration", got.Annotations)
	})

	t.Run("check deploy spec", func(t *testing.T) {
		emqx := instance.DeepCopy()

		got := generateDeployment(emqx)
		assert.Equal(t, int32(3), *got.Spec.Replicas)
		assert.Equal(t, replicantLabels, got.Spec.Selector.MatchLabels)
	})

	t.Run("check deploy template metadata", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Spec.ReplicantTemplate.Annotations = map[string]string{"foo": "bar"}
		emqx.Default()

		got := generateDeployment(emqx)
		assert.Equal(t, replicantLabels, got.Spec.Template.ObjectMeta.Labels)
		assert.Equal(t, map[string]string{
			"foo": "bar",
		}, got.Spec.Template.ObjectMeta.Annotations)
	})

	t.Run("check deploy template spec", func(t *testing.T) {
		emqx := instance.DeepCopy()

		emqx.Spec.ReplicantTemplate.Spec.Affinity = &corev1.Affinity{}
		emqx.Spec.ReplicantTemplate.Spec.ToleRations = []corev1.Toleration{{Key: "fake"}}
		emqx.Spec.ReplicantTemplate.Spec.NodeSelector = map[string]string{"fake": "fake"}
		emqx.Spec.ReplicantTemplate.Spec.NodeName = "fake"
		got := generateDeployment(emqx)
		assert.Equal(t, emqx.Spec.ReplicantTemplate.Spec.Affinity, got.Spec.Template.Spec.Affinity)
		assert.Equal(t, emqx.Spec.ReplicantTemplate.Spec.ToleRations, got.Spec.Template.Spec.Tolerations)
		assert.Equal(t, emqx.Spec.ReplicantTemplate.Spec.NodeSelector, got.Spec.Template.Spec.NodeSelector)
		assert.Equal(t, emqx.Spec.ReplicantTemplate.Spec.NodeName, got.Spec.Template.Spec.NodeName)

		emqx.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: "fake-secret"}}
		got = generateDeployment(emqx)
		assert.Equal(t, emqx.Spec.ImagePullSecrets, got.Spec.Template.Spec.ImagePullSecrets)

		emqx.Spec.ReplicantTemplate.Spec.PodSecurityContext = &corev1.PodSecurityContext{
			RunAsUser:  &[]int64{1001}[0],
			RunAsGroup: &[]int64{1001}[0],
			FSGroup:    &[]int64{1001}[0],
		}
		got = generateDeployment(emqx)
		assert.Equal(t, emqx.Spec.ReplicantTemplate.Spec.PodSecurityContext, got.Spec.Template.Spec.SecurityContext)

		emqx.Spec.ReplicantTemplate.Spec.InitContainers = []corev1.Container{{Name: "fake-init-container"}}
		got = generateDeployment(emqx)
		assert.Equal(t, emqx.Spec.ReplicantTemplate.Spec.InitContainers, got.Spec.Template.Spec.InitContainers)
	})

	t.Run("check deploy template spec containers", func(t *testing.T) {
		emqx := instance.DeepCopy()

		emqx.Spec.ReplicantTemplate.Spec.ExtraContainers = []corev1.Container{{Name: "fake-container"}}
		got := generateDeployment(emqx)
		assert.Len(t, got.Spec.Template.Spec.Containers, 2)

		emqx.Spec.Image = "emqx/emqx:5.0"
		emqx.Spec.ImagePullPolicy = corev1.PullIfNotPresent
		emqx.Spec.ReplicantTemplate.Spec.Command = []string{"fake"}
		emqx.Spec.ReplicantTemplate.Spec.Args = []string{"fake"}
		emqx.Spec.ReplicantTemplate.Spec.Ports = []corev1.ContainerPort{{Name: "fake"}}
		emqx.Spec.ReplicantTemplate.Spec.Env = []corev1.EnvVar{{Name: "foo", Value: "bar"}}
		emqx.Spec.ReplicantTemplate.Spec.EnvFrom = []corev1.EnvFromSource{
			{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "fake-config",
					},
				},
			},
		}
		emqx.Spec.ReplicantTemplate.Spec.Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("100Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("100Mi"),
			},
		}
		emqx.Spec.ReplicantTemplate.Spec.ContainerSecurityContext = &corev1.SecurityContext{
			RunAsUser:    &[]int64{1001}[0],
			RunAsGroup:   &[]int64{1001}[0],
			RunAsNonRoot: &[]bool{true}[0],
		}
		emqx.Spec.ReplicantTemplate.Spec.ReadinessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/status",
					Port: intstr.FromInt(18083),
				},
			},
			InitialDelaySeconds: int32(10),
			PeriodSeconds:       int32(5),
			FailureThreshold:    int32(30),
		}
		emqx.Spec.ReplicantTemplate.Spec.Lifecycle = &corev1.Lifecycle{
			PreStop: &corev1.LifecycleHandler{
				Exec: &corev1.ExecAction{
					Command: []string{"emqx", "ctl", "cluster", "leave"},
				},
			},
		}
		emqx.Spec.ReplicantTemplate.Spec.ExtraVolumeMounts = []corev1.VolumeMount{{Name: "fake-volume-mount"}}

		got = generateDeployment(emqx)
		assert.Equal(t, emqx.Spec.Image, got.Spec.Template.Spec.Containers[0].Image)
		assert.Equal(t, emqx.Spec.ImagePullPolicy, got.Spec.Template.Spec.Containers[0].ImagePullPolicy)
		assert.Equal(t, emqx.Spec.ReplicantTemplate.Spec.Command, got.Spec.Template.Spec.Containers[0].Command)
		assert.Equal(t, emqx.Spec.ReplicantTemplate.Spec.Args, got.Spec.Template.Spec.Containers[0].Args)
		assert.Equal(t, emqx.Spec.ReplicantTemplate.Spec.Ports, got.Spec.Template.Spec.Containers[0].Ports)
		assert.Equal(t, []corev1.EnvVar{
			{
				Name:  "EMQX_NODE__DB_ROLE",
				Value: "replicant",
			},
			{
				Name: "EMQX_HOST",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "status.podIP",
					},
				},
			},
			{
				Name: "EMQX_NODE__COOKIE",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "emqx-node-cookie",
						},
						Key: "node_cookie",
					},
				},
			},
			{
				Name:  "EMQX_DASHBOARD__BOOTSTRAP_USERS_FILE",
				Value: `"/opt/emqx/data/bootstrap_user"`,
			},
			{
				Name:  "foo",
				Value: "bar",
			},
		}, got.Spec.Template.Spec.Containers[0].Env)
		assert.Equal(t, emqx.Spec.ReplicantTemplate.Spec.EnvFrom, got.Spec.Template.Spec.Containers[0].EnvFrom)
		assert.Equal(t, emqx.Spec.ReplicantTemplate.Spec.Resources, got.Spec.Template.Spec.Containers[0].Resources)
		assert.Equal(t, emqx.Spec.ReplicantTemplate.Spec.ContainerSecurityContext, got.Spec.Template.Spec.Containers[0].SecurityContext)
		assert.Equal(t, emqx.Spec.ReplicantTemplate.Spec.ReadinessProbe, got.Spec.Template.Spec.Containers[0].ReadinessProbe)
		assert.Equal(t, emqx.Spec.ReplicantTemplate.Spec.Lifecycle, got.Spec.Template.Spec.Containers[0].Lifecycle)
		assert.Equal(t, []corev1.VolumeMount{
			{
				Name:      "bootstrap-user",
				MountPath: "/opt/emqx/data/bootstrap_user",
				SubPath:   "bootstrap_user",
				ReadOnly:  true,
			},
			{
				Name:      "bootstrap-config",
				MountPath: "/opt/emqx/etc/emqx.conf",
				SubPath:   "emqx.conf",
				ReadOnly:  true,
			},
			{
				Name:      "emqx-replicant-data",
				MountPath: "/opt/emqx/data",
			},
			{
				Name: "fake-volume-mount",
			},
		}, got.Spec.Template.Spec.Containers[0].VolumeMounts)
	})

	t.Run("check deploy spec volume", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Spec.ReplicantTemplate.Spec.ExtraVolumes = []corev1.Volume{{Name: "fake-volume"}}

		got := generateDeployment(emqx)
		assert.Equal(t, []corev1.Volume{
			{
				Name: "bootstrap-user",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: emqx.BootstrapUserNamespacedName().Name,
					},
				},
			},
			{
				Name: "bootstrap-config",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: emqx.BootstrapConfigNamespacedName().Name,
						},
					},
				},
			},
			{
				Name: "emqx-replicant-data",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: "fake-volume",
			},
		}, got.Spec.Template.Spec.Volumes)
	})
}
