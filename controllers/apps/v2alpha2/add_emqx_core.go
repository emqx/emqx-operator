package v2alpha2

import (
	"context"
	"fmt"
	"reflect"

	emperror "emperror.dev/errors"
	"github.com/cisco-open/k8s-objectmatcher/patch"
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

type addCore struct {
	*EMQXReconciler
}

func (a *addCore) reconcile(ctx context.Context, instance *appsv2alpha2.EMQX, _ innerReq.RequesterInterface) subResult {
	preSts := a.getNewStatefulSet(ctx, instance)
	if preSts.UID == "" {
		_ = ctrl.SetControllerReference(instance, preSts, a.Scheme)
		if err := a.Handler.Create(preSts); err != nil {
			if k8sErrors.IsAlreadyExists(emperror.Cause(err)) {
				if instance.Status.CoreNodesStatus.CollisionCount == nil {
					instance.Status.CoreNodesStatus.CollisionCount = pointer.Int32(0)
				}
				*instance.Status.CoreNodesStatus.CollisionCount++
				_ = a.Client.Status().Update(ctx, instance)
				return subResult{result: ctrl.Result{Requeue: true}}
			}
			return subResult{err: emperror.Wrap(err, "failed to create statefulSet")}
		}
		instance.Status.SetCondition(metav1.Condition{
			Type:    appsv2alpha2.CoreNodesProgressing,
			Status:  metav1.ConditionTrue,
			Reason:  "CreateNewStatefulSet",
			Message: "Create new statefulSet",
		})
		instance.Status.RemoveCondition(appsv2alpha2.Ready)
		instance.Status.RemoveCondition(appsv2alpha2.Available)
		instance.Status.RemoveCondition(appsv2alpha2.CoreNodesReady)
		instance.Status.CoreNodesStatus.CurrentRevision = preSts.Labels[appsv2alpha2.PodTemplateHashLabelKey]
		_ = a.Client.Status().Update(ctx, instance)
	} else {
		storageSts := &appsv1.StatefulSet{}
		_ = a.Client.Get(ctx, client.ObjectKeyFromObject(preSts), storageSts)
		patchResult, _ := a.Patcher.Calculate(storageSts, preSts,
			patch.IgnoreStatusFields(),
			patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(),
		)
		if !patchResult.IsEmpty() {
			logger := log.FromContext(ctx)
			logger.V(1).Info("got different statefulSet for EMQX core nodes, will update statefulSet", "patch", string(patchResult.Patch))

			if err := a.Handler.Update(preSts); err != nil {
				return subResult{err: emperror.Wrap(err, "failed to update statefulSet")}
			}

			instance.Status.SetCondition(metav1.Condition{
				Type:    appsv2alpha2.CoreNodesProgressing,
				Status:  metav1.ConditionTrue,
				Reason:  "CreateNewStatefulSet",
				Message: "Create new statefulSet",
			})
			instance.Status.RemoveCondition(appsv2alpha2.Ready)
			instance.Status.RemoveCondition(appsv2alpha2.Available)
			instance.Status.RemoveCondition(appsv2alpha2.CoreNodesReady)
			_ = a.Client.Status().Update(ctx, instance)
		}
	}

	return subResult{}
}

func (a *addCore) getNewStatefulSet(ctx context.Context, instance *appsv2alpha2.EMQX) *appsv1.StatefulSet {
	preSts := generateStatefulSet(instance)
	podTemplateSpecHash := computeHash(preSts.Spec.Template.DeepCopy(), instance.Status.CoreNodesStatus.CollisionCount)
	preSts.Name = preSts.Name + "-" + podTemplateSpecHash
	preSts.Labels = appsv2alpha2.CloneAndAddLabel(preSts.Labels, appsv2alpha2.PodTemplateHashLabelKey, podTemplateSpecHash)
	preSts.Spec.Template.Labels = appsv2alpha2.CloneAndAddLabel(preSts.Spec.Template.Labels, appsv2alpha2.PodTemplateHashLabelKey, podTemplateSpecHash)
	preSts.Spec.Selector = appsv2alpha2.CloneSelectorAndAddLabel(preSts.Spec.Selector, appsv2alpha2.PodTemplateHashLabelKey, podTemplateSpecHash)

	currentSts, _ := getStateFulSetList(ctx, a.Client, instance)
	if currentSts == nil {
		return preSts
	}

	patchResult, _ := a.Patcher.Calculate(
		currentSts.DeepCopy(),
		preSts.DeepCopy(),
		justCheckPodTemplate(),
	)
	if patchResult.IsEmpty() {
		preSts.ObjectMeta = currentSts.ObjectMeta
		preSts.Spec.Template.ObjectMeta = currentSts.Spec.Template.ObjectMeta
		preSts.Spec.Selector = currentSts.Spec.Selector
		return preSts
	}

	logger := log.FromContext(ctx)
	logger.V(1).Info("got different pod template for EMQX core nodes, will create new statefulSet", "patch", string(patchResult.Patch))
	return preSts
}

func generateStatefulSet(instance *appsv2alpha2.EMQX) *appsv1.StatefulSet {
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
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: instance.Spec.CoreTemplate.Labels,
				},
				Spec: corev1.PodSpec{
					ReadinessGates: []corev1.PodReadinessGate{
						{
							ConditionType: appsv2alpha2.PodOnServing,
						},
					},
					ImagePullSecrets:   instance.Spec.ImagePullSecrets,
					SecurityContext:    instance.Spec.CoreTemplate.Spec.PodSecurityContext,
					Affinity:           instance.Spec.CoreTemplate.Spec.Affinity,
					Tolerations:        instance.Spec.CoreTemplate.Spec.ToleRations,
					NodeName:           instance.Spec.CoreTemplate.Spec.NodeName,
					NodeSelector:       instance.Spec.CoreTemplate.Spec.NodeSelector,
					ServiceAccountName: instance.Spec.ServiceAccountName,
					InitContainers:     instance.Spec.CoreTemplate.Spec.InitContainers,
					Containers: append([]corev1.Container{
						{
							Name:            appsv2alpha2.DefaultContainerName,
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
									Name:  "EMQX_CLUSTER__DISCOVERY_STRATEGY",
									Value: "dns",
								},
								{
									Name:  "EMQX_CLUSTER__DNS__RECORD_TYPE",
									Value: "srv",
								},
								{
									Name:  "EMQX_CLUSTER__DNS__NAME",
									Value: fmt.Sprintf("%s.%s.svc.%s", instance.HeadlessServiceNamespacedName().Name, instance.Namespace, instance.Spec.ClusterDomain),
								},
								{
									Name:  "EMQX_HOST",
									Value: "$(POD_NAME).$(EMQX_CLUSTER__DNS__NAME)",
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
									Name:      instance.Spec.CoreTemplate.Name + "-log",
									MountPath: "/opt/emqx/log",
								},
								{
									Name:      instance.Spec.CoreTemplate.Name + "-data",
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
							Name: instance.Spec.CoreTemplate.Name + "-log",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					}, instance.Spec.CoreTemplate.Spec.ExtraVolumes...),
				},
			},
		},
	}

	if !reflect.ValueOf(instance.Spec.CoreTemplate.Spec.VolumeClaimTemplates).IsZero() {
		sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instance.Spec.CoreTemplate.Name + "-data",
					Namespace: instance.Namespace,
					Labels:    instance.Spec.CoreTemplate.Labels,
				},
				Spec: instance.Spec.CoreTemplate.Spec.VolumeClaimTemplates,
			},
		}
	} else {
		sts.Spec.Template.Spec.Volumes = append([]corev1.Volume{
			{
				Name: instance.Spec.CoreTemplate.Name + "-data",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		}, sts.Spec.Template.Spec.Volumes...)
	}

	return sts
}
