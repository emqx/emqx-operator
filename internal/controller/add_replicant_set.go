package controller

import (
	"slices"

	emperror "emperror.dev/errors"
	"github.com/cisco-open/k8s-objectmatcher/patch"
	crdv2 "github.com/emqx/emqx-operator/api/v2"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	resources "github.com/emqx/emqx-operator/internal/controller/resources"
	util "github.com/emqx/emqx-operator/internal/controller/util"
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

type addReplicantSet struct {
	*EMQXReconciler
}

func (a *addReplicantSet) reconcile(r *reconcileRound, instance *crdv2.EMQX) subResult {
	// Cluster w/o replicants, skip this step.
	if !instance.Spec.HasReplicants() {
		return subResult{}
	}

	// Core nodes are still spinning up, wait for them to be ready.
	if !instance.Status.IsConditionTrue(crdv2.CoreNodesReady) {
		return subResult{}
	}

	rs := newReplicaSet(instance, r.conf)
	rsHash := rs.Labels[crdv2.LabelPodTemplateHash]

	needCreate := false
	updateReplicantSet := r.state.updateReplicantSet(instance)
	if updateReplicantSet == nil {
		r.log.Info("creating new replicaSet",
			"replicaSet", klog.KObj(rs),
			"reason", "no existing replicaSet",
		)
		needCreate = true
	} else {
		patchResult, _ := a.Patcher.Calculate(updateReplicantSet, rs, justCheckPodTemplate())
		if !patchResult.IsEmpty() {
			r.log.Info("creating new replicaSet",
				"replicaSet", klog.KObj(rs),
				"reason", "pod template has changed",
				"patch", string(patchResult.Patch),
			)
			needCreate = true
		}
	}

	if needCreate {
		_ = ctrl.SetControllerReference(instance, rs, a.Scheme)
		if err := a.Handler.Create(r.ctx, rs); err != nil {
			if k8sErrors.IsAlreadyExists(emperror.Cause(err)) {
				cond := instance.Status.GetLastTrueCondition()
				if cond != nil && cond.Type != crdv2.Available && cond.Type != crdv2.Ready {
					// Sometimes the updated replicaSet will not be ready, because the EMQX node can not be started.
					// And then we will rollback EMQX CR spec, the EMQX operator controller will create a new replicaSet.
					// But the new replicaSet will be the same as the previous one, so we didn't need to create it, just change the EMQX status.
					if rsHash == instance.Status.ReplicantNodesStatus.CurrentRevision {
						_ = a.updateEMQXStatus(r, instance, "RevertReplicaSet", rsHash)
						return subResult{}
					}
				}
				if instance.Status.ReplicantNodesStatus.CollisionCount == nil {
					instance.Status.ReplicantNodesStatus.CollisionCount = ptr.To(int32(0))
				}
				*instance.Status.ReplicantNodesStatus.CollisionCount++
				_ = a.Client.Status().Update(r.ctx, instance)
				return subResult{result: ctrl.Result{Requeue: true}}
			}
			return subResult{err: emperror.Wrap(err, "failed to create replicaSet")}
		}
		updateResult := a.updateEMQXStatus(r, instance, "CreateReplicaSet", rsHash)
		return subResult{err: updateResult}
	}

	rs.ObjectMeta = updateReplicantSet.ObjectMeta
	rs.Spec.Template.ObjectMeta = updateReplicantSet.Spec.Template.ObjectMeta
	rs.Spec.Selector = updateReplicantSet.Spec.Selector
	if patchResult, _ := a.Patcher.Calculate(
		updateReplicantSet,
		rs,
		patch.IgnoreStatusFields(),
		patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(),
	); !patchResult.IsEmpty() {
		// Update replicaSet
		r.log.Info("updating replicaSet",
			"replicaSet", klog.KObj(rs),
			"reason", "replicaSet has changed",
			"patch", string(patchResult.Patch),
		)
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			storage := &appsv1.ReplicaSet{}
			_ = a.Client.Get(r.ctx, client.ObjectKeyFromObject(rs), storage)
			rs.ResourceVersion = storage.ResourceVersion
			return a.Handler.Update(r.ctx, rs)
		}); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to update replicaSet")}
		}
		updateResult := a.updateEMQXStatus(r, instance, "UpdateReplicaSet", rsHash)
		return subResult{err: updateResult}
	}
	return subResult{}
}

func (a *addReplicantSet) updateEMQXStatus(r *reconcileRound, instance *crdv2.EMQX, reason, podTemplateHash string) error {
	instance.Status.ResetConditions(reason)
	instance.Status.ReplicantNodesStatus.UpdateRevision = podTemplateHash
	return a.Client.Status().Update(r.ctx, instance)
}

