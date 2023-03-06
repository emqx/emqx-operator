package v2alpha1

import (
	"context"
	"time"

	emperror "emperror.dev/errors"
	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addRepl struct {
	*EMQXReconciler
}

func (a *addRepl) reconcile(ctx context.Context, instance *appsv2alpha1.EMQX) subResult {
	if !instance.Status.IsRunning() && !instance.Status.IsCoreNodesReady() {
		return subResult{result: ctrl.Result{RequeueAfter: time.Second}}
	}

	deploy := generateDeployment(instance)
	if err := a.CreateOrUpdateList(instance, a.Scheme, []client.Object{deploy}); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to create or update deployment")}
	}

	return subResult{}
}

func generateDeployment(instance *appsv2alpha1.EMQX) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.Namespace,
			Name:        instance.Spec.ReplicantTemplate.Name,
			Labels:      instance.Spec.ReplicantTemplate.Labels,
			Annotations: instance.Spec.ReplicantTemplate.Annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: instance.Spec.ReplicantTemplate.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: instance.Spec.ReplicantTemplate.Labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      instance.Spec.ReplicantTemplate.Labels,
					Annotations: instance.Spec.ReplicantTemplate.Annotations,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: instance.Spec.ImagePullSecrets,
					SecurityContext:  instance.Spec.ReplicantTemplate.Spec.PodSecurityContext,
					Affinity:         instance.Spec.ReplicantTemplate.Spec.Affinity,
					Tolerations:      instance.Spec.ReplicantTemplate.Spec.ToleRations,
					NodeName:         instance.Spec.ReplicantTemplate.Spec.NodeName,
					NodeSelector:     instance.Spec.ReplicantTemplate.Spec.NodeSelector,
					InitContainers:   instance.Spec.ReplicantTemplate.Spec.InitContainers,
					Containers: append([]corev1.Container{
						{
							Name:            EMQXContainerName,
							Image:           instance.Spec.Image,
							ImagePullPolicy: instance.Spec.ImagePullPolicy,
							Command:         instance.Spec.ReplicantTemplate.Spec.Command,
							Args:            instance.Spec.ReplicantTemplate.Spec.Args,
							Ports:           instance.Spec.ReplicantTemplate.Spec.Ports,
							Env: append([]corev1.EnvVar{
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
							}, instance.Spec.ReplicantTemplate.Spec.Env...),
							EnvFrom:         instance.Spec.ReplicantTemplate.Spec.EnvFrom,
							Resources:       instance.Spec.ReplicantTemplate.Spec.Resources,
							SecurityContext: instance.Spec.ReplicantTemplate.Spec.ContainerSecurityContext,
							LivenessProbe:   instance.Spec.ReplicantTemplate.Spec.LivenessProbe,
							ReadinessProbe:  instance.Spec.ReplicantTemplate.Spec.ReadinessProbe,
							StartupProbe:    instance.Spec.ReplicantTemplate.Spec.StartupProbe,
							Lifecycle:       instance.Spec.ReplicantTemplate.Spec.Lifecycle,
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
									Name:      instance.ReplicantNodeNamespacedName().Name + "-data",
									MountPath: "/opt/emqx/data",
								},
							}, instance.Spec.ReplicantTemplate.Spec.ExtraVolumeMounts...),
						},
					}, instance.Spec.ReplicantTemplate.Spec.ExtraContainers...),
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
							Name: instance.ReplicantNodeNamespacedName().Name + "-data",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					}, instance.Spec.ReplicantTemplate.Spec.ExtraVolumes...),
				},
			},
		},
	}
}
