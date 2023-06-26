package v2alpha2

import (
	"context"

	emperror "emperror.dev/errors"
	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type addRepl struct {
	*EMQXReconciler
}

func (a *addRepl) reconcile(ctx context.Context, instance *appsv2alpha2.EMQX, _ innerReq.RequesterInterface) subResult {
	if instance.Spec.ReplicantTemplate == nil {
		return subResult{}
	}
	if !instance.Status.IsConditionTrue(appsv2alpha2.CodeNodesReady) {
		return subResult{}
	}

	rs := a.getNewReplicaSet(ctx, instance)
	if rs.UID == "" {
		_ = ctrl.SetControllerReference(instance, rs, a.Scheme)
		if err := a.Handler.Create(rs); err != nil {
			if k8sErrors.IsAlreadyExists(emperror.Cause(err)) {
				if instance.Status.ReplicantNodesStatus.CollisionCount == nil {
					instance.Status.ReplicantNodesStatus.CollisionCount = pointer.Int32(0)
				}
				*instance.Status.ReplicantNodesStatus.CollisionCount++
				_ = a.Client.Status().Update(ctx, instance)
				return subResult{result: ctrl.Result{Requeue: true}}
			}
			return subResult{err: emperror.Wrap(err, "failed to create replicaSet")}
		}

	} else {
		if err := a.Handler.CreateOrUpdate(rs); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to update replicaSet")}
		}
	}

	if instance.Status.ReplicantNodesStatus.CurrentVersion != rs.Labels[appsv1.DefaultDeploymentUniqueLabelKey] {
		instance.Status.ReplicantNodesStatus.CurrentVersion = rs.Labels[appsv1.DefaultDeploymentUniqueLabelKey]
		_ = a.Client.Status().Update(ctx, instance)
	}

	if err := a.syncReplicaSet(ctx, instance); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to sync replicaSet")}
	}

	return subResult{}
}

func (a *addRepl) getNewReplicaSet(ctx context.Context, instance *appsv2alpha2.EMQX) *appsv1.ReplicaSet {
	preRs := generateReplicaSet(instance)

	list := &appsv1.ReplicaSetList{}
	_ = a.Client.List(context.TODO(), list,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(appsv2alpha2.CloneAndAddLabel(
			instance.Spec.ReplicantTemplate.Labels,
			appsv1.DefaultDeploymentUniqueLabelKey,
			instance.Status.ReplicantNodesStatus.CurrentVersion,
		)),
	)
	if len(list.Items) > 0 {
		rs := list.Items[0].DeepCopy()
		patchResult, _ := a.Patcher.Calculate(
			rs,
			preRs.DeepCopy(),
			justCheckPodTemplateSpec(),
		)
		if patchResult.IsEmpty() {
			preRs.ObjectMeta = rs.ObjectMeta
			preRs.Spec.Template.ObjectMeta = rs.Spec.Template.ObjectMeta
			preRs.Spec.Selector = rs.Spec.Selector
			return preRs
		}
		logger := log.FromContext(ctx)
		logger.V(1).Info("got different patch for EMQX replicant nodes, will create new replicaSet", "patch", string(patchResult.Patch))
	}

	podTemplateSpecHash := computeHash(preRs.Spec.Template.DeepCopy(), instance.Status.ReplicantNodesStatus.CollisionCount)
	preRs.Name = preRs.Name + "-" + podTemplateSpecHash
	preRs.Labels = appsv2alpha2.CloneAndAddLabel(preRs.Labels, appsv1.DefaultDeploymentUniqueLabelKey, podTemplateSpecHash)
	preRs.Spec.Template.Labels = appsv2alpha2.CloneAndAddLabel(preRs.Spec.Template.Labels, appsv1.DefaultDeploymentUniqueLabelKey, podTemplateSpecHash)
	preRs.Spec.Selector = appsv2alpha2.CloneSelectorAndAddLabel(preRs.Spec.Selector, appsv1.DefaultDeploymentUniqueLabelKey, podTemplateSpecHash)

	return preRs
}

func (a *addRepl) syncReplicaSet(ctx context.Context, instance *appsv2alpha2.EMQX) error {
	rsList := getReplicaSetList(ctx, a.Client,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
	)
	if len(rsList) <= 1 {
		return nil
	}

	old := rsList[0].DeepCopy()
	eList := getEventList(ctx, a.Clientset, old)

	if canBeScaledDown(instance, appsv2alpha2.CodeNodesReady, eList) {
		old.Spec.Replicas = pointer.Int32(old.Status.Replicas - 1)
		if err := a.Client.Update(ctx, old); err != nil {
			return emperror.Wrap(err, "failed to scale down old replicaSet")
		}
		return nil
	}
	return nil
}

func generateReplicaSet(instance *appsv2alpha2.EMQX) *appsv1.ReplicaSet {
	return &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ReplicaSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.Namespace,
			Name:        instance.Spec.ReplicantTemplate.Name,
			Annotations: instance.Spec.ReplicantTemplate.Annotations,
			Labels:      instance.Spec.ReplicantTemplate.Labels,
		},
		Spec: appsv1.ReplicaSetSpec{
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
					ReadinessGates: []corev1.PodReadinessGate{
						{
							ConditionType: appsv2alpha2.PodOnServing,
						},
					},
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
									Name:  "EMQX_API_KEY__BOOTSTRAP_FILE",
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
									Name:      instance.Spec.ReplicantTemplate.Name + "-log",
									MountPath: "/opt/emqx/log",
								},
								{
									Name:      instance.Spec.ReplicantTemplate.Name + "-data",
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
							Name: instance.Spec.ReplicantTemplate.Name + "-log",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: instance.Spec.ReplicantTemplate.Name + "-data",
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
