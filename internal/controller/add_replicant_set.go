package controller

import (
	"slices"
	"time"

	emperror "emperror.dev/errors"
	"github.com/cisco-open/k8s-objectmatcher/patch"
	crd "github.com/emqx/emqx-operator/api/v3beta1"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	resources "github.com/emqx/emqx-operator/internal/controller/resources"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
)

type addReplicantSet struct {
	*EMQXReconciler
}

func (a *addReplicantSet) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	// Cluster w/o replicants, skip this step.
	if !instance.Spec.HasReplicants() {
		return subResult{}
	}

	// Core nodes are still spinning up, wait for them to be ready.
	coreSet := r.state.coreSet()
	if coreSet == nil || coreSet.Status.AvailableReplicas == 0 {
		return reconcilePostpone()
	}

	// Postpone until at least one core of newest revision.
	// If there's a rolling update involving version upgrade, replicants should be
	// able to connect to at least one core.
	if r.state.numCoresRevision(coreSet.Status.UpdateRevision) == 0 {
		return reconcilePostpone()
	}

	rs := newReplicaSet(instance, r.conf)
	rsHash := rs.Labels[crd.LabelPodTemplateHash]

	needCreate := false
	rollingUpdate := false
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
			rollingUpdate = true
		}
	}

	if needCreate {
		_ = ctrl.SetControllerReference(instance, rs, a.Scheme)
		// New ReplicaSet is created at 0 replicas; `syncReplicantSets` scales it within maxSurge budget.
		if rollingUpdate {
			rs.Spec.Replicas = ptr.To(int32(0))
		}
		if err := a.Create(r.ctx, rs); err != nil {
			if k8sErrors.IsAlreadyExists(emperror.Cause(err)) {
				if !instance.Status.IsConditionTrue(crd.Ready) {
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
				return reconcileRequeue()
			}
			return reconcileError(emperror.Wrap(err, "failed to create replicaSet"))
		}
		updateResult := a.updateEMQXStatus(r, instance, rsHash)
		return subResult{err: updateResult, immediateResult: &ctrl.Result{RequeueAfter: time.Second}}
	}

	rs.ObjectMeta = updateReplicantSet.ObjectMeta
	rs.Spec.Template.ObjectMeta = updateReplicantSet.Spec.Template.ObjectMeta
	rs.Spec.Selector = updateReplicantSet.Spec.Selector
	patchResult, _ := a.Patcher.Calculate(
		updateReplicantSet,
		rs,
		patch.IgnoreStatusFields(),
		patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(),
		// Ignore if number of replicas has changed.
		// Reconciler `syncReplicantSets` will handle scaling of the statefulSet.
		ignoreField([]string{"spec", "replicas"}),
	)
	if !patchResult.IsEmpty() {
		// Update replicaSet
		r.log.Info("updating replicaSet",
			"replicaSet", klog.KObj(rs),
			"reason", "replicaSet has changed",
			"patch", string(patchResult.Patch),
		)
		// NOTE
		// Conflicts are expected as ReplicaSet contoller may act concurrently on the resource.
		// Conflicts are handled on `EMQXReconciler` level.
		err := a.Update(r.ctx, rs)
		if err != nil {
			return reconcileError(emperror.Wrap(err, "failed to update replicaSet"))
		}
		updateResult := a.updateEMQXStatus(r, instance, rsHash)
		return subResult{err: updateResult, immediateResult: &ctrl.Result{RequeueAfter: time.Second}}
	}

	return subResult{}
}

func (a *addReplicantSet) updateEMQXStatus(r *reconcileRound, instance *crd.EMQX, podTemplateHash string) error {
	instance.Status.ReplicantNodesStatus.UpdateRevision = podTemplateHash
	forceReplicantNodesProgressing(instance)
	return a.Client.Status().Update(r.ctx, instance)
}

func newReplicaSet(instance *crd.EMQX, conf *config.EMQX) *appsv1.ReplicaSet {
	rs := generateReplicaSet(instance)
	podTemplateHash := computeHash(rs.Spec.Template.DeepCopy(), instance.Status.ReplicantNodesStatus.CollisionCount)
	rs.Name = rs.Name + "-" + podTemplateHash
	rs.Labels[crd.LabelPodTemplateHash] = podTemplateHash
	rs.Spec.Template.Labels[crd.LabelPodTemplateHash] = podTemplateHash
	rs.Spec.Selector = util.CloneSelectorAndAddLabel(rs.Spec.Selector, crd.LabelPodTemplateHash, podTemplateHash)
	rs.Spec.Template.Spec.Containers[0].Ports = util.MergeContainerPorts(
		rs.Spec.Template.Spec.Containers[0].Ports,
		util.MapServicePortsToContainerPorts(conf.GetDashboardServicePorts()),
	)
	return rs
}

func generateReplicaSet(instance *crd.EMQX) *appsv1.ReplicaSet {
	template := instance.Spec.ReplicantTemplate

	// Add a PreStop hook to leave the cluster when the pod is asked to stop.
	lifecycle := &corev1.Lifecycle{}
	if template.Spec.Lifecycle != nil {
		lifecycle = template.Spec.Lifecycle.DeepCopy()
	}
	lifecycle.PreStop = &corev1.LifecycleHandler{
		Exec: &corev1.ExecAction{
			Command: []string{"/bin/sh", "-c", "emqx ctl cluster leave"},
		},
	}

	// Prefer evacuation-aware probe over older-version defaults.
	readinessProbe := resources.EvacuationReadinessProbe()
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
			Replicas:        template.Spec.Replicas,
			MinReadySeconds: instance.Spec.ReplicantTemplate.Spec.MinReadySeconds,
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
					DNSConfig:                 template.Spec.DNSConfig,
					InitContainers:            template.Spec.InitContainers,
					Containers: append([]corev1.Container{
						{
							Name:            crd.DefaultContainerName,
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
									Value: crd.RoleReplicant,
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
func replicaSetLabels(instance *crd.EMQX) map[string]string {
	return instance.DefaultLabelsWith(crd.ReplicantLabels(), instance.Spec.ReplicantTemplate.Labels)
}
