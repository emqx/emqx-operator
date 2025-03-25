package v2beta1

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	emperror "emperror.dev/errors"
	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	req "github.com/emqx/emqx-operator/internal/requester"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type syncPods struct {
	*EMQXReconciler
}

type syncPodsReconciliation struct {
	*syncPods
	instance   *appsv2beta1.EMQX
	updateSts  *appsv1.StatefulSet
	currentSts *appsv1.StatefulSet
	updateRs   *appsv1.ReplicaSet
	currentRs  *appsv1.ReplicaSet
}

type scaleDownAdmission struct {
	Pod    *corev1.Pod
	Reason string
}

func (s *syncPods) reconcile(ctx context.Context, logger logr.Logger, instance *appsv2beta1.EMQX, r req.RequesterInterface) subResult {
	result := subResult{}

	if r == nil {
		return result
	}
	if !instance.Status.IsConditionTrue(appsv2beta1.Available) {
		return result
	}

	updateSts, currentSts, _ := getStateFulSetList(ctx, s.Client, instance)
	updateRs, currentRs, _ := getReplicaSetList(ctx, s.Client, instance)
	rec := &syncPodsReconciliation{
		syncPods:   s,
		instance:   instance,
		updateSts:  updateSts,
		currentSts: currentSts,
		updateRs:   updateRs,
		currentRs:  currentRs,
	}

	result = rec.reconcileReplicaSets(ctx, r)
	if result.err != nil {
		return result
	}

	result = rec.reconcileStatefulSets(ctx, r)
	return result
}

func (r *syncPodsReconciliation) reconcileReplicaSets(ctx context.Context, req req.RequesterInterface) subResult {
	if r.updateRs == nil || r.currentRs == nil {
		return subResult{}
	}
	if r.updateRs.UID != r.currentRs.UID {
		return r.migrateReplicaSet(ctx, req)
	}
	return subResult{}
}

func (r *syncPodsReconciliation) reconcileStatefulSets(ctx context.Context, req req.RequesterInterface) subResult {
	if r.updateSts == nil || r.currentSts == nil {
		return subResult{}
	}
	if r.updateSts.UID != r.currentSts.UID {
		return r.migrateStatefulSet(ctx, req)
	}
	return r.scaleStatefulSet(ctx, req)
}

// Orchestrates gradual scale down of the old replicaSet, by migrating workloads to the new replicaSet.
func (r *syncPodsReconciliation) migrateReplicaSet(ctx context.Context, req req.RequesterInterface) subResult {
	admission, err := r.canScaleDownReplicaSet(ctx, req)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to check if old replicaSet can be scaled down")}
	}
	if admission.Pod != nil && admission.Pod.DeletionTimestamp == nil {
		if admission.Pod.Annotations == nil {
			admission.Pod.Annotations = make(map[string]string)
		}

		// https://kubernetes.io/docs/concepts/workloads/controllers/replicaset/#pod-deletion-cost
		admission.Pod.Annotations["controller.kubernetes.io/pod-deletion-cost"] = "-99999"
		if err := r.Client.Update(ctx, admission.Pod); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to update pod deletion cost")}
		}

		pod := &corev1.Pod{}
		if err := r.Client.Get(ctx, client.ObjectKeyFromObject(admission.Pod), pod); err != nil {
			if !k8sErrors.IsNotFound(err) {
				return subResult{err: emperror.Wrap(err, "failed to get pod to be scaled down")}
			}
		}
		if _, ok := pod.Annotations["controller.kubernetes.io/pod-deletion-cost"]; ok {
			// https://github.com/emqx/emqx-operator/issues/1105
			*r.currentRs.Spec.Replicas = *r.currentRs.Spec.Replicas - 1
			if err := r.Client.Update(ctx, r.currentRs); err != nil {
				return subResult{err: emperror.Wrap(err, "failed to scale down old replicaSet")}
			}
		}
	}
	return subResult{}
}

// Orchestrates gradual scale down of the old statefulSet, by migrating workloads to the new statefulSet.
func (r *syncPodsReconciliation) migrateStatefulSet(ctx context.Context, req req.RequesterInterface) subResult {
	admission, err := r.canScaleDownStatefulSet(ctx, req)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to check if old statefulSet can be scaled down")}
	}
	if admission.Pod != nil {
		*r.currentSts.Spec.Replicas = *r.currentSts.Spec.Replicas - 1
		if err := r.Client.Update(ctx, r.currentSts); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to scale down old statefulSet")}
		}
	}
	return subResult{}
}

