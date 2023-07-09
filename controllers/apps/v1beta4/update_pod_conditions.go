package v1beta4

import (
	"context"
	"encoding/json"
	"fmt"

	emperror "emperror.dev/errors"
	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type updatePodConditions struct {
	*EmqxReconciler
	Requester innerReq.RequesterInterface
}

func (u updatePodConditions) reconcile(ctx context.Context, instance appsv1beta4.Emqx, _ ...any) subResult {
	pods := &corev1.PodList{}
	_ = u.Client.List(ctx, pods,
		client.InNamespace(instance.GetNamespace()),
		client.MatchingLabels(instance.GetSpec().GetTemplate().Labels),
	)
	clusterPods := make(map[string]struct{})
	for _, emqx := range instance.GetStatus().GetEmqxNodes() {
		podName := extractPodName(emqx.Node)
		clusterPods[podName] = struct{}{}
	}

	for _, pod := range pods.Items {
		if _, ok := clusterPods[pod.Name]; !ok {
			continue
		}

		hash := make(map[corev1.PodConditionType]corev1.PodCondition)

		for _, condition := range pod.Status.Conditions {
			hash[condition.Type] = condition
		}

		if c, ok := hash[corev1.ContainersReady]; !ok || c.Status != corev1.ConditionTrue {
			continue
		}

		onServerCondition := corev1.PodCondition{
			Type:               appsv1beta4.PodOnServing,
			Status:             corev1.ConditionFalse,
			LastProbeTime:      metav1.Now(),
			LastTransitionTime: metav1.Now(),
		}

		if c, ok := hash[appsv1beta4.PodOnServing]; ok {
			onServerCondition.LastTransitionTime = c.LastTransitionTime
		}

		onServerCondition.Status = corev1.ConditionTrue
		if enterprise, ok := instance.(*appsv1beta4.EmqxEnterprise); ok {
			s, err := u.checkRebalanceStatus(enterprise, pod.DeepCopy())
			if err != nil {
				return subResult{err: err}
			}
			onServerCondition.Status = s
		}

		patchBytes, _ := json.Marshal(corev1.Pod{
			Status: corev1.PodStatus{
				Conditions: []corev1.PodCondition{onServerCondition},
			},
		})
		err := u.Client.Status().Patch(ctx, &pod, client.RawPatch(types.StrategicMergePatchType, patchBytes))
		if err != nil {
			return subResult{err: emperror.Wrap(err, "failed to patch pod conditions")}
		}
	}
	return subResult{}
}

func (u updatePodConditions) checkRebalanceStatus(instance *appsv1beta4.EmqxEnterprise, pod *corev1.Pod) (corev1.ConditionStatus, error) {
	requester := &innerReq.Requester{
		Username: u.Requester.GetUsername(),
		Password: u.Requester.GetPassword(),
		Host:     fmt.Sprintf("%s:8081", pod.Status.PodIP),
	}
	resp, _, err := requester.Request("GET", requester.GetURL("api/v4/load_rebalance/availability_check"), nil, nil)
	if err != nil {
		return corev1.ConditionUnknown, emperror.Wrapf(err, "failed to check availability for pod/%s", pod.Name)
	}
	if resp.StatusCode != 200 {
		return corev1.ConditionFalse, nil
	}
	return corev1.ConditionTrue, nil
}
