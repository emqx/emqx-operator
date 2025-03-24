package v2beta1

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	emperror "emperror.dev/errors"
	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	ds "github.com/emqx/emqx-operator/controllers/apps/v2beta1/ds"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	"github.com/go-logr/logr"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type updateStatus struct {
	*EMQXReconciler
}

func (u *updateStatus) reconcile(ctx context.Context, logger logr.Logger, instance *appsv2beta1.EMQX, r innerReq.RequesterInterface) subResult {
	instance.Status.CoreNodesStatus.Replicas = *instance.Spec.CoreTemplate.Spec.Replicas
	if instance.Spec.ReplicantTemplate != nil {
		instance.Status.ReplicantNodesStatus.Replicas = *instance.Spec.ReplicantTemplate.Spec.Replicas
	}

	if instance.Status.CoreNodesStatus.UpdateRevision != "" && instance.Status.CoreNodesStatus.CurrentRevision == "" {
		instance.Status.CoreNodesStatus.CurrentRevision = instance.Status.CoreNodesStatus.UpdateRevision
	}
	if instance.Status.ReplicantNodesStatus.UpdateRevision != "" && instance.Status.ReplicantNodesStatus.CurrentRevision == "" {
		instance.Status.ReplicantNodesStatus.CurrentRevision = instance.Status.ReplicantNodesStatus.UpdateRevision
	}

	updateSts, currentSts, oldStsList := getStateFulSetList(ctx, u.Client, instance)
	if updateSts != nil {
		if currentSts == nil || (updateSts.UID != currentSts.UID && currentSts.Status.Replicas == 0) {
			var i int
			for i = 0; i < len(oldStsList); i++ {
				if oldStsList[i].Status.Replicas > 0 {
					currentSts = oldStsList[i]
					break
				}
			}
			if i == len(oldStsList) {
				currentSts = updateSts
			}
			instance.Status.CoreNodesStatus.CurrentRevision = currentSts.Labels[appsv2beta1.LabelsPodTemplateHashKey]
			if err := u.Client.Status().Update(ctx, instance); err != nil {
				return subResult{err: emperror.Wrap(err, "failed to update status")}
			}
			return subResult{}
		}
	}

	updateRs, currentRs, oldRsList := getReplicaSetList(ctx, u.Client, instance)
	if updateRs != nil {
		if currentRs == nil || (updateRs.UID != currentRs.UID && currentRs.Status.Replicas == 0) {
			var i int
			for i = 0; i < len(oldRsList); i++ {
				if oldRsList[i].Status.Replicas > 0 {
					currentRs = oldRsList[i]
					break
				}
			}
			if i == len(oldRsList) {
				currentRs = updateRs
			}
			instance.Status.ReplicantNodesStatus.CurrentRevision = currentRs.Labels[appsv2beta1.LabelsPodTemplateHashKey]
			if err := u.Client.Status().Update(ctx, instance); err != nil {
				return subResult{err: emperror.Wrap(err, "failed to update status")}
			}
			return subResult{}
		}
	}

	if r == nil {
		return subResult{}
	}

	// check emqx node status
	coreNodes, replNodes, err := u.getEMQXNodes(ctx, instance, r)
	if err != nil {
		u.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedToGetNodeStatuses", err.Error())
	}

	instance.Status.CoreNodes = coreNodes
	instance.Status.CoreNodesStatus.ReadyReplicas = 0
	instance.Status.CoreNodesStatus.CurrentReplicas = 0
	instance.Status.CoreNodesStatus.UpdateReplicas = 0
	for _, node := range coreNodes {
		if node.NodeStatus == "running" {
			instance.Status.CoreNodesStatus.ReadyReplicas++
		}
		if currentSts != nil && node.ControllerUID == currentSts.UID {
			instance.Status.CoreNodesStatus.CurrentReplicas++
		}
		if updateSts != nil && node.ControllerUID == updateSts.UID {
			instance.Status.CoreNodesStatus.UpdateReplicas++
		}
	}

	instance.Status.ReplicantNodes = replNodes
	instance.Status.ReplicantNodesStatus.ReadyReplicas = 0
	instance.Status.ReplicantNodesStatus.CurrentReplicas = 0
	instance.Status.ReplicantNodesStatus.UpdateReplicas = 0
	for _, node := range replNodes {
		if node.NodeStatus == "running" {
			instance.Status.ReplicantNodesStatus.ReadyReplicas++
		}
		if currentRs != nil && node.ControllerUID == currentRs.UID {
			instance.Status.ReplicantNodesStatus.CurrentReplicas++
		}
		if updateRs != nil && node.ControllerUID == updateRs.UID {
			instance.Status.ReplicantNodesStatus.UpdateReplicas++
		}
	}

	isEnterprise := false
	for _, node := range coreNodes {
		if node.ControllerUID == currentSts.UID && node.Edition == "Enterprise" {
			isEnterprise = true
			break
		}
	}

	if isEnterprise {
		nodeEvacuationsStatus, err := getNodeEvacuationStatusByAPI(r)
		if err != nil {
			u.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedToGetNodeEvacuationStatuses", err.Error())
		}
		instance.Status.NodeEvacuationsStatus = nodeEvacuationsStatus
	}

	if isEnterprise {
		dsReplicationStatus, err := ds.GetReplicationStatus(r)
		if err != nil {
			u.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedToGetDSReplicationStatus", err.Error())
		}
		status := appsv2beta1.DSReplicationStatus{
			DBs: []appsv2beta1.DSDBReplicationStatus{},
		}
		for _, db := range dsReplicationStatus.DBs {
			minReplicas := 0
			maxReplicas := 0
			numTransitions := 0
			if len(db.Shards) > 0 {
				minReplicas = len(db.Shards[0].Replicas)
				maxReplicas = len(db.Shards[0].Replicas)
			}
			for _, shard := range db.Shards {
				minReplicas = min(minReplicas, len(shard.Replicas))
				maxReplicas = max(maxReplicas, len(shard.Replicas))
				numTransitions += len(shard.Transitions)
			}
			status.DBs = append(status.DBs, appsv2beta1.DSDBReplicationStatus{
				Name:           db.Name,
				NumShards:      int32(len(db.Shards)),
				NumTransitions: int32(numTransitions),
				MinReplicas:    int32(minReplicas),
				MaxReplicas:    int32(maxReplicas),
			})
		}
		instance.Status.DSReplication = status
	}

	// update status condition
	newEMQXStatusMachine(u.Client, instance).NextStatus(ctx)

	if err := u.Client.Status().Update(ctx, instance); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to update status")}
	}
	return subResult{}
}

