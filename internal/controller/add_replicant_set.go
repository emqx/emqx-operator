package controller

import (
	"fmt"
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
	if instance.Spec.ReplicantTemplate == nil {
		return subResult{}
	}

	// Core nodes are still spinning up, wait for them to be ready.
	if !r.state.areCoresReady(instance) {
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
				if !instance.Status.IsConditionTrue(crdv2.Ready) {
					// The updated replicaSet may not be ready because the EMQX node can not be started.
					// If the user reverts the CR spec, the desired RS matches the current revision —
					// just update the status instead of creating a duplicate.
					if rsHash == instance.Status.ReplicantNodesStatus.CurrentRevision {
						updateResult := a.updateEMQXStatus(r, instance, rsHash)
						return subResult{err: updateResult}
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
		updateResult := a.updateEMQXStatus(r, instance, rsHash)
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
		updateResult := a.updateEMQXStatus(r, instance, rsHash)
		return subResult{err: updateResult}
	}
	return subResult{}
}

func (a *addReplicantSet) updateEMQXStatus(r *reconcileRound, instance *crdv2.EMQX, podTemplateHash string) error {
	instance.Status.ReplicantNodesStatus.UpdateRevision = podTemplateHash
	forceReplicantNodesProgressing(instance)
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
	template := instance.Spec.ReplicantTemplate

	// Add a PreStop hook to leave the cluster when the pod is asked to stop.
	// This is especially important when DS Raft is enabled, otherwise there will be a
	// lot of leftover records in the DS cluster metadata.
	lifecycle := &corev1.Lifecycle{}
	if template.Spec.Lifecycle != nil {
		lifecycle = template.Spec.Lifecycle.DeepCopy()
	}
	lifecycle.PreStop = &corev1.LifecycleHandler{
		Exec: &corev1.ExecAction{
			Command: []string{"/bin/sh", "-c", "emqx ctl cluster leave"},
		},
	}

	readinessProbe := resources.EvacuationReadinessProbe()

	// Prefer evacuation-aware probe over older-version defaults.
	if template.Spec.ReadinessProbe != nil {
		if template.Spec.ReadinessProbe.HTTPGet != nil &&
			template.Spec.ReadinessProbe.HTTPGet.Path != "/status" {
			readinessProbe = template.Spec.ReadinessProbe.DeepCopy()
		}
	}

	cookie := resources.Cookie(instance)
	config := resources.EMQXConfig(instance)

	return &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ReplicaSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.Namespace,
			Name:        instance.ReplicantNamespacedName().Name,
			Annotations: util.CloneAnnotations(template.Annotations),
			Labels:      replicaSetLabels(instance),
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: template.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: replicaSetLabels(instance),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: util.CloneAnnotations(template.Annotations),
					Labels:      replicaSetLabels(instance),
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets:          instance.Spec.ImagePullSecrets,
					ServiceAccountName:        instance.Spec.ServiceAccountName,
					SecurityContext:           template.Spec.PodSecurityContext,
					Affinity:                  template.Spec.Affinity,
					Tolerations:               template.Spec.Tolerations,
					TopologySpreadConstraints: template.Spec.TopologySpreadConstraints,
					NodeName:                  template.Spec.NodeName,
					NodeSelector:              template.Spec.NodeSelector,
					InitContainers:            template.Spec.InitContainers,
					Containers: append([]corev1.Container{
						{
							Name:            crdv2.DefaultContainerName,
							Image:           instance.Spec.Image,
							ImagePullPolicy: instance.Spec.ImagePullPolicy,
							Command:         template.Spec.Command,
							Args:            template.Spec.Args,
							Ports:           template.Spec.Ports,
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
								cookie.EnvVar(),
							}, template.Spec.Env...),
							EnvFrom:         template.Spec.EnvFrom,
							Resources:       template.Spec.Resources,
							SecurityContext: template.Spec.ContainerSecurityContext,
							LivenessProbe:   template.Spec.LivenessProbe,
							ReadinessProbe:  readinessProbe,
							StartupProbe:    template.Spec.StartupProbe,
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
								template.Spec.ExtraVolumeMounts,
							),
						},
					}, template.Spec.ExtraContainers...),
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
					}, template.Spec.ExtraVolumes...),
				},
			},
		},
	}
}

// Combine instance labels, replicant labels and template labels.
func replicaSetLabels(instance *crdv2.EMQX) map[string]string {
	return instance.DefaultLabelsWith(crdv2.ReplicantLabels(), instance.Spec.ReplicantTemplate.Labels)
}
