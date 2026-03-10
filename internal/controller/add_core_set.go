package controller

import (
	"fmt"
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
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

type addCoreSet struct {
	*EMQXReconciler
}

func (a *addCoreSet) reconcile(r *reconcileRound, instance *crdv2.EMQX) subResult {
	sts := newStatefulSet(instance, r.conf)

	existing := r.state.coreSet()
	if existing == nil {
		// No StatefulSet exists yet.
		r.log.Info("creating statefulSet",
			"statefulSet", klog.KObj(sts),
			"reason", "no existing statefulSet",
		)
		_ = ctrl.SetControllerReference(instance, sts, a.Scheme)
		if err := a.Handler.Create(r.ctx, sts); err != nil {
			if k8sErrors.IsAlreadyExists(emperror.Cause(err)) {
				return subResult{result: ctrl.Result{Requeue: true}}
			}
			return subResult{err: emperror.Wrap(err, "failed to create statefulSet")}
		}
		return subResult{}
	}

	// StatefulSet exists.
	// Update it in place if the spec has changed.
	// With OnDelete strategy, updating the spec does not restart pods.
	sts.ObjectMeta = existing.ObjectMeta
	sts.Spec.Template.ObjectMeta = existing.Spec.Template.ObjectMeta
	sts.Spec.Selector = existing.Spec.Selector
	patchResult, _ := a.Patcher.Calculate(
		existing,
		sts,
		patch.IgnoreStatusFields(),
		patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(),
		// Ignore if number of replicas has changed.
		// Reconciler `syncCoreSets` will handle scaling of the statefulSet.
		ignoreStatefulSetReplicas(),
	)
	if !patchResult.IsEmpty() {
		r.log.Info("updating statefulSet",
			"statefulSet", klog.KObj(sts),
			"reason", "spec has changed",
			"patch", string(patchResult.Patch),
		)
		err := a.Handler.Update(r.ctx, sts)
		if err != nil {
			return subResult{err: emperror.Wrap(err, "failed to update statefulSet")}
		}
		// Reset conditions: pods have not been updated yet and the cluster needs
		// a rolling update, so it's no longer Ready / CoreNodesReady.
		instance.Status.ResetConditions("UpdateStatefulSet")
		err = a.Client.Status().Update(r.ctx, instance)
		if err != nil {
			return subResult{err: emperror.Wrap(err, "failed to update status after statefulSet update")}
		}
	}
	return subResult{}
}

func newStatefulSet(instance *crdv2.EMQX, conf *config.EMQX) *appsv1.StatefulSet {
	sts := generateStatefulSet(instance)
	sts.Spec.Template.Spec.Containers[0].Ports = util.MergeContainerPorts(
		sts.Spec.Template.Spec.Containers[0].Ports,
		util.MapServicePortsToContainerPorts(conf.GetDashboardServicePorts()),
	)
	return sts
}

func generateStatefulSet(instance *crdv2.EMQX) *appsv1.StatefulSet {
	// Add a PreStop hook to leave the cluster when the pod is asked to stop.
	// This is especially important when DS Raft is enabled, otherwise there will be a
	// lot of leftover records in the DS cluster metadata.
	lifecycle := instance.Spec.CoreTemplate.Spec.Lifecycle
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

	cookie := resources.Cookie(instance)
	bootstrapAPIKeys := resources.BootstrapAPIKey(instance)
	config := resources.EMQXConfig(instance)

	// Use OnDelete update strategy so the operator controls pod replacement.
	updateStrategy := appsv1.StatefulSetUpdateStrategy{
		Type: appsv1.OnDeleteStatefulSetStrategyType,
	}

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
			ServiceName:         instance.HeadlessServiceNamespacedName().Name,
			Replicas:            instance.Spec.CoreTemplate.Spec.Replicas,
			UpdateStrategy:      updateStrategy,
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Selector: &metav1.LabelSelector{
				MatchLabels: statefulSetLabels(instance),
			},
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
								cookie.EnvVar(),
								bootstrapAPIKeys.EnvVar(),
							}, instance.Spec.CoreTemplate.Spec.Env...),
							EnvFrom:         instance.Spec.CoreTemplate.Spec.EnvFrom,
							Resources:       instance.Spec.CoreTemplate.Spec.Resources,
							SecurityContext: instance.Spec.CoreTemplate.Spec.ContainerSecurityContext,
							LivenessProbe:   instance.Spec.CoreTemplate.Spec.LivenessProbe,
							ReadinessProbe:  instance.Spec.CoreTemplate.Spec.ReadinessProbe,
							StartupProbe:    instance.Spec.CoreTemplate.Spec.StartupProbe,
							Lifecycle:       lifecycle,
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
