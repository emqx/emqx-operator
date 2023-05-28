package v2alpha1

import (
	"context"
	"reflect"

	emperror "emperror.dev/errors"
	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	"github.com/emqx/emqx-operator/internal/handler"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addCore struct {
	*EMQXReconciler
}

func (a *addCore) reconcile(ctx context.Context, instance *appsv2alpha1.EMQX, _ Requester) subResult {
	sts := generateStatefulSet(instance)
	if err := a.CreateOrUpdateList(instance, a.Scheme, []client.Object{sts}); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to create or update statefulSet")}
	}
	return subResult{}
}

func generateStatefulSet(instance *appsv2alpha1.EMQX) *appsv1.StatefulSet {
	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      instance.Spec.CoreTemplate.Labels,
			Annotations: instance.Spec.CoreTemplate.Annotations,
		},
		Spec: corev1.PodSpec{
			ImagePullSecrets: instance.Spec.ImagePullSecrets,
			SecurityContext:  instance.Spec.CoreTemplate.Spec.PodSecurityContext,
			Affinity:         instance.Spec.CoreTemplate.Spec.Affinity,
			Tolerations:      instance.Spec.CoreTemplate.Spec.ToleRations,
			NodeName:         instance.Spec.CoreTemplate.Spec.NodeName,
			NodeSelector:     instance.Spec.CoreTemplate.Spec.NodeSelector,
			InitContainers:   instance.Spec.CoreTemplate.Spec.InitContainers,
			Containers: append([]corev1.Container{
				{
					Name:            EMQXContainerName,
					Image:           instance.Spec.Image,
					ImagePullPolicy: corev1.PullPolicy(instance.Spec.ImagePullPolicy),
					Command:         instance.Spec.CoreTemplate.Spec.Command,
					Args:            instance.Spec.CoreTemplate.Spec.Args,
					Ports:           instance.Spec.CoreTemplate.Spec.Ports,
					Env: append([]corev1.EnvVar{
						{
							Name: "POD_NAME",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.name",
								},
							},
						},
						{
							Name: "POD_NAMESPACE",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.namespace",
								},
							},
						},
						{
							Name: "STS_HEADLESS_SERVICE_NAME",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.annotations['apps.emqx.io/headless-service-name']",
								},
							},
						},
						{
							Name:  "EMQX_HOST",
							Value: "$(POD_NAME).$(STS_HEADLESS_SERVICE_NAME).$(POD_NAMESPACE).svc.cluster.local",
						},
						{
							Name:  "EMQX_NODE__DB_ROLE",
							Value: "core",
						},
						{
							Name: "EMQX_NODE__COOKIE",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: instance.NodeCookieNamespacedName().Name,
									},
									Key: "node_cookie",
								},
							},
						},
						{
							Name:  "EMQX_DASHBOARD__BOOTSTRAP_USERS_FILE",
							Value: `"/opt/emqx/data/bootstrap_user"`,
						},
					}, instance.Spec.CoreTemplate.Spec.Env...),
					EnvFrom:         instance.Spec.CoreTemplate.Spec.EnvFrom,
					Resources:       instance.Spec.CoreTemplate.Spec.Resources,
					SecurityContext: instance.Spec.CoreTemplate.Spec.ContainerSecurityContext,
					LivenessProbe:   instance.Spec.CoreTemplate.Spec.LivenessProbe,
					ReadinessProbe:  instance.Spec.CoreTemplate.Spec.ReadinessProbe,
					StartupProbe:    instance.Spec.CoreTemplate.Spec.StartupProbe,
					Lifecycle:       instance.Spec.CoreTemplate.Spec.Lifecycle,
					VolumeMounts: append([]corev1.VolumeMount{
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
							Name:      instance.CoreNodeNamespacedName().Name + "-log",
							MountPath: "/opt/emqx/log",
						},
						{
							Name:      instance.CoreNodeNamespacedName().Name + "-data",
							MountPath: "/opt/emqx/data",
						},
					}, instance.Spec.CoreTemplate.Spec.ExtraVolumeMounts...),
				},
			}, instance.Spec.CoreTemplate.Spec.ExtraContainers...),
			Volumes: append([]corev1.Volume{
				{
					Name: "bootstrap-user",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: instance.BootstrapUserNamespacedName().Name,
						},
					},
				},
				{
					Name: "bootstrap-config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: instance.BootstrapConfigNamespacedName().Name,
							},
						},
					},
				},
				{
					Name: instance.CoreNodeNamespacedName().Name + "-log",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			}, instance.Spec.CoreTemplate.Spec.ExtraVolumes...),
		},
	}

	podTemplate.Annotations = handler.SetManagerContainerAnnotation(podTemplate.Annotations, podTemplate.Spec.Containers)
	podTemplate.Annotations["apps.emqx.io/headless-service-name"] = instance.HeadlessServiceNamespacedName().Name

	sts := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.Namespace,
			Name:        instance.Spec.CoreTemplate.Name,
			Labels:      instance.Spec.CoreTemplate.Labels,
			Annotations: instance.Spec.CoreTemplate.Annotations,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: instance.HeadlessServiceNamespacedName().Name,
			Replicas:    instance.Spec.CoreTemplate.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: instance.Spec.CoreTemplate.Labels,
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template:            podTemplate,
		},
	}
	if !reflect.ValueOf(instance.Spec.CoreTemplate.Spec.VolumeClaimTemplates).IsZero() {
		sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instance.CoreNodeNamespacedName().Name + "-data",
					Namespace: instance.Namespace,
					Labels:    instance.Spec.CoreTemplate.Labels,
				},
				Spec: instance.Spec.CoreTemplate.Spec.VolumeClaimTemplates,
			},
		}
	} else {
		sts.Spec.Template.Spec.Volumes = append([]corev1.Volume{
			{
				Name: instance.CoreNodeNamespacedName().Name + "-data",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		}, sts.Spec.Template.Spec.Volumes...)
	}

	return sts
}
