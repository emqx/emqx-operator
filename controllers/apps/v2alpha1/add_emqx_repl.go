package v2alpha1

import (
	"context"
	"encoding/json"
	"time"

	emperror "emperror.dev/errors"
	"github.com/banzaicloud/k8s-objectmatcher/patch"
	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	"github.com/tidwall/gjson"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addRepl struct {
	*EMQXReconciler
}

func (a *addRepl) reconcile(ctx context.Context, instance *appsv2alpha1.EMQX, _ *portForwardAPI) subResult {
	if !instance.Status.IsRunning() && !instance.Status.IsCoreNodesReady() {
		return subResult{}
	}

	deploy := a.getNewDeployment(ctx, instance)
	if deploy.UID == "" {
		if err := ctrl.SetControllerReference(instance, deploy, a.Scheme); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to set controller reference")}
		}
		if err := a.Handler.Create(deploy); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to create deployment")}
		}
		return subResult{result: ctrl.Result{}}
	}

	if err := a.CreateOrUpdateList(instance, a.Scheme, []client.Object{deploy}); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to create or update deployment")}
	}

	if err := a.syncDeployment(ctx, instance); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to sync deployment")}
	}

	return subResult{}
}

func (a *addRepl) getNewDeployment(ctx context.Context, instance *appsv2alpha1.EMQX) *appsv1.Deployment {
	list := &appsv1.DeploymentList{}
	_ = a.Client.List(ctx, list,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
	)

	deploy := generateDeployment(instance)

	patchOpts := []patch.CalculateOption{
		justCheckPodTemplate(),
	}

	for _, d := range list.Items {
		patchResult, _ := a.Patcher.Calculate(
			d.DeepCopy(),
			deploy.DeepCopy(),
			patchOpts...,
		)
		if patchResult.IsEmpty() {
			deploy.ObjectMeta = *d.ObjectMeta.DeepCopy()
			return deploy
		}
	}

	return deploy
}

func (a *addRepl) syncDeployment(ctx context.Context, instance *appsv2alpha1.EMQX) error {
	dList := getDeploymentList(ctx, a.Client,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
	)
	if len(dList) <= 1 {
		return nil
	}

	old := dList[0].DeepCopy()
	eList := getEventList(ctx, a.Clientset, old)

	if canBeScaledDown(instance, dList[len(dList)-1].DeepCopy(), eList) {
		old.Spec.Replicas = pointer.Int32Ptr(old.Status.Replicas - 1)
		if err := a.Client.Update(ctx, old); err != nil {
			return emperror.Wrap(err, "failed to scale down old deployment")
		}
		return nil
	}
	return nil
}

func canBeScaledDown(instance *appsv2alpha1.EMQX, currentDeployment *appsv1.Deployment, eList []*corev1.Event) bool {
	var initialDelaySecondsReady bool
	var waitTakeover bool
	for _, c := range currentDeployment.Status.Conditions {
		if c.Type == appsv1.DeploymentAvailable && c.Status == corev1.ConditionTrue {
			delay := time.Since(c.LastTransitionTime.Time).Seconds()
			if int32(delay) > instance.Spec.BlueGreenUpdate.InitialDelaySeconds {
				initialDelaySecondsReady = true
			}
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

func generateDeployment(instance *appsv2alpha1.EMQX) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    instance.Namespace,
			GenerateName: instance.Spec.ReplicantTemplate.Name + "-",
			Labels:       instance.Spec.ReplicantTemplate.Labels,
			Annotations:  instance.Spec.ReplicantTemplate.Annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: instance.Spec.ReplicantTemplate.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: instance.Spec.ReplicantTemplate.Labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      instance.Spec.ReplicantTemplate.Labels,
					Annotations: instance.Spec.ReplicantTemplate.Annotations,
				},
				Spec: corev1.PodSpec{
					ReadinessGates: []corev1.PodReadinessGate{
						{
							ConditionType: appsv2alpha1.PodInCluster,
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
									Name:  "EMQX_DASHBOARD__BOOTSTRAP_USERS_FILE",
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
			},
		},
	}
}

// JustCheckPodTemplate will check only the differences between the podTemplate of the two statefulSets
func justCheckPodTemplate() patch.CalculateOption {
	getPodTemplate := func(obj []byte) ([]byte, error) {
		podTemplateJson := gjson.GetBytes(obj, "spec.template")
		podTemplate := &corev1.PodTemplateSpec{}
		_ = json.Unmarshal([]byte(podTemplateJson.String()), podTemplate)

		emptySts := &appsv1.StatefulSet{}
		emptySts.Spec.Template = *podTemplate
		return json.Marshal(emptySts)
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
