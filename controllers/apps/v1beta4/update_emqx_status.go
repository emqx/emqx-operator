package v1beta4

import (
	"context"
	"encoding/json"

	emperror "emperror.dev/errors"
	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	"github.com/tidwall/gjson"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type updateEmqxStatus struct {
	*EmqxReconciler
	*portForwardAPI
}

func (s updateEmqxStatus) reconcile(ctx context.Context, instance appsv1beta4.Emqx, _ ...any) subResult {
	if err := s.updateReadyReplicas(instance); err != nil {
		return subResult{cont: true, err: emperror.Wrap(err, "failed to update ready replicas")}
	}
	if err := s.updateCondition(instance); err != nil {
		return subResult{cont: true, err: emperror.Wrap(err, "failed to update condition")}
	}
	if err := s.Client.Status().Update(ctx, instance); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to update emqx status")}
	}

	return subResult{}
}

func (s updateEmqxStatus) updateReadyReplicas(instance appsv1beta4.Emqx) error {
	emqxNodes, err := s.getNodeStatusesByAPI()
	if err != nil {
		return emperror.Wrap(err, "failed to get node statuses")
	}

	readyReplicas := int32(0)
	for _, node := range emqxNodes {
		if node.NodeStatus == "Running" {
			readyReplicas++
		}
	}
	instance.GetStatus().SetEmqxNodes(emqxNodes)
	instance.GetStatus().SetReadyReplicas(readyReplicas)
	instance.GetStatus().SetReplicas(*instance.GetSpec().GetReplicas())
	return nil
}

func (s updateEmqxStatus) updateCondition(instance appsv1beta4.Emqx) error {
	inClusterStss, err := getInClusterStatefulSets(s.Client, instance)
	if err != nil {
		return emperror.Wrap(err, "failed to get in cluster statefulsets")
	}
	if len(inClusterStss) == 1 {
		instance.GetStatus().SetCurrentStatefulSetVersion(inClusterStss[0].Status.CurrentRevision)
		if instance.GetStatus().GetReadyReplicas() == instance.GetStatus().GetReplicas() {
			instance.GetStatus().AddCondition(
				appsv1beta4.ConditionRunning,
				corev1.ConditionTrue,
				"ClusterReady",
				"All resources are ready",
			)
		} else {
			instance.GetStatus().AddCondition(
				appsv1beta4.ConditionRunning,
				corev1.ConditionFalse,
				"ClusterNotReady",
				"Some resources are not ready",
			)
		}
		if enterprise, ok := instance.(*appsv1beta4.EmqxEnterprise); ok {
			enterprise.Status.EmqxBlueGreenUpdateStatus = nil
		}
	}
	if len(inClusterStss) > 1 {
		enterprise, ok := instance.(*appsv1beta4.EmqxEnterprise)
		if !ok {
			return emperror.New("blueGreenUpdatingStatus only support EmqxEnterprise")
		}

		originSts := inClusterStss[len(inClusterStss)-1]
		currentSts := inClusterStss[len(inClusterStss)-2]

		enterprise.GetStatus().SetCurrentStatefulSetVersion(currentSts.Status.CurrentRevision)
		enterprise.GetStatus().AddCondition(
			appsv1beta4.ConditionBlueGreenUpdating,
			corev1.ConditionTrue,
			"",
			"",
		)

		ok, err := s.checkEndpointSliceIsReady(enterprise, currentSts)
		if err != nil {
			return emperror.Wrap(err, "failed to check endpoint slice is ready")
		}
		if !ok {
			return nil
		}

		if enterprise.Status.EmqxBlueGreenUpdateStatus == nil {
			enterprise.Status.EmqxBlueGreenUpdateStatus = &appsv1beta4.EmqxBlueGreenUpdateStatus{}
		}
		enterprise.Status.EmqxBlueGreenUpdateStatus.CurrentStatefulSet = currentSts.Name
		enterprise.Status.EmqxBlueGreenUpdateStatus.OriginStatefulSet = originSts.Name

		if enterprise.Status.EmqxBlueGreenUpdateStatus.StartedAt == nil {
			now := metav1.Now()
			enterprise.Status.EmqxBlueGreenUpdateStatus.StartedAt = &now
		}

		evacuationsStatus, err := s.getEvacuationStatusByAPI()
		if err != nil {
			return emperror.Wrap(err, "failed to get evacuation status")
		}
		enterprise.Status.EmqxBlueGreenUpdateStatus.EvacuationsStatus = evacuationsStatus
	}
	return nil
}

func (s updateEmqxStatus) checkEndpointSliceIsReady(instance appsv1beta4.Emqx, currentSts *appsv1.StatefulSet) (bool, error) {
	// make sure that only latest ready sts is in endpoints
	endpointSlice := &discoveryv1.EndpointSliceList{}
	if err := s.Client.List(context.Background(), endpointSlice,
		client.InNamespace(instance.GetNamespace()),
		client.MatchingLabels(instance.GetSpec().GetServiceTemplate().Labels),
	); err != nil {
		return false, err
	}

	podMap, _ := getPodMap(s.Client, instance, []*appsv1.StatefulSet{currentSts})

	hitEndpoints := 0
	for _, endpointSlice := range endpointSlice.Items {
		if len(endpointSlice.Endpoints) != int(*instance.GetSpec().GetReplicas()) {
			continue
		}
		for _, endpoint := range endpointSlice.Endpoints {
			if endpoint.Conditions.Ready == nil || !*endpoint.Conditions.Ready {
				continue
			}

			for _, pod := range podMap[currentSts.UID] {
				if endpoint.TargetRef.UID != pod.UID {
					continue
				}
				hitEndpoints++
			}
		}
	}
	if hitEndpoints != len(podMap[currentSts.UID]) {
		// Wait for endpoints to be ready
		return false, nil
	}
	return true, nil
}

// Request API
func (s updateEmqxStatus) getNodeStatusesByAPI() ([]appsv1beta4.EmqxNode, error) {
	_, body, err := s.portForwardAPI.requestAPI("GET", "api/v4/nodes", nil)
	if err != nil {
		return nil, err
	}

	emqxNodes := []appsv1beta4.EmqxNode{}
	data := gjson.GetBytes(body, "data")
	if err := json.Unmarshal([]byte(data.Raw), &emqxNodes); err != nil {
		return nil, emperror.Wrap(err, "failed to unmarshal node statuses")
	}
	return emqxNodes, nil
}

func (s updateEmqxStatus) getEvacuationStatusByAPI() ([]appsv1beta4.EmqxEvacuationStatus, error) {
	_, body, err := s.portForwardAPI.requestAPI("GET", "api/v4/load_rebalance/global_status", nil)
	if err != nil {
		return nil, err
	}

	evacuationStatuses := []appsv1beta4.EmqxEvacuationStatus{}
	data := gjson.GetBytes(body, "evacuations")
	if err := json.Unmarshal([]byte(data.Raw), &evacuationStatuses); err != nil {
		return nil, emperror.Wrap(err, "failed to unmarshal node statuses")
	}
	return evacuationStatuses, nil
}
