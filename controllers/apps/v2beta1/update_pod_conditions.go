package v2beta1

import (
	"context"
	"encoding/json"
	"net"
	"strconv"

	semver "github.com/Masterminds/semver/v3"
	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type updatePodConditions struct {
	*EMQXReconciler
}

func (u *updatePodConditions) reconcile(ctx context.Context, logger logr.Logger, instance *appsv2beta1.EMQX, r innerReq.RequesterInterface) subResult {
	updateRs, currentRs, _ := getReplicaSetList(ctx, u.Client, instance)
	updateSts, currentSts, _ := getStateFulSetList(ctx, u.Client, instance)

	pods := &corev1.PodList{}
	_ = u.Client.List(ctx, pods,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(appsv2beta1.DefaultLabels(instance)),
	)

	for _, p := range pods.Items {
		pod := p.DeepCopy()
		controllerRef := metav1.GetControllerOf(pod)
		if controllerRef == nil {
			continue
		}

		onServingCondition := corev1.PodCondition{
			Type:               appsv2beta1.PodOnServing,
			Status:             corev1.ConditionFalse,
			LastProbeTime:      metav1.Now(),
			LastTransitionTime: metav1.Now(),
		}

		if instance.Status.IsConditionTrue(appsv2beta1.Available) {
			if (updateSts != nil && controllerRef.UID == updateSts.UID) ||
				(updateRs != nil && controllerRef.UID == updateRs.UID) {
				for _, condition := range pod.Status.Conditions {
					if condition.Type == corev1.ContainersReady && condition.Status == corev1.ConditionTrue {
						onServingCondition.Status = u.checkInCluster(instance, r, pod)
						break
					}
				}
			}
		} else {
			if (currentSts != nil && controllerRef.UID == currentSts.UID) ||
				(currentRs != nil && controllerRef.UID == currentRs.UID) ||
				(updateSts != nil && controllerRef.UID == updateSts.UID) ||
				(updateRs != nil && controllerRef.UID == updateRs.UID) {
				for _, condition := range pod.Status.Conditions {
					if condition.Type == corev1.ContainersReady && condition.Status == corev1.ConditionTrue {
						onServingCondition.Status = u.checkInCluster(instance, r, pod)
						break
					}
				}
			}
		}

		patchBytes, _ := json.Marshal(corev1.Pod{
			Status: corev1.PodStatus{
				Conditions: []corev1.PodCondition{onServingCondition},
			},
		})
		_ = u.Client.Status().Patch(ctx, pod.DeepCopy(), client.RawPatch(types.StrategicMergePatchType, patchBytes))
	}
	return subResult{}
}

func (u *updatePodConditions) checkInCluster(instance *appsv2beta1.EMQX, r innerReq.RequesterInterface, pod *corev1.Pod) corev1.ConditionStatus {
	nodes := instance.Status.CoreNodes
	if appsv2beta1.IsExistReplicant(instance) {
		nodes = append(nodes, instance.Status.ReplicantNodes...)
	}
	for _, node := range nodes {
		if pod.UID == node.PodUID {
			if node.Edition == "Enterprise" {
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

func (u *updatePodConditions) checkRebalanceStatus(instance *appsv2beta1.EMQX, r innerReq.RequesterInterface, pod *corev1.Pod) corev1.ConditionStatus {
	if r == nil {
		return corev1.ConditionFalse
	}

	portMap, _ := appsv2beta1.GetDashboardPortMap(instance.Spec.Config.Data)

	var schema, port string
	if dashboardHttps, ok := portMap["dashboard-https"]; ok {
		schema = "https"
		port = strconv.FormatInt(int64(dashboardHttps), 10)
	}
	if dashboard, ok := portMap["dashboard"]; ok {
		schema = "http"
		port = strconv.FormatInt(int64(dashboard), 10)
	}

	requester := &innerReq.Requester{
		Schema:   schema,
		Host:     net.JoinHostPort(pod.Status.PodIP, port),
		Username: r.GetUsername(),
		Password: r.GetPassword(),
	}

	url := requester.GetURL("api/v5/load_rebalance/availability_check")
	resp, _, err := requester.Request("GET", url, nil, nil)
	if err != nil {
		return corev1.ConditionUnknown
	}
	if resp.StatusCode != 200 {
		return corev1.ConditionFalse
	}
	return corev1.ConditionTrue
}
