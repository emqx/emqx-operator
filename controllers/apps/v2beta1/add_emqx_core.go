package v2beta1

import (
	"context"
	"fmt"
	"reflect"
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
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type addCore struct {
	*EMQXReconciler
}

func (a *addCore) reconcile(ctx context.Context, instance *appsv2beta1.EMQX, _ innerReq.RequesterInterface) subResult {
	preSts, err := a.getNewStatefulSet(ctx, instance)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get new statefulSet")}
	}
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
			Type:    appsv2beta1.CoreNodesProgressing,
			Status:  metav1.ConditionTrue,
			Reason:  "CreateNewStatefulSet",
			Message: "Create new statefulSet",
		})
		instance.Status.RemoveCondition(appsv2beta1.Ready)
		instance.Status.RemoveCondition(appsv2beta1.Available)
		instance.Status.RemoveCondition(appsv2beta1.CoreNodesReady)
		instance.Status.CoreNodesStatus.UpdateRevision = preSts.Labels[appsv2beta1.LabelsPodTemplateHashKey]
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
			logger.Info("got different statefulSet for EMQX core nodes, will update statefulSet", "statefulSet", klog.KObj(preSts), "patch", string(patchResult.Patch))

			if err := a.Handler.Update(preSts); err != nil {
				return subResult{err: emperror.Wrap(err, "failed to update statefulSet")}
			}

			instance.Status.SetCondition(metav1.Condition{
				Type:    appsv2beta1.CoreNodesProgressing,
				Status:  metav1.ConditionTrue,
				Reason:  "CreateNewStatefulSet",
				Message: "Create new statefulSet",
			})
			instance.Status.RemoveCondition(appsv2beta1.Ready)
			instance.Status.RemoveCondition(appsv2beta1.Available)
			instance.Status.RemoveCondition(appsv2beta1.CoreNodesReady)
			_ = a.Client.Status().Update(ctx, instance)
		}
	}

	return subResult{}
}

func (a *addCore) getNewStatefulSet(ctx context.Context, instance *appsv2beta1.EMQX) (*appsv1.StatefulSet, error) {
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

	preSts := generateStatefulSet(instance)
	podTemplateSpecHash := computeHash(preSts.Spec.Template.DeepCopy(), instance.Status.CoreNodesStatus.CollisionCount)
	preSts.Name = preSts.Name + "-" + podTemplateSpecHash
	preSts.Labels = appsv2beta1.CloneAndAddLabel(preSts.Labels, appsv2beta1.LabelsPodTemplateHashKey, podTemplateSpecHash)
	preSts.Spec.Selector = appsv2beta1.CloneSelectorAndAddLabel(preSts.Spec.Selector, appsv2beta1.LabelsPodTemplateHashKey, podTemplateSpecHash)
	preSts.Spec.Template.Labels = appsv2beta1.CloneAndAddLabel(preSts.Spec.Template.Labels, appsv2beta1.LabelsPodTemplateHashKey, podTemplateSpecHash)
	preSts.Spec.Template.Spec.Containers[0].Ports = appsv2beta1.MergeContainerPorts(
		preSts.Spec.Template.Spec.Containers[0].Ports,
		[]corev1.ContainerPort{
			containerPort,
		},
	)
	preSts.Spec.Template.Spec.Containers[0].Env = append([]corev1.EnvVar{
		{Name: "EMQX_DASHBOARD__LISTENERS__HTTP__BIND", Value: strconv.Itoa(int(containerPort.ContainerPort))},
	}, preSts.Spec.Template.Spec.Containers[0].Env...)

	updateSts, _, _ := getStateFulSetList(ctx, a.Client, instance)
	if updateSts == nil {
		return preSts, nil
	}

	patchResult, err := a.Patcher.Calculate(
		updateSts.DeepCopy(),
		preSts.DeepCopy(),
		justCheckPodTemplate(),
	)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to calculate patch")
	}
	if patchResult.IsEmpty() {
		preSts.ObjectMeta = updateSts.DeepCopy().ObjectMeta
		preSts.Spec.Template.ObjectMeta = updateSts.DeepCopy().Spec.Template.ObjectMeta
		preSts.Spec.Selector = updateSts.DeepCopy().Spec.Selector
		return preSts, nil
	}

	logger := log.FromContext(ctx)
	logger.Info("got different pod template for EMQX core nodes, will create new statefulSet", "statefulSet", klog.KObj(preSts), "patch", string(patchResult.Patch))
	return preSts, nil
}

func generateStatefulSet(instance *appsv2beta1.EMQX) *appsv1.StatefulSet {
	labels := appsv2beta1.CloneAndMergeMap(
		appsv2beta1.DefaultCoreLabels(instance),
		instance.Spec.CoreTemplate.Labels,
	)

	sts := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.Namespace,
			Name:        instance.CoreNamespacedName().Name,
			Annotations: instance.Spec.CoreTemplate.DeepCopy().Annotations,
			Labels:      labels,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: instance.HeadlessServiceNamespacedName().Name,
			Replicas:    instance.Spec.CoreTemplate.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: instance.Spec.CoreTemplate.DeepCopy().Annotations,
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
					SecurityContext:    instance.Spec.CoreTemplate.Spec.PodSecurityContext,
					Affinity:           instance.Spec.CoreTemplate.Spec.Affinity,
					Tolerations: append(
						instance.Spec.CoreTemplate.Spec.Tolerations,
						// TODO: just for compatible with old version, will remove in future
						instance.Spec.CoreTemplate.Spec.ToleRations...,
					),
					NodeName:       instance.Spec.CoreTemplate.Spec.NodeName,
					NodeSelector:   instance.Spec.CoreTemplate.Spec.NodeSelector,
					InitContainers: instance.Spec.CoreTemplate.Spec.InitContainers,
					Containers: append([]corev1.Container{
						{
							Name:            appsv2beta1.DefaultContainerName,
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
									Value: `"/opt/emqx/data/bootstrap_api_key"`,
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
									Name:      instance.CoreNamespacedName().Name + "-log",
									MountPath: "/opt/emqx/log",
								},
								{
									Name:      instance.CoreNamespacedName().Name + "-data",
									MountPath: "/opt/emqx/data",
								},
							}, instance.Spec.CoreTemplate.Spec.ExtraVolumeMounts...),
						},
					}, instance.Spec.CoreTemplate.Spec.ExtraContainers...),
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
							Name: instance.CoreNamespacedName().Name + "-log",
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
					Labels:    labels,
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