// Returns the list of nodes to migrate workloads to.
func (r *syncPodsReconciliation) migrationTargetNodes() []string {
	targets := []string{}
	if appsv2beta1.IsExistReplicant(r.instance) {
		if r.updateRs != nil {
			for _, node := range r.instance.Status.ReplicantNodes {
				if node.ControllerUID == r.updateRs.UID {
					targets = append(targets, node.Node)
				}
			}
		}
	} else {
		if r.updateSts != nil {
			for _, node := range r.instance.Status.CoreNodes {
				if node.ControllerUID == r.updateSts.UID {
					targets = append(targets, node.Node)
				}
			}
		}
	}
	return targets
}

// Scale up or down the existing statefulSet.
func (r *syncPodsReconciliation) scaleStatefulSet(ctx context.Context, req req.RequesterInterface) subResult {
	sts := r.currentSts
	desiredReplicas := *r.instance.Spec.CoreTemplate.Spec.Replicas
	currentReplicas := *sts.Spec.Replicas

	if currentReplicas < desiredReplicas {
		*sts.Spec.Replicas = desiredReplicas
		if err := r.Client.Update(ctx, sts); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to scale up statefulSet")}
		}
		return subResult{}
	}

	if currentReplicas > desiredReplicas {
		admission, err := r.canScaleDownStatefulSet(ctx, req)
		if err != nil {
			return subResult{err: emperror.Wrap(err, "failed to check if statefulSet can be scaled down")}
		}
		if admission.Pod != nil {
			*sts.Spec.Replicas = *sts.Spec.Replicas - 1
			if err := r.Client.Update(ctx, sts); err != nil {
				return subResult{err: emperror.Wrap(err, "failed to scale down statefulSet")}
			}
			return subResult{}
		}
	}

	return subResult{}
}

func (r *syncPodsReconciliation) canScaleDownReplicaSet(ctx context.Context, req req.RequesterInterface) (scaleDownAdmission, error) {
	var err error
	var scaleDownPod *corev1.Pod
	var scaleDownPodUID types.UID
	var scaleDownNodeName string
	var scaleDownPodInfo *appsv2beta1.EMQXNode
	status := &r.instance.Status

	// Disallow scaling down the replicaSet if the instance just recently became ready.
	if !checkInitialDelaySecondsReady(r.instance) {
		return scaleDownAdmission{Reason: "instance is not ready"}, nil
	}

	// Nothing to do if the replicaSet has no pods.
	currentPods := listPodsManagedBy(ctx, r.Client, r.instance, r.currentRs.UID)
	sort.Sort(PodsByNameOlder(currentPods))
	if len(currentPods) == 0 {
		return scaleDownAdmission{Reason: "no more pods"}, nil
	}

	// If a pod is already being deleted, return it.
	for _, pod := range currentPods {
		if pod.DeletionTimestamp != nil {
			return scaleDownAdmission{Pod: pod, Reason: "pod is being deleted"}, nil
		}
		if _, ok := pod.Annotations["controller.kubernetes.io/pod-deletion-cost"]; ok {
			return scaleDownAdmission{Pod: pod, Reason: "pod is being deleted"}, nil
		}
	}

	if len(status.NodeEvacuationsStatus) > 0 {
		if status.NodeEvacuationsStatus[0].State != "prohibiting" {
			return scaleDownAdmission{Reason: "node evacuation is still in progress"}, nil
		}
		scaleDownNodeName = status.NodeEvacuationsStatus[0].Node
		for _, node := range status.ReplicantNodes {
			if node.Node == scaleDownNodeName {
				scaleDownPodUID = node.PodUID
				scaleDownPodInfo = &node
				break
			}
		}
		for _, pod := range currentPods {
			if pod.UID == scaleDownPodUID {
				scaleDownPod = pod
				break
			}
		}
	} else {
		// If there is no node evacuation, return the oldest pod.
		scaleDownPod = currentPods[0]
		scaleDownNodeName = fmt.Sprintf("emqx@%s", scaleDownPod.Status.PodIP)
		scaleDownPodInfo, err = getEMQXNodeInfoByAPI(req, scaleDownNodeName)
		if err != nil {
			return scaleDownAdmission{}, emperror.Wrap(err, "failed to get node info by API")
		}
		// If the pod is already stopped, return it.
		if scaleDownPodInfo.NodeStatus == "stopped" {
			return scaleDownAdmission{Pod: scaleDownPod, Reason: "node is already stopped"}, nil
		}
	}

	// If the pod runs Enterprise edition and has at least one session, start node evacuation.
	if scaleDownPodInfo.Edition == "Enterprise" && scaleDownPodInfo.Session > 0 {
		migrateTo := r.migrationTargetNodes()
		if err := startEvacuationByAPI(req, r.instance, migrateTo, scaleDownNodeName); err != nil {
			return scaleDownAdmission{Reason: "failed to start node evacuation"}, emperror.Wrap(err, "failed to start node evacuation")
		}
		r.EventRecorder.Event(r.instance, corev1.EventTypeNormal, "NodeEvacuation", fmt.Sprintf("Node %s is being evacuated", scaleDownNodeName))
		return scaleDownAdmission{Reason: "node needs to be evacuated"}, nil
	}

	// Open Source or Enterprise with no session
	if !checkWaitTakeoverReady(r.instance, getEventList(ctx, r.Clientset, r.currentRs)) {
		return scaleDownAdmission{Reason: "node evacuation just finished"}, nil
	}

	return scaleDownAdmission{Pod: scaleDownPod}, nil
}