func (u *updateStatus) getEMQXNodes(ctx context.Context, instance *appsv2beta1.EMQX, r innerReq.RequesterInterface) (coreNodes, replicantNodes []appsv2beta1.EMQXNode, err error) {
	emqxNodes, err := getEMQXNodesByAPI(r)
	if err != nil {
		return nil, nil, emperror.Wrap(err, "failed to get node statues by API")
	}

	list := &corev1.PodList{}
	_ = u.Client.List(ctx, list,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(appsv2beta1.DefaultLabels(instance)),
	)
	for _, node := range emqxNodes {
		for _, p := range list.Items {
			pod := p.DeepCopy()
			host := strings.Split(node.Node[strings.Index(node.Node, "@")+1:], ":")[0]
			if node.Role == "core" && strings.HasPrefix(host, pod.Name) {
				node.PodName = pod.Name
				node.PodUID = pod.UID
				controllerRef := metav1.GetControllerOf(pod)
				if controllerRef == nil {
					continue
				}
				node.ControllerUID = controllerRef.UID
				coreNodes = append(coreNodes, node)
			}
			if node.Role == "replicant" && host == pod.Status.PodIP {
				node.PodName = pod.Name
				node.PodUID = pod.UID
				controllerRef := metav1.GetControllerOf(pod)
				if controllerRef == nil {
					continue
				}
				node.ControllerUID = controllerRef.UID
				replicantNodes = append(replicantNodes, node)
			}
		}
	}

	sort.Slice(coreNodes, func(i, j int) bool {
		return coreNodes[i].Uptime < coreNodes[j].Uptime
	})
	sort.Slice(replicantNodes, func(i, j int) bool {
		return replicantNodes[i].Uptime < replicantNodes[j].Uptime
	})
	return
}

func getEMQXNodesByAPI(r innerReq.RequesterInterface) ([]appsv2beta1.EMQXNode, error) {
	url := r.GetURL("api/v5/nodes")

	resp, body, err := r.Request("GET", url, nil, nil)
	if err != nil {
		return nil, emperror.Wrapf(err, "failed to get API %s", url.String())
	}
	if resp.StatusCode != 200 {
		return nil, emperror.Errorf("failed to get API %s, status : %s, body: %s", url.String(), resp.Status, body)
	}

	nodeStatuses := []appsv2beta1.EMQXNode{}
	if err := json.Unmarshal(body, &nodeStatuses); err != nil {
		return nil, emperror.Wrap(err, "failed to unmarshal node statuses")
	}
	return nodeStatuses, nil
}

func getNodeEvacuationStatusByAPI(r innerReq.RequesterInterface) ([]appsv2beta1.NodeEvacuationStatus, error) {
	url := r.GetURL("api/v5/load_rebalance/global_status")
	resp, body, err := r.Request("GET", url, nil, nil)
	if err != nil {
		return nil, emperror.Wrapf(err, "failed to get API %s", url.String())
	}
	if resp.StatusCode != 200 {
		return nil, emperror.Errorf("failed to get API %s, status : %s, body: %s", url.String(), resp.Status, body)
	}

	nodeEvacuationStatuses := []appsv2beta1.NodeEvacuationStatus{}
	data := gjson.GetBytes(body, "evacuations")
	if err := json.Unmarshal([]byte(data.Raw), &nodeEvacuationStatuses); err != nil {
		return nil, emperror.Wrap(err, "failed to unmarshal node statuses")
	}
	return nodeEvacuationStatuses, nil
}
