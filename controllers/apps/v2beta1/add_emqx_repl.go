package v2beta1

import (
	"context"
	"fmt"
	"strconv"

	emperror "emperror.dev/errors"
	"github.com/cisco-open/k8s-objectmatcher/patch"
	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type addRepl struct {
	*EMQXReconciler
}

func (a *addRepl) reconcile(ctx context.Context, instance *appsv2beta1.EMQX, _ innerReq.RequesterInterface) subResult {
	if instance.Spec.ReplicantTemplate == nil {
		return subResult{}
	}
	if !instance.Status.IsConditionTrue(appsv2beta1.CoreNodesReady) {
		return subResult{}
	}

	preRs, err := a.getNewReplicaSet(ctx, instance)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get new replicaSet")}
	}
	if preRs.UID == "" {
		_ = ctrl.SetControllerReference(instance, preRs, a.Scheme)
		if err := a.Handler.Create(preRs); err != nil {
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
		instance.Status.SetCondition(metav1.Condition{
			Type:    appsv2beta1.ReplicantNodesProgressing,
			Status:  metav1.ConditionTrue,
			Reason:  "CreateNewReplicaSet",
			Message: "Create new replicaSet",
		})
		instance.Status.RemoveCondition(appsv2beta1.Ready)
		instance.Status.RemoveCondition(appsv2beta1.Available)
		instance.Status.RemoveCondition(appsv2beta1.ReplicantNodesReady)
		instance.Status.ReplicantNodesStatus.UpdateRevision = preRs.Labels[appsv2beta1.LabelsPodTemplateHashKey]
		_ = a.Client.Status().Update(ctx, instance)
	} else {
		storageRs := &appsv1.ReplicaSet{}
		_ = a.Client.Get(ctx, client.ObjectKeyFromObject(preRs), storageRs)
		patchResult, _ := a.Patcher.Calculate(storageRs, preRs,
			patch.IgnoreStatusFields(),
			patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(),
		)
		if !patchResult.IsEmpty() {
			logger := log.FromContext(ctx)
			logger.Info("got different replicaSet for EMQX replicant nodes, will update replicaSet", "patch", string(patchResult.Patch))

			if err := a.Handler.Update(preRs); err != nil {
				return subResult{err: emperror.Wrap(err, "failed to update replicaSet")}
			}

			instance.Status.SetCondition(metav1.Condition{
				Type:    appsv2beta1.ReplicantNodesProgressing,
				Status:  metav1.ConditionTrue,
				Reason:  "CreateNewReplicaSet",
				Message: "Create new replicaSet",
			})
			instance.Status.RemoveCondition(appsv2beta1.Ready)
			instance.Status.RemoveCondition(appsv2beta1.Available)
			instance.Status.RemoveCondition(appsv2beta1.ReplicantNodesReady)
			_ = a.Client.Status().Update(ctx, instance)
		}
	}

	return subResult{}
}

func (a *addRepl) getNewReplicaSet(ctx context.Context, instance *appsv2beta1.EMQX) (*appsv1.ReplicaSet, error) {
	configMap := &corev1.ConfigMap{}
	if err := a.Client.Get(ctx, types.NamespacedName{
		Name:      instance.ConfigsNamespacedName().Name,
		Namespace: instance.Namespace,
	}, configMap); err != nil {
		return nil, emperror.Wrap(err, "failed to get configMap")
	}

	var containerPort corev1.ContainerPort
	if svcPort, err := appsv2beta1.GetDashboardServicePort(instance.Spec.Config.Data); err != nil {
		containerPort = corev1.ContainerPort{
			Name:          "dashboard",
			Protocol:      corev1.ProtocolTCP,
			ContainerPort: 18083,
		}
	} else {
		containerPort = corev1.ContainerPort{
			Name:          "dashboard",
			Protocol:      corev1.ProtocolTCP,
			ContainerPort: svcPort.Port,
		}
	}

	preRs := generateReplicaSet(instance)
	podTemplateSpecHash := computeHash(preRs.Spec.Template.DeepCopy(), instance.Status.ReplicantNodesStatus.CollisionCount)
	preRs.Name = preRs.Name + "-" + podTemplateSpecHash
	preRs.Labels = appsv2beta1.CloneAndAddLabel(preRs.Labels, appsv2beta1.LabelsPodTemplateHashKey, podTemplateSpecHash)
	preRs.Spec.Selector = appsv2beta1.CloneSelectorAndAddLabel(preRs.Spec.Selector, appsv2beta1.LabelsPodTemplateHashKey, podTemplateSpecHash)
	preRs.Spec.Template.Labels = appsv2beta1.CloneAndAddLabel(preRs.Spec.Template.Labels, appsv2beta1.LabelsPodTemplateHashKey, podTemplateSpecHash)
	preRs.Spec.Template.Spec.Containers[0].Ports = appsv2beta1.MergeContainerPorts(
		preRs.Spec.Template.Spec.Containers[0].Ports,
		[]corev1.ContainerPort{
			containerPort,
		},
	)
	preRs.Spec.Template.Spec.Containers[0].Env = append([]corev1.EnvVar{
		{Name: "EMQX_DASHBOARD__LISTENERS__HTTP__BIND", Value: strconv.Itoa(int(containerPort.ContainerPort))},
	}, preRs.Spec.Template.Spec.Containers[0].Env...)

	updateRs, _, _ := getReplicaSetList(ctx, a.Client, instance)
	if updateRs == nil {
		return preRs, nil
	}

	patchResult, err := a.Patcher.Calculate(
		updateRs.DeepCopy(),
		preRs.DeepCopy(),
		justCheckPodTemplate(),
	)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to calculate patch result")
	}
	if patchResult.IsEmpty() {
		preRs.ObjectMeta = updateRs.DeepCopy().ObjectMeta
		preRs.Spec.Template.ObjectMeta = updateRs.DeepCopy().Spec.Template.ObjectMeta
		preRs.Spec.Selector = updateRs.DeepCopy().Spec.Selector
		return preRs, nil
	}
	logger := log.FromContext(ctx)
	logger.Info("got different pod template for EMQX replicant nodes, will create new replicaSet", "patch", string(patchResult.Patch))

	return preRs, nil
}

