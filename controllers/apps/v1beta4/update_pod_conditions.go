package v1beta4

import (
	"context"
	"encoding/json"
	"fmt"

	emperror "emperror.dev/errors"
	"github.com/emqx/emqx-operator/apis/apps/v1beta4"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type updatePodConditions struct {
	*EmqxReconciler
	*portForwardAPI
}

func (u updatePodConditions) reconcile(ctx context.Context, instance v1beta4.Emqx, _ ...any) subResult {
	pods := &corev1.PodList{}
	_ = u.Client.List(ctx, pods,
		client.InNamespace(instance.GetNamespace()),
		client.MatchingLabels(instance.GetLabels()),
	)
	clusterNodes := make(map[string]struct{})
	for _, emqx := range instance.GetStatus().GetEmqxNodes() {
		clusterNodes[emqx.Node] = struct{}{}
	}

	for _, pod := range pods.Items {
		hash := make(map[corev1.PodConditionType]struct{})

		for _, condition := range pod.Status.Conditions {
			if condition.Status == corev1.ConditionTrue {
				hash[condition.Type] = struct{}{}
			}
		}

		if !isPodReady(hash) {
			patch := []map[string]interface{}{}
			emqxNodeName := fmt.Sprintf("%s@%s", "emqx", pod.Status.PodIP)
			if _, ok := clusterNodes[emqxNodeName]; ok {
				patch = append(patch,
					map[string]interface{}{
						"op":   "add",
						"path": "/status/conditions/-",
						"value": corev1.PodCondition{
							Type:               v1beta4.PodInCluster,
							Status:             corev1.ConditionTrue,
							LastProbeTime:      metav1.Now(),
							LastTransitionTime: metav1.Now(),
						},
					},
				)
			}

			resp, _, err := u.portForwardAPI.requestAPI("Get", "api/v4/load_rebalance/availability_check", nil)
			if err != nil {
				return subResult{err: emperror.Wrap(err, "failed to check emqx availability")}
			}
			if resp.StatusCode == 200 {
				patch = append(patch,
					map[string]interface{}{
						"op":   "add",
						"path": "/status/conditions/-",
						"value": corev1.PodCondition{
							Type:               v1beta4.PodOnServing,
							Status:             corev1.ConditionTrue,
							LastProbeTime:      metav1.Now(),
							LastTransitionTime: metav1.Now(),
						},
					},
				)
			}
			patchBytes, _ := json.Marshal(patch)
			if err := u.Client.Status().Patch(ctx, &pod, client.RawPatch(types.JSONPatchType, patchBytes)); err != nil {
				return subResult{err: emperror.Wrap(err, "failed to update pod conditions")}
			}
		}
	}
	return subResult{}
}

func isPodReady(podConditions map[corev1.PodConditionType]struct{}) bool {
	readyConditions := []corev1.PodConditionType{corev1.ContainersReady, corev1.PodReady, v1beta4.PodInCluster, v1beta4.PodOnServing}
	for _, condition := range readyConditions {
		if _, ok := podConditions[condition]; !ok {
			return false
		}
	}
	return true
}
