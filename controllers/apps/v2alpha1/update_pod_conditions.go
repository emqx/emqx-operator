package v2alpha1

import (
	"context"
	"encoding/json"

	emperror "emperror.dev/errors"
	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
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
		hash := make(map[corev1.PodConditionType]struct{})

		for _, condition := range pod.Status.Conditions {
			if condition.Status == corev1.ConditionTrue {
				hash[condition.Type] = struct{}{}
			}
		}

		_, containersReady := hash[corev1.ContainersReady]
		_, podReady := hash[corev1.PodReady]
		_, podInCluster := hash[appsv2alpha1.PodInCluster]
		if containersReady && !podReady && !podInCluster {
			for _, node := range instance.Status.EMQXNodes {
				if node.Node == "emqx@"+pod.Status.PodIP {
					patch := []map[string]interface{}{
						{
							"op":   "add",
							"path": "/status/conditions/-",
							"value": corev1.PodCondition{
								Type:               appsv2alpha1.PodInCluster,
								Status:             corev1.ConditionTrue,
								LastProbeTime:      metav1.Now(),
								LastTransitionTime: metav1.Now(),
							},
						},
					}
					patchBytes, _ := json.Marshal(patch)
					if err := u.Client.Status().Patch(ctx, &pod, client.RawPatch(types.JSONPatchType, patchBytes)); err != nil {
						return subResult{err: emperror.Wrap(err, "failed to update pod conditions")}
					}
				}
			}
		}
	}
	return subResult{}
}