func generateReplicaSet(instance *appsv2beta1.EMQX) *appsv1.ReplicaSet {
	labels := appsv2beta1.CloneAndMergeMap(
		appsv2beta1.DefaultReplicantLabels(instance),
		instance.Spec.ReplicantTemplate.Labels,
	)

	return &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ReplicaSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.Namespace,
			Name:        instance.ReplicantNamespacedName().Name,
			Annotations: instance.Spec.ReplicantTemplate.DeepCopy().Annotations,
			Labels:      labels,
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: instance.Spec.ReplicantTemplate.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: instance.Spec.ReplicantTemplate.DeepCopy().Annotations,
					Labels:      labels,
				},
				Spec: corev1.PodSpec{
					ReadinessGates: []corev1.PodReadinessGate{
						{
							ConditionType: appsv2beta1.PodOnServing,
						},
					},
					ImagePullSecrets:   instance.Spec.ImagePullSecrets,
					ServiceAccountName: instance.Spec.ServiceAccountName,
					SecurityContext:    instance.Spec.ReplicantTemplate.Spec.PodSecurityContext,
					Affinity:           instance.Spec.ReplicantTemplate.Spec.Affinity,
					Tolerations:        instance.Spec.ReplicantTemplate.Spec.ToleRations,
					NodeName:           instance.Spec.ReplicantTemplate.Spec.NodeName,
					NodeSelector:       instance.Spec.ReplicantTemplate.Spec.NodeSelector,
					InitContainers:     instance.Spec.ReplicantTemplate.Spec.InitContainers,
					Containers: append([]corev1.Container{
						{
							Name:            appsv2beta1.DefaultContainerName,
							Image:           instance.Spec.Image,
							ImagePullPolicy: instance.Spec.ImagePullPolicy,
							Command:         instance.Spec.ReplicantTemplate.Spec.Command,
							Args:            instance.Spec.ReplicantTemplate.Spec.Args,
							Ports:           instance.Spec.ReplicantTemplate.Spec.Ports,
							Env: append([]corev1.EnvVar{
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
									Name: "EMQX_HOST",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "EMQX_NODE__DATA_DIR",
									Value: "data",
								},
								{
									Name:  "EMQX_NODE__ROLE",
									Value: "replicant",
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
									Value: `"/opt/emqx/data/bootstrap_api_key"`,
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
									Name:      instance.ReplicantNamespacedName().Name + "-log",
									MountPath: "/opt/emqx/log",
								},
								{
									Name:      instance.ReplicantNamespacedName().Name + "-data",
									MountPath: "/opt/emqx/data",
								},
							}, instance.Spec.ReplicantTemplate.Spec.ExtraVolumeMounts...),
						},
					}, instance.Spec.ReplicantTemplate.Spec.ExtraContainers...),
					Volumes: append([]corev1.Volume{
						{
							Name: "bootstrap-api-key",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: instance.BootstrapAPIKeyNamespacedName().Name,
								},
							},
						},
						{
							Name: "bootstrap-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: instance.ConfigsNamespacedName().Name,
									},
								},
							},
						},
						{
							Name: instance.ReplicantNamespacedName().Name + "-log",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: instance.ReplicantNamespacedName().Name + "-data",
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
