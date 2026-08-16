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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

type addCoreSet struct {
	*EMQXReconciler
}

func (a *addCoreSet) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	existing := r.state.coreSet()
	coreSet := newStatefulSet(instance)
	_ = ctrl.SetControllerReference(instance, coreSet, a.Scheme)

	if existing == nil {
		// No StatefulSet exists yet.
		r.log.Info("creating statefulSet",
			"statefulSet", klog.KObj(coreSet),
			"reason", "no existing statefulSet",
		)
		util.AttachAnnotations(coreSet, initialCoreSetRetirementState(coreSet).annotations())
		if err := a.Create(r.ctx, coreSet); err != nil {
			if k8sErrors.IsAlreadyExists(emperror.Cause(err)) {
				return reconcileRequeue()
			}
			return reconcileError(emperror.Wrap(err, "failed to create statefulSet"))
		}
		// Force Requeue to give StatefulSet controller time to reflect status.
		return reconcileRequeueAfter(time.Second)
	} else {
		// StatefulSet already exists.
		// Preserve retirement annotations.
		util.UnsetAnnotations(coreSet, coreSetRetirementAnnotations...)
		util.AttachAnnotations(coreSet, util.PeekAnnotations(existing, coreSetRetirementAnnotations...))
		// Preserve the live replica count.
		// This update should not bypass syncCoreSet reusable-ordinal admission check.
		coreSet.Spec.Replicas = existing.Spec.Replicas
	}

	// StatefulSet exists.
	// Update it in place if the spec has changed.
	// With OnDelete strategy, updating the spec does not restart pods.
	patchResult, _ := a.Patcher.Calculate(
		existing,
		coreSet,
		patch.IgnoreStatusFields(),
		patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(),
		// Ignore if number of replicas has changed.
		// Reconciler `syncCoreSet` will handle scaling of the statefulSet.
		ignoreField([]string{"spec", "replicas"}),
	)
	if !patchResult.IsEmpty() {
		r.log.Info("updating statefulSet",
			"statefulSet", klog.KObj(coreSet),
			"reason", "spec has changed",
			"patch", string(patchResult.Patch),
		)
		// NOTE
		// Conflicts are expected as StatefulSet contoller may act concurrently on the resource.
		// Conflicts are handled on `EMQXReconciler` level.
		err := a.Update(r.ctx, coreSet)
		if err != nil {
			return reconcileError(emperror.Wrap(err, "failed to update statefulSet"))
		}
		forceCoreNodesProgressing(instance)
		updateResult := a.Client.Status().Update(r.ctx, instance)
		// Force Requeue to give StatefulSet controller time to reflect status.
		return subResult{err: updateResult, immediateResult: &ctrl.Result{RequeueAfter: time.Second}}
	}

	return subResult{}
}

func newStatefulSet(instance *crd.EMQX) *appsv1.StatefulSet {
	sts := generateStatefulSet(instance)
	attachRestartConfigHash(instance, &sts.Spec.Template)
	sts.Spec.Template.Spec.Containers[0].Ports = util.AppendMissingContainerPorts(
		sts.Spec.Template.Spec.Containers[0].Ports,
		util.MapServicePortsToContainerPorts(config.DashboardServicePorts(instance.Spec.Config.Roots)),
	)
	return sts
}

