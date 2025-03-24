package v2beta1

import (
	"context"
	"fmt"
	"strconv"

	emperror "emperror.dev/errors"
	"github.com/cisco-open/k8s-objectmatcher/patch"
	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	config "github.com/emqx/emqx-operator/controllers/apps/v2beta1/config"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addRepl struct {
	*EMQXReconciler
}

func (a *addRepl) reconcile(ctx context.Context, logger logr.Logger, instance *appsv2beta1.EMQX, _ innerReq.RequesterInterface) subResult {
	if instance.Spec.ReplicantTemplate == nil {
		return subResult{}
	}
	if !instance.Status.IsConditionTrue(appsv2beta1.CoreNodesReady) {
		return subResult{}
	}

	preRs := getNewReplicaSet(instance, a.conf)
	preRsHash := preRs.Labels[appsv2beta1.LabelsPodTemplateHashKey]
	updateRs, _, _ := getReplicaSetList(ctx, a.Client, instance)

	patchCalculateFunc := func(storage, new *appsv1.ReplicaSet) *patch.PatchResult {
		if storage == nil {
			return &patch.PatchResult{Patch: []byte("{should create new ReplicaSet}")}
		}
		patchResult, _ := a.Patcher.Calculate(
			storage.DeepCopy(),
			new.DeepCopy(),
			justCheckPodTemplate(),
		)
		return patchResult
	}

	if patchResult := patchCalculateFunc(updateRs, preRs); !patchResult.IsEmpty() {
		//Crete Rs
		logger.Info("got different pod template for EMQX replicant nodes, will create new replicaSet", "replicaSet", klog.KObj(preRs), "patch", string(patchResult.Patch))

		_ = ctrl.SetControllerReference(instance, preRs, a.Scheme)
		if err := a.Handler.Create(ctx, preRs); err != nil {
			if k8sErrors.IsAlreadyExists(emperror.Cause(err)) {
				cond := instance.Status.GetLastTrueCondition()
				if cond != nil && cond.Type != appsv2beta1.Available && cond.Type != appsv2beta1.Ready {
					// Sometimes the updated replicaSet will not be ready, because the EMQX node can not be started.
					// And then we will rollback EMQX CR spec, the EMQX operator controller will create a new replicaSet.
					// But the new replicaSet will be the same as the previous one, so we didn't need to create it, just change the EMQX status.
					if preRsHash == instance.Status.ReplicantNodesStatus.CurrentRevision {
						_ = a.updateEMQXStatus(ctx, instance, "RevertReplicaSet", "Revert to current replicaSet", preRsHash)
						return subResult{}
					}
				}
				if instance.Status.ReplicantNodesStatus.CollisionCount == nil {
					instance.Status.ReplicantNodesStatus.CollisionCount = ptr.To(int32(0))
				}
				*instance.Status.ReplicantNodesStatus.CollisionCount++
				_ = a.Client.Status().Update(ctx, instance)
				return subResult{result: ctrl.Result{Requeue: true}}
			}
			return subResult{err: emperror.Wrap(err, "failed to create replicaSet")}
		}
		_ = a.updateEMQXStatus(ctx, instance, "CreateReplicaSet", "Create new replicaSet", preRsHash)
		return subResult{}
	}

	preRs.ObjectMeta = updateRs.DeepCopy().ObjectMeta
	preRs.Spec.Template.ObjectMeta = updateRs.DeepCopy().Spec.Template.ObjectMeta
	preRs.Spec.Selector = updateRs.DeepCopy().Spec.Selector
	if patchResult, _ := a.Patcher.Calculate(
		updateRs.DeepCopy(),
		preRs.DeepCopy(),
		patch.IgnoreStatusFields(),
		patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(),
	); !patchResult.IsEmpty() {
		// Update replicaSet
		logger.Info("got different replicaSet for EMQX replicant nodes, will update replicaSet", "replicaSet", klog.KObj(preRs), "patch", string(patchResult.Patch))
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			storage := &appsv1.ReplicaSet{}
			_ = a.Client.Get(ctx, client.ObjectKeyFromObject(preRs), storage)
			preRs.ResourceVersion = storage.ResourceVersion
			return a.Handler.Update(ctx, preRs)
		}); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to update replicaSet")}
		}
		_ = a.updateEMQXStatus(ctx, instance, "UpdateReplicaSet", "Update exist replicaSet", preRsHash)
	}
	return subResult{}
}

