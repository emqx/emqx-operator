package controller

import (
	"reflect"
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

type addCoreSet struct {
	*EMQXReconciler
}

func (a *addCoreSet) reconcile(r *reconcileRound, instance *crdv2.EMQX) subResult {
	sts := newStatefulSet(instance, r.conf)
	stsHash := sts.Labels[crdv2.LabelPodTemplateHash]

	needCreate := false
	updateCoreSet := r.state.updateCoreSet(instance)
	if updateCoreSet == nil {
		r.log.Info("creating new statefulSet",
			"statefulSet", klog.KObj(sts),
			"reason", "no existing statefulSet",
		)
		needCreate = true
	} else {
		patchResult, _ := a.Patcher.Calculate(updateCoreSet, sts, justCheckPodTemplate())
		if !patchResult.IsEmpty() {
			r.log.Info("creating new statefulSet",
				"statefulSet", klog.KObj(sts),
				"reason", "pod template has changed",
				"patch", string(patchResult.Patch),
			)
			needCreate = true
		}
	}

	if needCreate {
		_ = ctrl.SetControllerReference(instance, sts, a.Scheme)
		if err := a.Handler.Create(r.ctx, sts); err != nil {
			if k8sErrors.IsAlreadyExists(emperror.Cause(err)) {
				cond := instance.Status.GetLastTrueCondition()
				if cond != nil && cond.Type != crdv2.Available && cond.Type != crdv2.Ready {
					// Sometimes the updated statefulSet will not be ready, because the EMQX node can not be started.
					// And then we will rollback EMQX CR spec, the EMQX operator controller will create a new statefulSet.
					// But the new statefulSet will be the same as the previous one, so we didn't need to create it, just change the EMQX status.
					if stsHash == instance.Status.CoreNodesStatus.CurrentRevision {
						_ = a.updateEMQXStatus(r, instance, "RevertStatefulSet", stsHash)
						return subResult{}
					}
				}
				if instance.Status.CoreNodesStatus.CollisionCount == nil {
					instance.Status.CoreNodesStatus.CollisionCount = ptr.To(int32(0))
				}
				*instance.Status.CoreNodesStatus.CollisionCount++
				_ = a.Client.Status().Update(r.ctx, instance)
				return subResult{result: ctrl.Result{Requeue: true}}
			}
			return subResult{err: emperror.Wrap(err, "failed to create statefulSet")}
		}
		updateResult := a.updateEMQXStatus(r, instance, "CreateNewStatefulSet", stsHash)
		return subResult{err: updateResult}
	}

	sts.ObjectMeta = updateCoreSet.ObjectMeta
	sts.Spec.Template.ObjectMeta = updateCoreSet.Spec.Template.ObjectMeta
	sts.Spec.Selector = updateCoreSet.Spec.Selector
	patchResult, _ := a.Patcher.Calculate(
		updateCoreSet,
		sts,
		// Ignore Status fields and VolumeClaimTemplate stuff.
		patch.IgnoreStatusFields(),
		patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(),
		// Ignore if number of replicas has changed.
		// Reconciler `syncCoreSets` will handle scaling up and down of the existing statefulSet.
		ignoreStatefulSetReplicas(),
	)
	if !patchResult.IsEmpty() {
		// Update statefulSet
		r.log.Info("updating statefulSet",
			"statefulSet", klog.KObj(sts),
			"reason", "statefulSet has changed",
			"patch", string(patchResult.Patch),
		)
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			storage := &appsv1.StatefulSet{}
			_ = a.Client.Get(r.ctx, client.ObjectKeyFromObject(sts), storage)
			sts.ResourceVersion = storage.ResourceVersion
			return a.Handler.Update(r.ctx, sts)
		}); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to update statefulSet")}
		}
		updateResult := a.updateEMQXStatus(r, instance, "UpdateStatefulSet", stsHash)
		return subResult{err: updateResult}
	}
	return subResult{}
}

func (a *addCoreSet) updateEMQXStatus(r *reconcileRound, instance *crdv2.EMQX, reason, podTemplateHash string) error {
	instance.Status.ResetConditions(reason)
	instance.Status.CoreNodesStatus.UpdateRevision = podTemplateHash
	return a.Client.Status().Update(r.ctx, instance)
}

func newStatefulSet(instance *crdv2.EMQX, conf *config.EMQX) *appsv1.StatefulSet {
	sts := generateStatefulSet(instance)
	podTemplateHash := computeHash(sts.Spec.Template.DeepCopy(), instance.Status.CoreNodesStatus.CollisionCount)
	sts.Name = sts.Name + "-" + podTemplateHash
	sts.Labels[crdv2.LabelPodTemplateHash] = podTemplateHash
	sts.Spec.Template.Labels[crdv2.LabelPodTemplateHash] = podTemplateHash
	sts.Spec.Selector = util.CloneSelectorAndAddLabel(sts.Spec.Selector, crdv2.LabelPodTemplateHash, podTemplateHash)
	sts.Spec.Template.Spec.Containers[0].Ports = util.MergeContainerPorts(
		sts.Spec.Template.Spec.Containers[0].Ports,
		util.MapServicePortsToContainerPorts(conf.GetDashboardServicePorts()),
	)
	return sts
}

