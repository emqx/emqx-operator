package v2alpha2

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	emperror "emperror.dev/errors"
	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type addCore struct {
	*EMQXReconciler
}

func (a *addCore) reconcile(ctx context.Context, instance *appsv2alpha2.EMQX, _ innerReq.RequesterInterface) subResult {
	sts := a.getNewStatefulSet(ctx, instance)
	if sts.UID == "" {
		_ = ctrl.SetControllerReference(instance, sts, a.Scheme)
		if err := a.Handler.Create(sts); err != nil {
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
	} else {
		if err := a.Handler.CreateOrUpdate(sts); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to update statefulSet")}
		}
	}

	if instance.Status.CoreNodesStatus.CurrentRevision != sts.Labels[appsv2alpha2.PodTemplateHashLabelKey] {
		instance.Status.CoreNodesStatus.CurrentRevision = sts.Labels[appsv2alpha2.PodTemplateHashLabelKey]
		_ = a.Client.Status().Update(ctx, instance)
	}

	if err := a.sync(ctx, instance); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to sync replicaSet")}
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

func (a *addCore) sync(ctx context.Context, instance *appsv2alpha2.EMQX) error {
	if isExistReplicant(instance) {
		_, oldRsList := getReplicaSetList(ctx, a.Client, instance)
		if len(oldRsList) != 0 {
			// wait for replicaSet finished the scale down
			return nil
		}
	}

	_, oldStsList := getStateFulSetList(ctx, a.Client, instance)
	if len(oldStsList) == 0 {
		return nil
	}

	oldest := oldStsList[0].DeepCopy()

	if a.findCanBeDeletePod(ctx, instance, oldest) != nil {
		oldest.Spec.Replicas = pointer.Int32Ptr(oldest.Status.Replicas - 1)
		if err := a.Client.Update(ctx, oldest); err != nil {
			return emperror.Wrap(err, "failed to scale down old replicaSet")
		}
		return nil
	}

	return nil
}

func (a *addCore) findCanBeDeletePod(ctx context.Context, instance *appsv2alpha2.EMQX, old *appsv1.StatefulSet) *corev1.Pod {
	if !canBeScaledDown(instance, appsv2alpha2.Ready, getEventList(ctx, a.Clientset, old)) {
		return nil
	}
	pod := &corev1.Pod{}
	_ = a.Client.Get(ctx, types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      fmt.Sprintf("%s-%d", old.Name, old.Status.Replicas-1),
	}, pod)

	for _, node := range instance.Status.CoreNodesStatus.Nodes {
		host := strings.Split(node.Node[strings.Index(node.Node, "@")+1:], ":")[0]
		if strings.HasPrefix(host, pod.Name) {
			if node.Edition == "Enterprise" && node.Session != 0 {
				return nil
			}
			return pod
		}
	}
	return nil
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
					ImagePullSecrets: instance.Spec.ImagePullSecrets,
					SecurityContext:  instance.Spec.CoreTemplate.Spec.PodSecurityContext,
					Affinity:         instance.Spec.CoreTemplate.Spec.Affinity,
					Tolerations:      instance.Spec.CoreTemplate.Spec.ToleRations,
					NodeName:         instance.Spec.CoreTemplate.Spec.NodeName,
					NodeSelector:     instance.Spec.CoreTemplate.Spec.NodeSelector,
					InitContainers:   instance.Spec.CoreTemplate.Spec.InitContainers,
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
									Name: "POD_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									Name: "STS_HEADLESS_SERVICE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.annotations['apps.emqx.io/headless-service-name']",
										},
									},
								},
								{
									Name:  "EMQX_NODE__DB_ROLE",
									Value: "core",
								},
								{
									Name:  "EMQX_HOST",
									Value: "$(POD_NAME).$(STS_HEADLESS_SERVICE_NAME).$(POD_NAMESPACE).svc.cluster.local",
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

	sts.Spec.Template.Annotations = appsv2alpha2.CloneAndAddLabel(
		sts.Spec.Template.Annotations,
		"apps.emqx.io/headless-service-name",
		sts.Spec.ServiceName,
	)

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
