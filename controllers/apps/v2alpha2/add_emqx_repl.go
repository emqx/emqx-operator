package v2alpha2

import (
	"context"
	"encoding/json"
	"time"

	emperror "emperror.dev/errors"
	"github.com/banzaicloud/k8s-objectmatcher/patch"
	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	"github.com/tidwall/gjson"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addRepl struct {
	*EMQXReconciler
}

func (a *addRepl) reconcile(ctx context.Context, instance *appsv2alpha2.EMQX, _ innerReq.RequesterInterface) subResult {
	if !isExistReplicant(instance) {
		return subResult{}
	}
	if !instance.Status.IsConditionTrue(appsv2alpha2.CodeNodesReady) {
		return subResult{}
	}

	rs, collisionCount := a.getNewReplicaSet(ctx, instance)
	if collisionCount != instance.Status.ReplicantNodesStatus.CollisionCount {
		instance.Status.ReplicantNodesStatus.CollisionCount = collisionCount
		_ = a.Client.Status().Update(ctx, instance)
	}

	if rs.UID == "" {
		if err := ctrl.SetControllerReference(instance, rs, a.Scheme); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to set controller reference")}
		}
		if err := a.Handler.Create(rs); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to create replicaSet")}
		}
		return subResult{result: ctrl.Result{}}
	}

	if err := a.CreateOrUpdateList(instance, a.Scheme, []client.Object{rs}); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to create or update replicaSet")}
	}

	if err := a.syncReplicaSet(ctx, instance); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to sync replicaSet")}
	}

	return subResult{}
}

func (a *addRepl) getNewReplicaSet(ctx context.Context, instance *appsv2alpha2.EMQX) (*appsv1.ReplicaSet, *int32) {
	list := &appsv1.ReplicaSetList{}
	_ = a.Client.List(ctx, list,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
	)

	rs, collisionCount := a.generateReplicaSet(instance)

	for _, r := range list.Items {
		patchResult, _ := a.Patcher.Calculate(
			r.DeepCopy(),
			rs.DeepCopy(),
			justCheckPodTemplateSpec(),
		)
		if patchResult.IsEmpty() {
			rs.ObjectMeta = *r.ObjectMeta.DeepCopy()
			rs.Spec.Template.ObjectMeta = *r.Spec.Template.ObjectMeta.DeepCopy()
			rs.Spec.Selector = r.Spec.Selector.DeepCopy()
			return rs, instance.Status.ReplicantNodesStatus.CollisionCount
		}
	}

	return rs, collisionCount
}

func (a *addRepl) generateReplicaSet(instance *appsv2alpha2.EMQX) (*appsv1.ReplicaSet, *int32) {
	var collisionCount *int32
	var rsName string
	var podTemplateSpecHash string

	collisionCount = instance.Status.ReplicantNodesStatus.CollisionCount
	if collisionCount == nil {
		collisionCount = new(int32)
	}

	podTemplate := generatePodTemplateSpec(instance)

	// Do-while loop
	for {
		podTemplateSpecHash = computeHash(podTemplate.DeepCopy(), collisionCount)
		rsName = instance.Spec.ReplicantTemplate.Name + "-" + podTemplateSpecHash
		err := a.Client.Get(context.TODO(), types.NamespacedName{
			Namespace: instance.Namespace,
			Name:      rsName,
		}, &appsv1.ReplicaSet{})
		if k8sErrors.IsNotFound(err) {
			break
		}
		*collisionCount++
	}

	podTemplate.Labels = appsv2alpha2.CloneAndAddLabel(podTemplate.Labels, appsv1.DefaultDeploymentUniqueLabelKey, podTemplateSpecHash)
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        rsName,
			Namespace:   instance.Namespace,
			Annotations: instance.Spec.ReplicantTemplate.Annotations,
			Labels:      podTemplate.Labels,
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: instance.Spec.ReplicantTemplate.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: podTemplate.Labels,
			},
			Template: podTemplate,
		},
	}
	rs.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("ReplicaSet"))
	return rs, collisionCount
}

func (a *addRepl) syncReplicaSet(ctx context.Context, instance *appsv2alpha2.EMQX) error {
	rsList := getReplicaSetList(ctx, a.Client,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
	)
	if len(rsList) <= 1 {
		return nil
	}

	old := rsList[0].DeepCopy()
	eList := getEventList(ctx, a.Clientset, old)

	if canBeScaledDownRs(instance, rsList[len(rsList)-1].DeepCopy(), eList) {
		old.Spec.Replicas = pointer.Int32Ptr(old.Status.Replicas - 1)
		if err := a.Client.Update(ctx, old); err != nil {
			return emperror.Wrap(err, "failed to scale down old replicaSet")
		}
		return nil
	}
	return nil
}

