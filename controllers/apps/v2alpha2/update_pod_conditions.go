package v2alpha2

import (
	"context"
	"encoding/json"
	"fmt"

	semver "github.com/Masterminds/semver/v3"
	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type updatePodConditions struct {
	*EMQXReconciler
}

func (u *updatePodConditions) reconcile(ctx context.Context, instance *appsv2alpha2.EMQX, r innerReq.RequesterInterface) subResult {
	pods := &corev1.PodList{}
	labels := instance.Spec.CoreTemplate.Labels
	if isExistReplicant(instance) {
		labels = instance.Spec.ReplicantTemplate.Labels
	}
	_ = u.Client.List(ctx, pods,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(labels),
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
			Type:               appsv2alpha2.PodOnServing,
			Status:             u.checkInCluster(instance, r, pod.DeepCopy()),
			LastProbeTime:      metav1.Now(),
			LastTransitionTime: metav1.Now(),
		}
		if index, ok := hash[appsv2alpha2.PodOnServing]; ok {
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

func (u *updatePodConditions) checkInCluster(instance *appsv2alpha2.EMQX, r innerReq.RequesterInterface, pod *corev1.Pod) corev1.ConditionStatus {
	nodes := instance.Status.CoreNodesStatus.Nodes
	if isExistReplicant(instance) {
		nodes = instance.Status.ReplicantNodesStatus.Nodes
	}
	for _, node := range nodes {
		if node.Node == "emqx@"+pod.Status.PodIP {
			if node.Edition == "enterprise" {
				v, _ := semver.NewVersion(node.Version)
				if v.Compare(semver.MustParse("5.0.3")) >= 0 {
					return u.checkRebalanceStatus(instance, r, pod)
				}
			}
			return corev1.ConditionTrue
		}
	}
	return corev1.ConditionFalse
}

func (u *updatePodConditions) checkRebalanceStatus(instance *appsv2alpha2.EMQX, r innerReq.RequesterInterface, pod *corev1.Pod) corev1.ConditionStatus {
	var port string
	dashboardPort, err := appsv2alpha2.GetDashboardServicePort(instance)
	if err != nil || dashboardPort == nil {
		port = "18083"
	}

	if dashboardPort != nil {
		port = dashboardPort.TargetPort.String()
	}

	requester := &innerReq.Requester{
		Username: r.GetUsername(),
		Password: r.GetPassword(),
		Host:     fmt.Sprintf("%s:%s", pod.Status.PodIP, port),
	}

	resp, _, err := requester.Request("GET", "api/v5/load_rebalance/availability_check", nil)
	if err != nil {
		return corev1.ConditionUnknown
	}
	if resp.StatusCode != 200 {
		return corev1.ConditionFalse
	}
	return corev1.ConditionTrue
}