func generateStatefulSet(instance *crdv2.EMQX) *appsv1.StatefulSet {
	cookie := resources.Cookie(instance)
	bootstrapAPIKeys := resources.BootstrapAPIKey(instance)
	config := resources.EMQXConfig(instance)

	sts := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.Namespace,
			Name:        instance.CoreNamespacedName().Name,
			Annotations: instance.Spec.CoreTemplate.DeepCopy().Annotations,
			Labels:      statefulSetLabels(instance),
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: instance.HeadlessServiceNamespacedName().Name,
			Replicas:    instance.Spec.CoreTemplate.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: statefulSetLabels(instance),
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: instance.Spec.CoreTemplate.DeepCopy().Annotations,
					Labels:      statefulSetLabels(instance),
				},
				Spec: corev1.PodSpec{
					ReadinessGates: []corev1.PodReadinessGate{
						{
							ConditionType: crdv2.PodOnServing,
						},
					},
					ImagePullSecrets:          instance.Spec.ImagePullSecrets,
					ServiceAccountName:        instance.Spec.ServiceAccountName,
					SecurityContext:           instance.Spec.CoreTemplate.Spec.PodSecurityContext,
					Affinity:                  instance.Spec.CoreTemplate.Spec.Affinity,
					Tolerations:               instance.Spec.CoreTemplate.Spec.Tolerations,
					TopologySpreadConstraints: instance.Spec.CoreTemplate.Spec.TopologySpreadConstraints,
					NodeName:                  instance.Spec.CoreTemplate.Spec.NodeName,
					NodeSelector:              instance.Spec.CoreTemplate.Spec.NodeSelector,
					InitContainers:            instance.Spec.CoreTemplate.Spec.InitContainers,
					Containers: append([]corev1.Container{
						{
							Name:            crdv2.DefaultContainerName,
							Image:           instance.Spec.Image,
							ImagePullPolicy: instance.Spec.ImagePullPolicy,
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
									Value: clusterDNSName(instance),
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
								cookie.EnvVar(),
								bootstrapAPIKeys.EnvVar(),
							}, instance.Spec.CoreTemplate.Spec.Env...),
							EnvFrom:         instance.Spec.CoreTemplate.Spec.EnvFrom,
							Resources:       instance.Spec.CoreTemplate.Spec.Resources,
							SecurityContext: instance.Spec.CoreTemplate.Spec.ContainerSecurityContext,
							LivenessProbe:   instance.Spec.CoreTemplate.Spec.LivenessProbe,
							ReadinessProbe:  instance.Spec.CoreTemplate.Spec.ReadinessProbe,
							StartupProbe:    instance.Spec.CoreTemplate.Spec.StartupProbe,
							Lifecycle:       instance.Spec.CoreTemplate.Spec.Lifecycle,
							VolumeMounts: slices.Concat(
								[]corev1.VolumeMount{
									{
										Name:      instance.CoreName() + "-log",
										MountPath: "/opt/emqx/log",
									},
									{
										Name:      instance.CoreName() + "-data",
										MountPath: "/opt/emqx/data",
									},
									bootstrapAPIKeys.VolumeMount(),
								},
								config.VolumeMounts(),
								instance.Spec.CoreTemplate.Spec.ExtraVolumeMounts,
							),
						},
					}, instance.Spec.CoreTemplate.Spec.ExtraContainers...),
					Volumes: append([]corev1.Volume{
						config.Volume(),
						bootstrapAPIKeys.Volume(),
						{
							Name: instance.CoreName() + "-log",
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
		volumeClaimTemplates := instance.Spec.CoreTemplate.Spec.VolumeClaimTemplates.DeepCopy()
		if volumeClaimTemplates.VolumeMode == nil {
			// Wait https://github.com/cisco-open/k8s-objectmatcher/issues/51 fixed
			fs := corev1.PersistentVolumeFilesystem
			volumeClaimTemplates.VolumeMode = &fs
		}
		sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instance.CoreNamespacedName().Name + "-data",
					Namespace: instance.Namespace,
					Labels:    statefulSetLabels(instance),
				},
				Spec: *volumeClaimTemplates,
			},
		}
	} else {
		sts.Spec.Template.Spec.Volumes = append([]corev1.Volume{
			{
				Name: instance.CoreNamespacedName().Name + "-data",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		}, sts.Spec.Template.Spec.Volumes...)
	}

	return sts
}

// Combine instance labels, core labels and template labels.
func statefulSetLabels(instance *crdv2.EMQX) map[string]string {
	return instance.DefaultLabelsWith(crdv2.CoreLabels(), instance.Spec.CoreTemplate.Labels)
}