func newReplicaSet(instance *crdv2.EMQX, conf *config.EMQX) *appsv1.ReplicaSet {
	rs := generateReplicaSet(instance)
	podTemplateHash := computeHash(rs.Spec.Template.DeepCopy(), instance.Status.ReplicantNodesStatus.CollisionCount)
	rs.Name = rs.Name + "-" + podTemplateHash
	rs.Labels[crdv2.LabelPodTemplateHash] = podTemplateHash
	rs.Spec.Template.Labels[crdv2.LabelPodTemplateHash] = podTemplateHash
	rs.Spec.Selector = util.CloneSelectorAndAddLabel(rs.Spec.Selector, crdv2.LabelPodTemplateHash, podTemplateHash)
	rs.Spec.Template.Spec.Containers[0].Ports = util.MergeContainerPorts(
		rs.Spec.Template.Spec.Containers[0].Ports,
		util.MapServicePortsToContainerPorts(conf.GetDashboardServicePorts()),
	)
	return rs
}

func generateReplicaSet(instance *crdv2.EMQX) *appsv1.ReplicaSet {
	cookie := resources.Cookie(instance)
	config := resources.EMQXConfig(instance)

	// Replicants should leave the cluster while still alive. Once stopped, EMQX
	// normally no longer sees them as cluster nodes to force-leave.
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
			Labels:      replicaSetLabels(instance),
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: instance.Spec.ReplicantTemplate.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: replicaSetLabels(instance),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: instance.Spec.ReplicantTemplate.DeepCopy().Annotations,
					Labels:      replicaSetLabels(instance),
				},
				Spec: corev1.PodSpec{
					ReadinessGates: []corev1.PodReadinessGate{
						{
							ConditionType: crdv2.PodOnServing,
						},
					},
					ImagePullSecrets:          instance.Spec.ImagePullSecrets,
					ServiceAccountName:        instance.Spec.ServiceAccountName,
					SecurityContext:           instance.Spec.ReplicantTemplate.Spec.PodSecurityContext,
					Affinity:                  instance.Spec.ReplicantTemplate.Spec.Affinity,
					Tolerations:               instance.Spec.ReplicantTemplate.Spec.Tolerations,
					TopologySpreadConstraints: instance.Spec.ReplicantTemplate.Spec.TopologySpreadConstraints,
					NodeName:                  instance.Spec.ReplicantTemplate.Spec.NodeName,
					NodeSelector:              instance.Spec.ReplicantTemplate.Spec.NodeSelector,
					InitContainers:            instance.Spec.ReplicantTemplate.Spec.InitContainers,
					Containers: append([]corev1.Container{
						{
							Name:            crdv2.DefaultContainerName,
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
									Value: clusterDNSName(instance),
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
								cookie.EnvVar(),
							}, instance.Spec.ReplicantTemplate.Spec.Env...),
							EnvFrom:         instance.Spec.ReplicantTemplate.Spec.EnvFrom,
							Resources:       instance.Spec.ReplicantTemplate.Spec.Resources,
							SecurityContext: instance.Spec.ReplicantTemplate.Spec.ContainerSecurityContext,
							LivenessProbe:   instance.Spec.ReplicantTemplate.Spec.LivenessProbe,
							ReadinessProbe:  instance.Spec.ReplicantTemplate.Spec.ReadinessProbe,
							StartupProbe:    instance.Spec.ReplicantTemplate.Spec.StartupProbe,
							Lifecycle:       lifecycle,
							VolumeMounts: slices.Concat(
								[]corev1.VolumeMount{
									{
										Name:      instance.ReplicantName() + "-log",
										MountPath: "/opt/emqx/log",
									},
									{
										Name:      instance.ReplicantName() + "-data",
										MountPath: "/opt/emqx/data",
									},
								},
								config.VolumeMounts(),
								instance.Spec.ReplicantTemplate.Spec.ExtraVolumeMounts,
							),
						},
					}, instance.Spec.ReplicantTemplate.Spec.ExtraContainers...),
					Volumes: append([]corev1.Volume{
						config.Volume(),
						{
							Name: instance.ReplicantName() + "-log",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: instance.ReplicantName() + "-data",
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

// Combine instance labels, replicant labels and template labels.
func replicaSetLabels(instance *crdv2.EMQX) map[string]string {
	return instance.DefaultLabelsWith(crdv2.ReplicantLabels(), instance.Spec.ReplicantTemplate.Labels)
}
