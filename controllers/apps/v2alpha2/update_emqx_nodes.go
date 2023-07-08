package v2alpha2

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	emperror "emperror.dev/errors"
	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

type updateNodes struct {
	*EMQXReconciler
}

func (u *updateNodes) reconcile(ctx context.Context, instance *appsv2alpha2.EMQX, r innerReq.RequesterInterface) subResult {
	if r == nil {
		return subResult{}
	}

	currentSts, oldStsList := getStateFulSetList(ctx, u.Client, instance)
	currentRs, oldRsList := getReplicaSetList(ctx, u.Client, instance)

	targetedEMQXNodesName := []string{}
	if isExistReplicant(instance) {
		for _, node := range instance.Status.ReplicantNodesStatus.Nodes {
			if node.ControllerUID == currentRs.UID {
				targetedEMQXNodesName = append(targetedEMQXNodesName, node.Node)
			}
		}
	} else {
		for _, node := range instance.Status.CoreNodesStatus.Nodes {
			if node.ControllerUID == currentSts.UID {
				targetedEMQXNodesName = append(targetedEMQXNodesName, node.Node)
			}
		}
	}

	if len(oldRsList) > 0 {
		oldRs := oldRsList[0]
		shouldDeletePod, err := u.canBeScaleDownRs(ctx, instance, r, oldRs, targetedEMQXNodesName)
		if err != nil {
			return subResult{err: emperror.Wrap(err, "failed to check if pod can be scale down")}
		}
		if shouldDeletePod != nil {
			if shouldDeletePod.Annotations == nil {
				shouldDeletePod.Annotations = make(map[string]string)
			}
			// https://kubernetes.io/docs/concepts/workloads/controllers/replicaset/#pod-deletion-cost
			shouldDeletePod.Annotations["controller.kubernetes.io/pod-deletion-cost"] = "-99999"
			if err := u.Client.Update(ctx, shouldDeletePod); err != nil {
				return subResult{err: emperror.Wrap(err, "failed update pod deletion cost")}
			}

			oldRs.Spec.Replicas = pointer.Int32(*oldRs.Spec.Replicas - 1)
			if err := u.Client.Update(ctx, oldRs); err != nil {
				return subResult{err: emperror.Wrap(err, "failed to scale down old replicaSet")}
			}
		}
		return subResult{}
	}
	if len(oldStsList) > 0 {
		oldSts := oldStsList[0]
		canBeScaledDown, err := u.canBeScaleDownSts(ctx, instance, r, oldSts, targetedEMQXNodesName)
		if err != nil {
			return subResult{err: emperror.Wrap(err, "failed to check if sts can be scale down")}
		}
		if canBeScaledDown {
			oldSts.Spec.Replicas = pointer.Int32(*oldSts.Spec.Replicas - 1)
			if err := u.Client.Update(ctx, oldSts); err != nil {
				return subResult{err: emperror.Wrap(err, "failed to scale down old statefulSet")}
			}
		}
		return subResult{}
	}
	return subResult{}
}

func (u *updateNodes) canBeScaleDownSts(ctx context.Context, instance *appsv2alpha2.EMQX, r innerReq.RequesterInterface,
	oldSts *appsv1.StatefulSet, targetedEMQXNodesName []string,
) (bool, error) {
	if isExistReplicant(instance) {
		if instance.Status.ReplicantNodesStatus.ReadyReplicas != *instance.Spec.ReplicantTemplate.Spec.Replicas {
			return false, nil
		}
	}

	if !checkInitialDelaySecondsReady(instance) {
		return false, nil
	}

	shouldDeletePod := &corev1.Pod{}
	_ = u.Client.Get(ctx, types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      fmt.Sprintf("%s-%d", oldSts.Name, *oldSts.Spec.Replicas-1),
	}, shouldDeletePod)

	shouldDeletePodInfo, err := getEMQXNodeInfoByAPI(r, fmt.Sprintf("emqx@%s.%s.%s.svc.cluster.local", shouldDeletePod.Name, oldSts.Spec.ServiceName, oldSts.Namespace))
	if err != nil {
		return false, emperror.Wrap(err, "failed to get node info by API")
	}

	if shouldDeletePodInfo.NodeStatus == "stopped" {
		return true, nil
	}

	if shouldDeletePodInfo.Edition == "Enterprise" {
		if shouldDeletePodInfo.Session > 0 && len(instance.Status.NodeEvacuationsStatus) == 0 {
			if err := startEvacuationByAPI(r, instance, targetedEMQXNodesName, shouldDeletePodInfo.Node); err != nil {
				return false, emperror.Wrap(err, "failed to start node evacuation")
			}
			u.EventRecorder.Event(instance, corev1.EventTypeNormal, "NodeEvacuation", fmt.Sprintf("Node %s is being evacuated", shouldDeletePodInfo.Node))
			return false, nil
		}
	}
	// Open Source or Enterprise with no session
	if !checkWaitTakeoverReady(instance, getEventList(ctx, u.Clientset, oldSts)) {
		return false, nil
	}
	return true, nil
}