func (a *addRepl) updateEMQXStatus(ctx context.Context, instance *appsv2beta1.EMQX, reason, message, podTemplateHash string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_ = a.Client.Get(ctx, client.ObjectKeyFromObject(instance), instance)
		instance.Status.SetCondition(metav1.Condition{
			Type:    appsv2beta1.ReplicantNodesProgressing,
			Status:  metav1.ConditionTrue,
			Reason:  reason,
			Message: message,
		})
		instance.Status.RemoveCondition(appsv2beta1.Ready)
		instance.Status.RemoveCondition(appsv2beta1.Available)
		instance.Status.RemoveCondition(appsv2beta1.ReplicantNodesReady)
		instance.Status.ReplicantNodesStatus.UpdateRevision = podTemplateHash
		return a.Client.Status().Update(ctx, instance)
	})
}

func getNewReplicaSet(instance *appsv2beta1.EMQX, conf *config.Conf) *appsv1.ReplicaSet {
	svcPorts := conf.GetDashboardServicePort()
	preRs := generateReplicaSet(instance)
	podTemplateSpecHash := computeHash(preRs.Spec.Template.DeepCopy(), instance.Status.ReplicantNodesStatus.CollisionCount)
	preRs.Name = preRs.Name + "-" + podTemplateSpecHash
	preRs.Labels = appsv2beta1.CloneAndAddLabel(preRs.Labels, appsv2beta1.LabelsPodTemplateHashKey, podTemplateSpecHash)
	preRs.Spec.Selector = appsv2beta1.CloneSelectorAndAddLabel(preRs.Spec.Selector, appsv2beta1.LabelsPodTemplateHashKey, podTemplateSpecHash)
	preRs.Spec.Template.Labels = appsv2beta1.CloneAndAddLabel(preRs.Spec.Template.Labels, appsv2beta1.LabelsPodTemplateHashKey, podTemplateSpecHash)
	preRs.Spec.Template.Spec.Containers[0].Ports = appsv2beta1.MergeContainerPorts(
		preRs.Spec.Template.Spec.Containers[0].Ports,
		appsv2beta1.TransServicePortsToContainerPorts(svcPorts),
	)
	for _, p := range preRs.Spec.Template.Spec.Containers[0].Ports {
		if p.Name == "dashboard" {
			preRs.Spec.Template.Spec.Containers[0].Env = append([]corev1.EnvVar{
				{Name: "EMQX_DASHBOARD__LISTENERS__HTTP__BIND", Value: strconv.Itoa(int(p.ContainerPort))},
			}, preRs.Spec.Template.Spec.Containers[0].Env...)
		}
		if p.Name == "dashboard-https" {
			preRs.Spec.Template.Spec.Containers[0].Env = append([]corev1.EnvVar{
				{Name: "EMQX_DASHBOARD__LISTENERS__HTTPS__BIND", Value: strconv.Itoa(int(p.ContainerPort))},
			}, preRs.Spec.Template.Spec.Containers[0].Env...)
		}
	}

	return preRs
}

func generateReplicaSet(instance *appsv2beta1.EMQX) *appsv1.ReplicaSet {
	labels := appsv2beta1.CloneAndMergeMap(
		appsv2beta1.DefaultReplicantLabels(instance),
		instance.Spec.ReplicantTemplate.Labels,
	)

	// Add a PreStop hook to leave the cluster when the pod is asked to stop.
	// This is especially important when DS Raft is enabled, otherwise there will be a
	// lot of leftover records in the DS cluster metadata.
	lifecycle := instance.Spec.ReplicantTemplate.Spec.Lifecycle
	if lifecycle == nil {
		lifecycle = &corev1.Lifecycle{}
	} else {
		lifecycle = lifecycle.DeepCopy()
	}
	lifecycle.PreStop = &corev1.LifecycleHandler{
		Exec: &corev1.ExecAction{
			Command: []string{"/bin/sh", "-c", "emqx ctl cluster leave"},
		},
	}

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
					Tolerations: append(
						instance.Spec.ReplicantTemplate.Spec.Tolerations,
						// TODO: just for compatible with old version, will remove in future
						instance.Spec.ReplicantTemplate.Spec.ToleRations...,
					),
					TopologySpreadConstraints: instance.Spec.CoreTemplate.Spec.TopologySpreadConstraints,
					NodeName:                  instance.Spec.ReplicantTemplate.Spec.NodeName,
					NodeSelector:              instance.Spec.ReplicantTemplate.Spec.NodeSelector,
					InitContainers:            instance.Spec.ReplicantTemplate.Spec.InitContainers,
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
							Lifecycle:       lifecycle,
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
