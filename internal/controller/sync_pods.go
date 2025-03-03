package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	emperror "emperror.dev/errors"
	appsv2beta1 "github.com/emqx/emqx-operator/api/v2beta1"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type syncPods struct {
	*EMQXReconciler
}

func (s *syncPods) reconcile(ctx context.Context, logger logr.Logger, instance *appsv2beta1.EMQX, r innerReq.RequesterInterface) subResult {
	if r == nil {
		return subResult{}
	}

	if !instance.Status.IsConditionTrue(appsv2beta1.Available) {
		return subResult{}
	}

	updateSts, currentSts, _ := getStateFulSetList(ctx, s.Client, instance)
	updateRs, currentRs, _ := getReplicaSetList(ctx, s.Client, instance)

	targetedEMQXNodesName := []string{}
	if appsv2beta1.IsExistReplicant(instance) {
		if updateRs != nil {
			for _, node := range instance.Status.ReplicantNodes {
				if node.ControllerUID == updateRs.UID {
					targetedEMQXNodesName = append(targetedEMQXNodesName, node.Node)
				}
			}
		}
	} else {
		if updateSts != nil {
			for _, node := range instance.Status.CoreNodes {
				if node.ControllerUID == updateSts.UID {
					targetedEMQXNodesName = append(targetedEMQXNodesName, node.Node)
				}
			}
		}
	}

	if updateRs != nil && currentRs != nil && updateRs.UID != currentRs.UID {
		shouldDeletePod, err := s.canBeScaleDownRs(ctx, instance, r, currentRs, targetedEMQXNodesName)
		if err != nil {
			return subResult{err: emperror.Wrap(err, "failed to check if pod can be scale down")}
		}
		if shouldDeletePod != nil && shouldDeletePod.DeletionTimestamp == nil {
			if shouldDeletePod.Annotations == nil {
				shouldDeletePod.Annotations = make(map[string]string)
			}
			// https://kubernetes.io/docs/concepts/workloads/controllers/replicaset/#pod-deletion-cost
			shouldDeletePod.Annotations["controller.kubernetes.io/pod-deletion-cost"] = "-99999"
			if err := s.Client.Update(ctx, shouldDeletePod); err != nil {
				return subResult{err: emperror.Wrap(err, "failed update pod deletion cost")}
			}

			pod := &corev1.Pod{}
			if err := s.Client.Get(ctx, client.ObjectKeyFromObject(shouldDeletePod), pod); err != nil {
				if !k8sErrors.IsNotFound(err) {
					return subResult{err: emperror.Wrap(err, "failed to get should delete pod")}
				}
			}
			if _, ok := pod.Annotations["controller.kubernetes.io/pod-deletion-cost"]; ok {
				// https://github.com/emqx/emqx-operator/issues/1105
				currentRs.Spec.Replicas = ptr.To(*currentRs.Spec.Replicas - 1)
				if err := s.Client.Update(ctx, currentRs); err != nil {
					return subResult{err: emperror.Wrap(err, "failed to scale down old replicaSet")}
				}
			}

		}
		return subResult{}
	}

	if updateSts != nil && currentSts != nil && updateSts.UID != currentSts.UID {
		canBeScaledDown, err := s.canBeScaleDownSts(ctx, instance, r, currentSts, targetedEMQXNodesName)
		if err != nil {
			return subResult{err: emperror.Wrap(err, "failed to check if sts can be scale down")}
		}
		if canBeScaledDown {
			// https://github.com/emqx/emqx-operator/issues/1105
			currentSts.Spec.Replicas = ptr.To(*currentSts.Spec.Replicas - 1)
			if err := s.Client.Update(ctx, currentSts); err != nil {
				return subResult{err: emperror.Wrap(err, "failed to scale down old statefulSet")}
			}
		}
		return subResult{}
	}
	return subResult{}
}

func (s *syncPods) canBeScaleDownRs(
	ctx context.Context,
	instance *appsv2beta1.EMQX,
	r innerReq.RequesterInterface,
	oldRs *appsv1.ReplicaSet,
	targetedEMQXNodesName []string,
) (*corev1.Pod, error) {
	var shouldDeletePod *corev1.Pod
	var shouldDeletePodInfo *appsv2beta1.EMQXNode
	var err error

	if !checkInitialDelaySecondsReady(instance) {
		return nil, nil
	}

	oldRsPods := getRsPodMap(ctx, s.Client, instance)[oldRs.UID]
	if len(oldRsPods) == 0 {
		return nil, nil
	}

	if len(instance.Status.NodeEvacuationsStatus) > 0 {
		if instance.Status.NodeEvacuationsStatus[0].State != "prohibiting" {
			return nil, nil
		}
		emqxNode := instance.Status.NodeEvacuationsStatus[0].Node
	FindPod:
		for _, node := range instance.Status.ReplicantNodes {
			if node.Node == emqxNode {
				shouldDeletePodUID := node.PodUID
				for _, pod := range oldRsPods {
					if pod.UID == shouldDeletePodUID {
						shouldDeletePod = pod.DeepCopy()
						shouldDeletePodInfo = &node
						break FindPod
					}
				}
			}
		}
	} else {
		sort.Sort(PodsByNameOlder(oldRsPods))
		shouldDeletePod = oldRsPods[0].DeepCopy()
		for _, pod := range oldRsPods {
			if pod.DeletionTimestamp != nil {
				return pod.DeepCopy(), nil
			}
			if _, ok := pod.Annotations["controller.kubernetes.io/pod-deletion-cost"]; ok {
				return pod.DeepCopy(), nil
			}
		}

		shouldDeletePodInfo, err = getEMQXNodeInfoByAPI(r, fmt.Sprintf("emqx@%s", shouldDeletePod.Status.PodIP))
		if err != nil {
			return nil, emperror.Wrap(err, "failed to get node info by API")
		}

		if shouldDeletePodInfo.NodeStatus == "stopped" {
			return shouldDeletePod, nil
		}
	}

	if shouldDeletePodInfo.Edition == appsv2beta1.EnterpriseEdition && shouldDeletePodInfo.Session > 0 {
		if err := startEvacuationByAPI(r, instance, targetedEMQXNodesName, shouldDeletePodInfo.Node); err != nil {
			return nil, emperror.Wrap(err, "failed to start node evacuation")
		}
		s.EventRecorder.Event(instance, corev1.EventTypeNormal, "NodeEvacuation", fmt.Sprintf("Node %s is being evacuated", shouldDeletePodInfo.Node))
		return nil, nil
	}

	// Open Source or Enterprise with no session
	if !checkWaitTakeoverReady(instance, getEventList(ctx, s.Clientset, oldRs)) {
		return nil, nil
	}
	return shouldDeletePod, nil
}

