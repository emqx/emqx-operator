package v2beta1

import (
	"testing"

	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

func TestGenerateStatefulSet(t *testing.T) {
	instance := &appsv2beta1.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: appsv2beta1.EMQXSpec{
			Image:         "emqx/emqx:5.1",
			ClusterDomain: "cluster.local",
		},
	}
	instance.Spec.CoreTemplate.Spec.Replicas = pointer.Int32(3)
	instance.Default()
	assert.Nil(t, instance.ValidateCreate())

	t.Run("check metadata", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Annotations = map[string]string{
			"kubectl.kubernetes.io/last-applied-config": "fake",
		}

		got := generateStatefulSet(emqx)
		assert.Equal(t, appsv2beta1.DefaultCoreLabels(emqx), got.Labels)
		assert.NotContains(t, "kubectl.kubernetes.io/last-applied-config", got.Annotations)
	})

	t.Run("check sts spec", func(t *testing.T) {
		emqx := instance.DeepCopy()

		got := generateStatefulSet(emqx)
		assert.Equal(t, int32(3), *got.Spec.Replicas)
		assert.Equal(t, "emqx-headless", got.Spec.ServiceName)
		assert.Equal(t, appsv2beta1.DefaultCoreLabels(emqx), got.Spec.Selector.MatchLabels)
		assert.Equal(t, appsv1.ParallelPodManagement, got.Spec.PodManagementPolicy)
	})

	t.Run("check sts template spec", func(t *testing.T) {
		emqx := instance.DeepCopy()

		emqx.Spec.CoreTemplate.Spec.Affinity = &corev1.Affinity{}
		emqx.Spec.CoreTemplate.Spec.ToleRations = []corev1.Toleration{{Key: "fake"}}
		emqx.Spec.CoreTemplate.Spec.NodeSelector = map[string]string{"fake": "fake"}
		emqx.Spec.CoreTemplate.Spec.NodeName = "fake"
		got := generateStatefulSet(emqx)
		assert.Equal(t, emqx.Spec.CoreTemplate.Spec.Affinity, got.Spec.Template.Spec.Affinity)
		assert.Equal(t, emqx.Spec.CoreTemplate.Spec.ToleRations, got.Spec.Template.Spec.Tolerations)
		assert.Equal(t, emqx.Spec.CoreTemplate.Spec.NodeSelector, got.Spec.Template.Spec.NodeSelector)
		assert.Equal(t, emqx.Spec.CoreTemplate.Spec.NodeName, got.Spec.Template.Spec.NodeName)

		emqx.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: "fake-secret"}}
		got = generateStatefulSet(emqx)
		assert.Equal(t, emqx.Spec.ImagePullSecrets, got.Spec.Template.Spec.ImagePullSecrets)

		emqx.Spec.CoreTemplate.Spec.PodSecurityContext = &corev1.PodSecurityContext{
			RunAsUser:  pointer.Int64(1000),
			RunAsGroup: pointer.Int64(1000),
			FSGroup:    pointer.Int64(1000),
		}
		got = generateStatefulSet(emqx)
		assert.Equal(t, emqx.Spec.CoreTemplate.Spec.PodSecurityContext, got.Spec.Template.Spec.SecurityContext)

		emqx.Spec.CoreTemplate.Spec.InitContainers = []corev1.Container{{Name: "fake-init-container"}}
		got = generateStatefulSet(emqx)
		assert.Equal(t, emqx.Spec.CoreTemplate.Spec.InitContainers, got.Spec.Template.Spec.InitContainers)
	})

	t.Run("check sts template spec containers", func(t *testing.T) {
		emqx := instance.DeepCopy()

		emqx.Spec.CoreTemplate.Spec.ExtraContainers = []corev1.Container{{Name: "fake-container"}}
		got := generateStatefulSet(emqx)
		assert.Len(t, got.Spec.Template.Spec.Containers, 2)

		emqx.Spec.Image = "emqx/emqx:5.1"
		emqx.Spec.ImagePullPolicy = corev1.PullIfNotPresent
		emqx.Spec.CoreTemplate.Spec.Command = []string{"fake"}
		emqx.Spec.CoreTemplate.Spec.Args = []string{"fake"}
		emqx.Spec.CoreTemplate.Spec.Ports = []corev1.ContainerPort{{Name: "fake"}}
		emqx.Spec.CoreTemplate.Spec.Env = []corev1.EnvVar{{Name: "foo", Value: "bar"}}
		emqx.Spec.CoreTemplate.Spec.EnvFrom = []corev1.EnvFromSource{
			{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "fake-config",
					},
				},
			},
		}
		emqx.Spec.CoreTemplate.Spec.Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("100Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("100Mi"),
			},
		}
		emqx.Spec.CoreTemplate.Spec.ContainerSecurityContext = &corev1.SecurityContext{
			RunAsUser:    pointer.Int64(1000),
			RunAsGroup:   pointer.Int64(1000),
			RunAsNonRoot: pointer.Bool(true),
		}
		emqx.Spec.CoreTemplate.Spec.ReadinessProbe = &corev1.Probe{
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
		emqx.Spec.CoreTemplate.Spec.Lifecycle = &corev1.Lifecycle{
			PreStop: &corev1.LifecycleHandler{
				Exec: &corev1.ExecAction{
					Command: []string{"emqx", "ctl", "cluster", "leave"},
				},
			},
		}
		emqx.Spec.CoreTemplate.Spec.ExtraVolumeMounts = []corev1.VolumeMount{{Name: "fake-volume-mount"}}

		got = generateStatefulSet(emqx)
		assert.Equal(t, emqx.Spec.Image, got.Spec.Template.Spec.Containers[0].Image)
		assert.Equal(t, emqx.Spec.ImagePullPolicy, got.Spec.Template.Spec.Containers[0].ImagePullPolicy)
		assert.Equal(t, emqx.Spec.CoreTemplate.Spec.Command, got.Spec.Template.Spec.Containers[0].Command)
		assert.Equal(t, emqx.Spec.CoreTemplate.Spec.Args, got.Spec.Template.Spec.Containers[0].Args)
		assert.Equal(t, emqx.Spec.CoreTemplate.Spec.Ports, got.Spec.Template.Spec.Containers[0].Ports)
		assert.ElementsMatch(t, []corev1.EnvVar{
			{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
			{
				Name:  "EMQX_CLUSTER__DISCOVERY_STRATEGY",
				Value: "dns",
			},
			{
				Name:  "EMQX_CLUSTER__DNS__RECORD_TYPE",
				Value: "srv",
			},
			{
				Name:  "EMQX_CLUSTER__DNS__NAME",
				Value: "emqx-headless.emqx.svc.cluster.local",
			},
			{
				Name:  "EMQX_NODE__DATA_DIR",
				Value: "data",
			},
			{
				Name:  "EMQX_NODE__ROLE",
				Value: "core",
			},
			{
				Name:  "EMQX_HOST",
				Value: "$(POD_NAME).$(EMQX_CLUSTER__DNS__NAME)",
			},
			{
				Name: "EMQX_NODE__COOKIE",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: emqx.NodeCookieNamespacedName().Name,
						},
						Key: "node_cookie",
					},
				},
			},
			{
				Name:  "EMQX_API_KEY__BOOTSTRAP_FILE",
				Value: `"/opt/emqx/data/bootstrap_api_key"`,
			},
			{
				Name:  "foo",
				Value: "bar",
			},
		}, got.Spec.Template.Spec.Containers[0].Env)
		assert.Equal(t, emqx.Spec.CoreTemplate.Spec.EnvFrom, got.Spec.Template.Spec.Containers[0].EnvFrom)
		assert.Equal(t, emqx.Spec.CoreTemplate.Spec.Resources, got.Spec.Template.Spec.Containers[0].Resources)
		assert.Equal(t, emqx.Spec.CoreTemplate.Spec.ContainerSecurityContext, got.Spec.Template.Spec.Containers[0].SecurityContext)
		assert.Equal(t, emqx.Spec.CoreTemplate.Spec.ReadinessProbe, got.Spec.Template.Spec.Containers[0].ReadinessProbe)
		assert.Equal(t, emqx.Spec.CoreTemplate.Spec.Lifecycle, got.Spec.Template.Spec.Containers[0].Lifecycle)
		assert.Equal(t, []corev1.VolumeMount{
			{
				Name:      "bootstrap-api-key",
				MountPath: "/opt/emqx/data/bootstrap_api_key",
				SubPath:   "bootstrap_api_key",
				ReadOnly:  true,
			},
			{
				Name:      "bootstrap-config",
				MountPath: "/opt/emqx/etc/emqx.conf",
				SubPath:   "emqx.conf",
				ReadOnly:  true,
			},
			{
				Name:      "emqx-core-log",
				MountPath: "/opt/emqx/log",
			},
			{
				Name:      "emqx-core-data",
				MountPath: "/opt/emqx/data",
			},
			{
				Name: "fake-volume-mount",
			},
		}, got.Spec.Template.Spec.Containers[0].VolumeMounts)
	})

	t.Run("check sts spec volume", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Spec.CoreTemplate.Spec.ExtraVolumes = []corev1.Volume{{Name: "fake-volume"}}

		got := generateStatefulSet(emqx)
		assert.Equal(t, []corev1.Volume{
			{
				Name: "emqx-core-data",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: "bootstrap-api-key",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: emqx.BootstrapAPIKeyNamespacedName().Name,
					},
				},
			},
			{
				Name: "bootstrap-config",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: emqx.ConfigsNamespacedName().Name,
						},
					},
				},
			},
			{
				Name: "emqx-core-log",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: "fake-volume",
			},
		}, got.Spec.Template.Spec.Volumes)
	})

	t.Run("check sts volume claim templates", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Spec.CoreTemplate.Spec.VolumeClaimTemplates = corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
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
					Labels:    appsv2beta1.DefaultCoreLabels(emqx),
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.ResourceRequirements{
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