func (r *syncPodsReconciliation) canScaleDownStatefulSet(ctx context.Context, req req.RequesterInterface) (scaleDownAdmission, error) {
	// Disallow scaling down the statefulSet if replcants replicaSet is still updating.
	status := r.instance.Status
	if appsv2beta1.IsExistReplicant(r.instance) {
		if status.ReplicantNodesStatus.CurrentRevision != status.ReplicantNodesStatus.UpdateRevision {
			return scaleDownAdmission{Reason: "replicant replicaSet is still updating"}, nil
		}
	}

	if !checkInitialDelaySecondsReady(r.instance) {
		return scaleDownAdmission{Reason: "instance is not ready"}, nil
	}

	if len(status.NodeEvacuationsStatus) > 0 {
		if status.NodeEvacuationsStatus[0].State != "prohibiting" {
			return scaleDownAdmission{Reason: "node evacuation is still in progress"}, nil
		}
	}

	// Get the pod to be scaled down next.
	scaleDownPod := &corev1.Pod{}
	err := r.Client.Get(ctx, types.NamespacedName{
		Namespace: r.instance.Namespace,
		Name:      fmt.Sprintf("%s-%d", r.currentSts.Name, *r.currentSts.Spec.Replicas-1),
	}, scaleDownPod)

	// No more pods, no need to scale down.
	if err != nil && k8sErrors.IsNotFound(err) {
		return scaleDownAdmission{Reason: "no more pods"}, nil
	}

	// Disallow scaling down the pod that is already being deleted.
	if scaleDownPod.DeletionTimestamp != nil {
		return scaleDownAdmission{Reason: "pod deletion in progress"}, nil
	}

	// Disallow scaling down the pod that is still a DS replication site.
	// Only if DS is enabled in the current, most recent EMQX config.
	// Otherwise, if the user has disabled DS, the data is apparently no longer
	// needs to be preserved.
	if r.conf.IsDSEnabled() {
		dsCondition := appsv2beta1.FindPodCondition(scaleDownPod, appsv2beta1.DSReplicationSite)
		if dsCondition != nil && dsCondition.Status != corev1.ConditionFalse {
			return scaleDownAdmission{Reason: "pod is still a DS replication site"}, nil
		}
	}

	// Get the node info of the pod to be scaled down.
	scaleDownNodeName := fmt.Sprintf("emqx@%s.%s.%s.svc.cluster.local", scaleDownPod.Name, r.currentSts.Spec.ServiceName, r.currentSts.Namespace)
	scaleDownNode, err := getEMQXNodeInfoByAPI(req, scaleDownNodeName)
	if err != nil {
		return scaleDownAdmission{}, emperror.Wrap(err, "failed to get node info by API")
	}

	// Scale down the node that is already stopped.
	if scaleDownNode.NodeStatus == "stopped" {
		return scaleDownAdmission{Pod: scaleDownPod, Reason: "node is already stopped"}, nil
	}

	// Disallow scaling down the node that is Enterprise and has at least one session.
	if scaleDownNode.Edition == "Enterprise" && scaleDownNode.Session > 0 {
		migrateTo := r.migrationTargetNodes()
		if err := startEvacuationByAPI(req, r.instance, migrateTo, scaleDownNode.Node); err != nil {
			return scaleDownAdmission{}, emperror.Wrap(err, "failed to start node evacuation")
		}
		r.EventRecorder.Event(r.instance, corev1.EventTypeNormal, "NodeEvacuation", fmt.Sprintf("Node %s is being evacuated", scaleDownNode.Node))
		return scaleDownAdmission{Reason: "node needs to be evacuated"}, nil
	}

	// Open Source or Enterprise with no session
	if !checkWaitTakeoverReady(r.instance, getEventList(ctx, r.Clientset, r.currentSts)) {
		return scaleDownAdmission{Reason: "node evacuation just finished"}, nil
	}

	return scaleDownAdmission{Pod: scaleDownPod}, nil
}

func getEMQXNodeInfoByAPI(r req.RequesterInterface, nodeName string) (*appsv2beta1.EMQXNode, error) {
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

func startEvacuationByAPI(r req.RequesterInterface, instance *appsv2beta1.EMQX, migrateTo []string, nodeName string) error {
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
	//TODO:
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
