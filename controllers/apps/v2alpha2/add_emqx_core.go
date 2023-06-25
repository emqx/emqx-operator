package v2alpha2

import (
	"context"
	"reflect"

	emperror "emperror.dev/errors"
	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	"github.com/emqx/emqx-operator/internal/handler"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addCore struct {
	*EMQXReconciler
}

func (a *addCore) reconcile(ctx context.Context, instance *appsv2alpha2.EMQX, _ innerReq.RequesterInterface) subResult {
	sts, collisionCount := a.getNewStatefulSet(ctx, instance)
	if collisionCount != instance.Status.CoreNodesStatus.CollisionCount {
		instance.Status.CoreNodesStatus.CollisionCount = collisionCount
		_ = a.Client.Status().Update(ctx, instance)
	}

	if err := a.CreateOrUpdateList(instance, a.Scheme, []client.Object{sts}); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to create or update statefulSet")}
	}

	if err := a.syncStatefulSet(ctx, instance); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to sync replicaSet")}
	}

	return subResult{}
}

func (a *addCore) getNewStatefulSet(ctx context.Context, instance *appsv2alpha2.EMQX) (*appsv1.StatefulSet, *int32) {
	preSts := generateStatefulSet(instance)

	list := &appsv1.StatefulSetList{}
	_ = a.Client.List(ctx, list,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(instance.Spec.CoreTemplate.Labels),
	)

	for _, sts := range list.Items {
		patchResult, _ := a.Patcher.Calculate(
			sts.DeepCopy(),
			preSts.DeepCopy(),
			justCheckPodTemplateSpec(),
		)
		if patchResult.IsEmpty() {
			preSts.ObjectMeta = *sts.ObjectMeta.DeepCopy()
			preSts.Spec.Template.ObjectMeta = *sts.Spec.Template.ObjectMeta.DeepCopy()
			preSts.Spec.Selector = sts.Spec.Selector.DeepCopy()
			return preSts, instance.Status.CoreNodesStatus.CollisionCount
		}
	}

	var collisionCount *int32
	var staName, podTemplateSpecHash string
	podTemplate := preSts.Spec.Template.DeepCopy()

	collisionCount = instance.Status.CoreNodesStatus.CollisionCount
	if collisionCount == nil {
		collisionCount = new(int32)
	}

	// Do-while loop
	for {
		podTemplateSpecHash = computeHash(podTemplate.DeepCopy(), collisionCount)

		staName = instance.Spec.CoreTemplate.Name + "-" + podTemplateSpecHash
		err := a.Client.Get(context.TODO(), types.NamespacedName{
			Namespace: instance.Namespace,
			Name:      staName,
		}, &appsv1.StatefulSet{})
		if k8sErrors.IsNotFound(err) {
			break
		}
		*collisionCount++
	}

	preSts.Name = staName
	preSts.Labels = appsv2alpha2.CloneAndAddLabel(preSts.Labels, appsv1.DefaultDeploymentUniqueLabelKey, podTemplateSpecHash)
	preSts.Spec.Template.Labels = appsv2alpha2.CloneAndAddLabel(preSts.Spec.Template.Labels, appsv1.DefaultDeploymentUniqueLabelKey, podTemplateSpecHash)
	preSts.Spec.Selector = appsv2alpha2.CloneSelectorAndAddLabel(preSts.Spec.Selector, appsv1.DefaultDeploymentUniqueLabelKey, podTemplateSpecHash)
	return preSts, collisionCount
}

func (a *addCore) syncStatefulSet(ctx context.Context, instance *appsv2alpha2.EMQX) error {
	stsList := getStateFulSetList(ctx, a.Client,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(instance.Spec.CoreTemplate.Labels),
	)
	if len(stsList) <= 1 {
		return nil
	}

	old := stsList[0].DeepCopy()
	eList := getEventList(ctx, a.Clientset, old)

	if canBeScaledDown(instance, appsv2alpha2.CodeNodesReady, eList) {
		old.Spec.Replicas = pointer.Int32Ptr(old.Status.Replicas - 1)
		if err := a.Client.Update(ctx, old); err != nil {
			return emperror.Wrap(err, "failed to scale down old replicaSet")
		}
		return nil
	}
	return nil
}

func generateStatefulSet(instance *appsv2alpha2.EMQX) *appsv1.StatefulSet {
	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      instance.Spec.CoreTemplate.Labels,
			Annotations: instance.Spec.CoreTemplate.Annotations,
		},
		Spec: corev1.PodSpec{
			ImagePullSecrets: instance.Spec.ImagePullSecrets,
			SecurityContext:  instance.Spec.CoreTemplate.Spec.PodSecurityContext,
			Affinity:         instance.Spec.CoreTemplate.Spec.Affinity,
			Tolerations:      instance.Spec.CoreTemplate.Spec.ToleRations,
			NodeName:         instance.Spec.CoreTemplate.Spec.NodeName,
			NodeSelector:     instance.Spec.CoreTemplate.Spec.NodeSelector,
			InitContainers:   instance.Spec.CoreTemplate.Spec.InitContainers,
			Containers: append([]corev1.Container{
				{
					Name:            EMQXContainerName,
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
							Name:  "EMQX_HOST",
							Value: "$(POD_NAME).$(STS_HEADLESS_SERVICE_NAME).$(POD_NAMESPACE).svc.cluster.local",
						},
						{
							Name:  "EMQX_NODE__DB_ROLE",
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
							Name:      instance.CoreNodeNamespacedName().Name + "-log",
							MountPath: "/opt/emqx/log",
						},
						{
							Name:      instance.CoreNodeNamespacedName().Name + "-data",
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
					Name: instance.CoreNodeNamespacedName().Name + "-log",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			}, instance.Spec.CoreTemplate.Spec.ExtraVolumes...),
		},
	}

	podTemplate.Annotations = handler.SetManagerContainerAnnotation(podTemplate.Annotations, podTemplate.Spec.Containers)
	podTemplate.Annotations["apps.emqx.io/headless-service-name"] = instance.HeadlessServiceNamespacedName().Name

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
			Template:            podTemplate,
		},
	}
	if !reflect.ValueOf(instance.Spec.CoreTemplate.Spec.VolumeClaimTemplates).IsZero() {
		sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instance.CoreNodeNamespacedName().Name + "-data",
					Namespace: instance.Namespace,
					Labels:    instance.Spec.CoreTemplate.Labels,
				},
				Spec: instance.Spec.CoreTemplate.Spec.VolumeClaimTemplates,
			},
		}
	} else {
		sts.Spec.Template.Spec.Volumes = append([]corev1.Volume{
			{
				Name: instance.CoreNodeNamespacedName().Name + "-data",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		}, sts.Spec.Template.Spec.Volumes...)
	}

	return sts
}