func generateStatefulSet(instance *crd.EMQX) *appsv1.StatefulSet {
	template := &instance.Spec.CoreTemplate

	cookie := resources.Cookie(instance)
	bootstrapAPIKeys := resources.BootstrapAPIKey(instance)
	config := resources.EMQXConfig(instance)

	// Use OnDelete update strategy so the operator controls pod replacement.
	updateStrategy := appsv1.StatefulSetUpdateStrategy{
		Type: appsv1.OnDeleteStatefulSetStrategyType,
	}

	// Requires K8s >= 1.27 and the StatefulSetAutoDeletePVC feature gate (stable since K8s 1.32).
	pvcRetentionPolicy := appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
		WhenScaled:  appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
		WhenDeleted: appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
	}

	readinessProbe := resources.EvacuationReadinessProbe()

	// Prefer evacuation-aware probe over older-version defaults.
	if template.Spec.ReadinessProbe != nil {
		if template.Spec.ReadinessProbe.HTTPGet != nil &&
			template.Spec.ReadinessProbe.HTTPGet.Path != "/status" {
			readinessProbe = template.Spec.ReadinessProbe.DeepCopy()
		}
	}

	// User-provided env, volumes, and mounts are appended verbatim. Collisions can
	// override Operator-owned environment variables (including the node cookie),
	// make the Pod invalid through duplicate names, or shadow bootstrap files.
	sts := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.Namespace,
			Name:        instance.CoreNamespacedName().Name,
			Annotations: util.CloneAnnotations(template.Annotations),
			Labels:      statefulSetLabels(instance),
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName:                          instance.HeadlessServiceNamespacedName().Name,
			Replicas:                             template.Spec.Replicas,
			MinReadySeconds:                      instance.Spec.CoreTemplate.Spec.MinReadySeconds,
			UpdateStrategy:                       updateStrategy,
			PodManagementPolicy:                  appsv1.ParallelPodManagement,
			PersistentVolumeClaimRetentionPolicy: &pvcRetentionPolicy,
			Selector: &metav1.LabelSelector{
				MatchLabels: statefulSetSelectorLabels(instance),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: util.CloneAnnotations(template.Annotations),
					Labels:      statefulSetLabels(instance),
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
									Value: crd.RoleCore,
								},
								cookie.EnvVar(),
								bootstrapAPIKeys.EnvVar(),
							}, template.Spec.Env...),
							EnvFrom:         template.Spec.EnvFrom,
							Resources:       template.Spec.Resources,
							SecurityContext: template.Spec.ContainerSecurityContext,
							LivenessProbe:   template.Spec.LivenessProbe,
							ReadinessProbe:  readinessProbe,
							StartupProbe:    template.Spec.StartupProbe,
							Lifecycle:       template.Spec.Lifecycle,
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
								template.Spec.ExtraVolumeMounts,
							),
						},
					}, template.Spec.ExtraContainers...),
					Volumes: append([]corev1.Volume{
						config.Volume(),
						bootstrapAPIKeys.Volume(),
						{
							Name: instance.CoreName() + "-log",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					}, template.Spec.ExtraVolumes...),
				},
			},
		},
	}

	sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instance.CoreNamespacedName().Name + "-data",
				Namespace: instance.Namespace,
				// TODO
				// Labels from the core template are currently not attached to PVCs.
				// This is deliberate, as it simplifies StatefulSet management.
				// If this is needed, care must be taken to propagate the core template
				// changes correctly: updating PVC template (if only just labels) inside
				// the StatefulSet spec is explicitly forbidden by K8S API server.
				Labels: instance.DefaultLabelsWith(crd.CoreLabels()),
			},
			Spec: coreDataVolumeClaimSpec(template),
		},
	}

	return sts
}

// Combine instance labels, core labels and template labels.
func statefulSetLabels(instance *crd.EMQX) map[string]string {
	return instance.DefaultLabelsWith(crd.CoreLabels(), instance.Spec.CoreTemplate.Labels)
}

// Combine just instance labels and core labels.
// Should be stable across EMQX spec changes.
func statefulSetSelectorLabels(instance *crd.EMQX) map[string]string {
	return instance.DefaultLabelsWith(crd.CoreLabels())
}

func coreDataVolumeClaimSpec(template *crd.EMQXCoreTemplate) corev1.PersistentVolumeClaimSpec {
	spec := template.Spec.PersistentVolumeClaimSpec.DeepCopy()
	if len(spec.AccessModes) == 0 {
		spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}
	if spec.Resources.Requests == nil {
		spec.Resources.Requests = corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("500Mi")}
	}
	if spec.VolumeMode == nil {
		// https://github.com/cisco-open/k8s-objectmatcher/issues/51
		fs := corev1.PersistentVolumeFilesystem
		spec.VolumeMode = &fs
	}
	return *spec
}
