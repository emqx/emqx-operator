package v2alpha1

import (
	"context"
	"encoding/json"

	semver "github.com/Masterminds/semver/v3"
	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	innerPortFW "github.com/emqx/emqx-operator/internal/portforward"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type updatePodConditions struct {
	*EMQXReconciler
}

func (u *updatePodConditions) reconcile(ctx context.Context, instance *appsv2alpha1.EMQX, p *portForwardAPI) subResult {
	pods := &corev1.PodList{}
	_ = u.Client.List(ctx, pods,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
	)

	for _, pod := range pods.Items {
		hash := make(map[corev1.PodConditionType]int)

		for i, condition := range pod.Status.Conditions {
			hash[condition.Type] = i
		}

		if index, ok := hash[corev1.ContainersReady]; !ok || pod.Status.Conditions[index].Status != corev1.ConditionTrue {
			continue
		}

		onServingCondition := corev1.PodCondition{
			Type:               appsv2alpha1.PodOnServing,
			Status:             u.checkInCluster(instance, p, pod.DeepCopy()),
			LastProbeTime:      metav1.Now(),
			LastTransitionTime: metav1.Now(),
		}
		if index, ok := hash[appsv2alpha1.PodOnServing]; ok {
			onServingCondition.LastTransitionTime = pod.Status.Conditions[index].LastTransitionTime
		}

		patchBytes, _ := json.Marshal(corev1.Pod{
			Status: corev1.PodStatus{
				Conditions: []corev1.PodCondition{onServingCondition},
			},
		})
		_ = u.Client.Status().Patch(ctx, &pod, client.RawPatch(types.StrategicMergePatchType, patchBytes))
	}
	return subResult{}
}

func (u *updatePodConditions) checkInCluster(instance *appsv2alpha1.EMQX, p *portForwardAPI, pod *corev1.Pod) corev1.ConditionStatus {
	for _, node := range instance.Status.EMQXNodes {
		if node.Node == "emqx@"+pod.Status.PodIP {
			if node.Edition == "enterprise" {
				v, _ := semver.NewVersion(node.Version)
				if v.Compare(semver.MustParse("5.0.3")) >= 0 {
					return u.checkRebalanceStatus(instance, p, pod)
				}
			}
			return corev1.ConditionTrue
		}
	}
	return corev1.ConditionFalse
}

func (u *updatePodConditions) checkRebalanceStatus(instance *appsv2alpha1.EMQX, p *portForwardAPI, pod *corev1.Pod) corev1.ConditionStatus {
	// Need check every pods, so must create new port forward options
	o, _ := innerPortFW.NewPortForwardOptions(u.Clientset, u.Config, pod, "8081")
	if o == nil {
		return corev1.ConditionUnknown
	}
	defer close(o.StopChannel)
	if err := o.ForwardPorts(); err != nil {
		return corev1.ConditionUnknown
	}
	resp, _, err := (&portForwardAPI{
		// Doesn't need get username and password from secret
		// because they are same as the emqx cluster
		Username: p.Username,
		Password: p.Password,
		Options:  o,
	}).requestAPI("GET", "api/v5/load_rebalance/availability_check", nil)
	if err != nil {
		return corev1.ConditionUnknown
	}
	if resp.StatusCode != 200 {
		return corev1.ConditionFalse
	}
	return corev1.ConditionTrue
}