func (s *syncPods) canBeScaleDownSts(
	ctx context.Context,
	instance *appsv2beta1.EMQX,
	r innerReq.RequesterInterface,
	oldSts *appsv1.StatefulSet,
	targetedEMQXNodesName []string,
) (bool, error) {
	var shouldDeletePod *corev1.Pod
	var shouldDeletePodInfo *appsv2beta1.EMQXNode
	var err error

	if appsv2beta1.IsExistReplicant(instance) {
		if instance.Status.ReplicantNodesStatus.CurrentRevision != instance.Status.ReplicantNodesStatus.UpdateRevision {
			return false, nil
		}
	}

	if !checkInitialDelaySecondsReady(instance) {
		return false, nil
	}

	if len(instance.Status.NodeEvacuationsStatus) > 0 {
		if instance.Status.NodeEvacuationsStatus[0].State != "prohibiting" {
			return false, nil
		}
	}

	shouldDeletePod = &corev1.Pod{}
	_ = s.Client.Get(ctx, types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      fmt.Sprintf("%s-%d", oldSts.Name, *oldSts.Spec.Replicas-1),
	}, shouldDeletePod)

	if shouldDeletePod.DeletionTimestamp != nil {
		return false, nil
	}

	shouldDeletePodInfo, err = getEMQXNodeInfoByAPI(r, fmt.Sprintf("emqx@%s.%s.%s.svc.cluster.local", shouldDeletePod.Name, oldSts.Spec.ServiceName, oldSts.Namespace))
	if err != nil {
		return false, emperror.Wrap(err, "failed to get node info by API")
	}

	if shouldDeletePodInfo.NodeStatus == "stopped" {
		return true, nil
	}

	if shouldDeletePodInfo.Edition == appsv2beta1.EnterpriseEdition && shouldDeletePodInfo.Session > 0 {
		if err := startEvacuationByAPI(r, instance, targetedEMQXNodesName, shouldDeletePodInfo.Node); err != nil {
			return false, emperror.Wrap(err, "failed to start node evacuation")
		}
		s.EventRecorder.Event(instance, corev1.EventTypeNormal, "NodeEvacuation", fmt.Sprintf("Node %s is being evacuated", shouldDeletePodInfo.Node))
		return false, nil
	}
	// Open Source or Enterprise with no session
	if !checkWaitTakeoverReady(instance, getEventList(ctx, s.Clientset, oldSts)) {
		return false, nil
	}
	return true, nil
}

func getEMQXNodeInfoByAPI(r innerReq.RequesterInterface, nodeName string) (*appsv2beta1.EMQXNode, error) {
	url := r.GetURL(fmt.Sprintf("api/v5/nodes/%s", nodeName))

	resp, body, err := r.Request("GET", url, nil, nil)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get API api/v5/nodes")
	}
	if resp.StatusCode == 404 {
		return &appsv2beta1.EMQXNode{
			Node:       nodeName,
			NodeStatus: "stopped",
		}, nil
	}
	if resp.StatusCode != 200 {
		return nil, emperror.Errorf("failed to get API %s, status : %s, body: %s", url.String(), resp.Status, body)
	}

	nodeInfo := &appsv2beta1.EMQXNode{}
	if err := json.Unmarshal(body, &nodeInfo); err != nil {
		return nil, emperror.Wrap(err, "failed to unmarshal node statuses")
	}
	return nodeInfo, nil
}

func startEvacuationByAPI(r innerReq.RequesterInterface, instance *appsv2beta1.EMQX, migrateTo []string, nodeName string) error {
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

	url := r.GetURL("api/v5/load_rebalance/" + nodeName + "/evacuation/start")
	resp, respBody, err := r.Request("POST", url, b, nil)
	if err != nil {
		return emperror.Wrap(err, "failed to request API api/v5/load_rebalance/"+nodeName+"/evacuation/start")
	}
	// TODO:
	// the api/v5/load_rebalance/global_status have some bugs, so we need to ignore the 400 error
	// wait for EMQX Dev Team fix it.
	if resp.StatusCode == 400 && strings.Contains(string(respBody), "already_started") {
		return nil
	}
	if resp.StatusCode != 200 {
		return emperror.Errorf("failed to request API %s, status : %s, body: %s", url.String(), resp.Status, respBody)
	}
	return nil
}