func (u *updateNodes) canBeScaleDownRs(ctx context.Context, instance *appsv2alpha2.EMQX, r innerReq.RequesterInterface,
	oldRs *appsv1.ReplicaSet, targetedEMQXNodesName []string,
) (*corev1.Pod, error) {
	if !checkInitialDelaySecondsReady(instance) {
		return nil, nil
	}

	oldRsPods := getRsPodMap(ctx, u.Client, instance)[oldRs.UID]
	sort.Sort(PodsByNameOlder(oldRsPods))
	if len(oldRsPods) == 0 {
		return nil, nil
	}

	shouldDeletePod := oldRsPods[0].DeepCopy()
	shouldDeletePodInfo, err := getEMQXNodeInfoByAPI(r, fmt.Sprintf("emqx@%s", shouldDeletePod.Status.PodIP))
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get node info by API")
	}

	if shouldDeletePodInfo.NodeStatus == "stopped" {
		return shouldDeletePod, nil
	}

	if shouldDeletePodInfo.Edition == "Enterprise" {
		if shouldDeletePodInfo.Session > 0 {
			if len(instance.Status.NodeEvacuationsStatus) == 0 {
				if err := startEvacuationByAPI(r, instance, targetedEMQXNodesName, shouldDeletePodInfo.Node); err != nil {
					return nil, emperror.Wrap(err, "failed to start node evacuation")
				}
			}
			u.EventRecorder.Event(instance, corev1.EventTypeNormal, "NodeEvacuation", fmt.Sprintf("Node %s is being evacuated", shouldDeletePodInfo.Node))
			return nil, nil
		}
	}

	// Open Source or Enterprise with no session
	if !checkWaitTakeoverReady(instance, getEventList(ctx, u.Clientset, oldRs)) {
		return nil, nil
	}
	return shouldDeletePod, nil
}

func getEMQXNodeInfoByAPI(r innerReq.RequesterInterface, nodeName string) (*appsv2alpha2.EMQXNode, error) {
	path := fmt.Sprintf("api/v5/nodes/%s", nodeName)
	resp, body, err := r.Request("GET", path, nil)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get API api/v5/nodes")
	}
	if resp.StatusCode == 404 {
		return &appsv2alpha2.EMQXNode{
			Node:       nodeName,
			NodeStatus: "stopped",
		}, nil
	}
	if resp.StatusCode != 200 {
		return nil, emperror.Errorf("failed to get API %s, status : %s, body: %s", path, resp.Status, body)
	}

	nodeInfo := &appsv2alpha2.EMQXNode{}
	if err := json.Unmarshal(body, &nodeInfo); err != nil {
		return nil, emperror.Wrap(err, "failed to unmarshal node statuses")
	}
	return nodeInfo, nil
}

func startEvacuationByAPI(r innerReq.RequesterInterface, instance *appsv2alpha2.EMQX, migrateTo []string, nodeName string) error {
	body := map[string]interface{}{
		"conn_evict_rate": instance.Spec.UpdateStrategy.EvacuationStrategy.ConnEvictRate,
		"sess_evict_rate": instance.Spec.UpdateStrategy.EvacuationStrategy.SessEvictRate,
		"migrate_to":      migrateTo,
	}
	if instance.Spec.UpdateStrategy.EvacuationStrategy.WaitTakeover > 0 {
		body["wait_takeover"] = instance.Spec.UpdateStrategy.EvacuationStrategy.WaitTakeover
	}

	b, err := json.Marshal(body)
	if err != nil {
		return emperror.Wrap(err, "marshal body failed")
	}

	resp, respBody, err := r.Request("POST", "api/v5/load_rebalance/"+nodeName+"/evacuation/start", b)
	if err != nil {
		return emperror.Wrap(err, "failed to request API api/v5/load_rebalance/"+nodeName+"/evacuation/start")
	}
	//TODO:
	// the api/v5/load_rebalance/global_status have some bugs, so we need to ignore the 400 error
	// wait for EMQX Dev Team fix it.
	if resp.StatusCode == 400 && strings.Contains(string(respBody), "already_started") {
		return nil
	}
	if resp.StatusCode != 200 {
		return emperror.Errorf("failed to request API api/v5/load_rebalance/"+nodeName+"/evacuation/start, status : %s, body: %s", resp.Status, respBody)
	}
	return nil
}