func canBeScaledDownRs(instance *appsv2alpha2.EMQX, current *appsv1.ReplicaSet, eList []*corev1.Event) bool {
	var initialDelaySecondsReady bool
	var waitTakeover bool

	_, condition := instance.Status.GetCondition(appsv2alpha2.CodeNodesReady)
	if condition != nil && condition.Status == metav1.ConditionTrue {
		delay := time.Since(condition.LastTransitionTime.Time).Seconds()
		if int32(delay) > instance.Spec.BlueGreenUpdate.InitialDelaySeconds {
			initialDelaySecondsReady = true
		}
	}

	if len(eList) == 0 {
		waitTakeover = true
		return initialDelaySecondsReady && waitTakeover
	}

	lastEvent := eList[len(eList)-1]
	delay := time.Since(lastEvent.LastTimestamp.Time).Seconds()
	if int32(delay) > instance.Spec.BlueGreenUpdate.EvacuationStrategy.WaitTakeover {
		waitTakeover = true
	}

	return initialDelaySecondsReady && waitTakeover
}

func generatePodTemplateSpec(instance *appsv2alpha2.EMQX) corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      instance.Spec.ReplicantTemplate.Labels,
			Annotations: instance.Spec.ReplicantTemplate.Annotations,
		},
		Spec: corev1.PodSpec{
			ReadinessGates: []corev1.PodReadinessGate{
				{
					ConditionType: appsv2alpha2.PodOnServing,
				},
			},
			ImagePullSecrets: instance.Spec.ImagePullSecrets,
			SecurityContext:  instance.Spec.ReplicantTemplate.Spec.PodSecurityContext,
			Affinity:         instance.Spec.ReplicantTemplate.Spec.Affinity,
			Tolerations:      instance.Spec.ReplicantTemplate.Spec.ToleRations,
			NodeName:         instance.Spec.ReplicantTemplate.Spec.NodeName,
			NodeSelector:     instance.Spec.ReplicantTemplate.Spec.NodeSelector,
			InitContainers:   instance.Spec.ReplicantTemplate.Spec.InitContainers,
			Containers: append([]corev1.Container{
				{
					Name:            EMQXContainerName,
					Image:           instance.Spec.Image,
					ImagePullPolicy: instance.Spec.ImagePullPolicy,
					Command:         instance.Spec.ReplicantTemplate.Spec.Command,
					Args:            instance.Spec.ReplicantTemplate.Spec.Args,
					Ports:           instance.Spec.ReplicantTemplate.Spec.Ports,
					Env: append([]corev1.EnvVar{
						{
							Name:  "EMQX_NODE__DB_ROLE",
							Value: "replicant",
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
					}, instance.Spec.ReplicantTemplate.Spec.Env...),
					EnvFrom:         instance.Spec.ReplicantTemplate.Spec.EnvFrom,
					Resources:       instance.Spec.ReplicantTemplate.Spec.Resources,
					SecurityContext: instance.Spec.ReplicantTemplate.Spec.ContainerSecurityContext,
					LivenessProbe:   instance.Spec.ReplicantTemplate.Spec.LivenessProbe,
					ReadinessProbe:  instance.Spec.ReplicantTemplate.Spec.ReadinessProbe,
					StartupProbe:    instance.Spec.ReplicantTemplate.Spec.StartupProbe,
					Lifecycle:       instance.Spec.ReplicantTemplate.Spec.Lifecycle,
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
							Name:      instance.ReplicantNodeNamespacedName().Name + "-log",
							MountPath: "/opt/emqx/log",
						},
						{
							Name:      instance.ReplicantNodeNamespacedName().Name + "-data",
							MountPath: "/opt/emqx/data",
						},
					}, instance.Spec.ReplicantTemplate.Spec.ExtraVolumeMounts...),
				},
			}, instance.Spec.ReplicantTemplate.Spec.ExtraContainers...),
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
					Name: instance.ReplicantNodeNamespacedName().Name + "-log",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: instance.ReplicantNodeNamespacedName().Name + "-data",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			}, instance.Spec.ReplicantTemplate.Spec.ExtraVolumes...),
		},
	}
}

// JustCheckPodTemplate will check only the differences between the podTemplate of the two statefulSets
func justCheckPodTemplateSpec() patch.CalculateOption {
	getPodTemplate := func(obj []byte) ([]byte, error) {
		podTemplateSpecJson := gjson.GetBytes(obj, "spec.template.spec")
		podTemplateSpec := &corev1.PodSpec{}
		_ = json.Unmarshal([]byte(podTemplateSpecJson.String()), podTemplateSpec)

		emptyRs := &appsv1.ReplicaSet{}
		emptyRs.Spec.Template.Spec = *podTemplateSpec
		return json.Marshal(emptyRs)
	}

	return func(current, modified []byte) ([]byte, []byte, error) {
		current, err := getPodTemplate(current)
		if err != nil {
			return []byte{}, []byte{}, emperror.Wrap(err, "could not get pod template field from current byte sequence")
		}

		modified, err = getPodTemplate(modified)
		if err != nil {
			return []byte{}, []byte{}, emperror.Wrap(err, "could not get pod template field from modified byte sequence")
		}

		return current, modified, nil
	}
}
