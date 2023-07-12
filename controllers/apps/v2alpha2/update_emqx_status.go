package v2alpha2

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	emperror "emperror.dev/errors"
	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type updateStatus struct {
	*EMQXReconciler
}

func (u *updateStatus) reconcile(ctx context.Context, instance *appsv2alpha2.EMQX, r innerReq.RequesterInterface) subResult {
	if isExistReplicant(instance) && instance.Status.ReplicantNodesStatus == nil {
		instance.Status.ReplicantNodesStatus = &appsv2alpha2.EMQXNodesStatus{
			Replicas: *instance.Spec.ReplicantTemplate.Spec.Replicas,
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

	updateSts, currentSts, oldStsList := getStateFulSetList(ctx, u.Client, instance)
	if currentSts == nil {
		if len(oldStsList) > 0 {
			currentSts = oldStsList[0]
		} else {
			currentSts = updateSts
		}
		instance.Status.CoreNodesStatus.CurrentRevision = currentSts.Labels[appsv2alpha2.PodTemplateHashLabelKey]
	}

	instance.Status.CoreNodesStatus.Nodes = coreNodes
	instance.Status.CoreNodesStatus.Replicas = *instance.Spec.CoreTemplate.Spec.Replicas
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

	if len(replNodes) > 0 {
		updateRs, currentRs, oldRsList := getReplicaSetList(ctx, u.Client, instance)
		if currentRs == nil {
			if len(oldRsList) > 0 {
				currentRs = oldRsList[0]
			} else {
				currentRs = updateRs
			}
			instance.Status.ReplicantNodesStatus.CurrentRevision = currentRs.Labels[appsv2alpha2.PodTemplateHashLabelKey]
		}

		instance.Status.ReplicantNodesStatus.Nodes = replNodes
		instance.Status.ReplicantNodesStatus.Replicas = *instance.Spec.ReplicantTemplate.Spec.Replicas
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
	}

	isEnterpriser := false
	for _, node := range coreNodes {
		if node.ControllerUID == currentSts.UID && node.Edition == "Enterprise" {
			isEnterpriser = true
			break
		}
	}
	if isEnterpriser {
		nodeEvacuationsStatus, err := getNodeEvacuationStatusByAPI(r)
		if err != nil {
			u.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedToGetNodeEvacuationStatuses", err.Error())
		}
		instance.Status.NodeEvacuationsStatus = nodeEvacuationsStatus
	}

	// update status condition
	newEMQXStatusMachine(instance).NextStatus()

	if err := u.Client.Status().Update(ctx, instance); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to update status")}
	}
	return subResult{}
}

func (u *updateStatus) getEMQXNodes(ctx context.Context, instance *appsv2alpha2.EMQX, r innerReq.RequesterInterface) (coreNodes, replicantNodes []appsv2alpha2.EMQXNode, err error) {
	emqxNodes, err := getEMQXNodesByAPI(r)
	if err != nil {
		return nil, nil, emperror.Wrap(err, "failed to get node statues by API")
	}

	list := &corev1.PodList{}
	_ = u.Client.List(ctx, list,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(instance.Labels),
	)
	for _, node := range emqxNodes {
		for _, p := range list.Items {
			pod := p.DeepCopy()
			host := strings.Split(node.Node[strings.Index(node.Node, "@")+1:], ":")[0]
			if node.Role == "core" && strings.HasPrefix(host, pod.Name) {
				node.PodUID = pod.UID
				controllerRef := metav1.GetControllerOf(pod)
				if controllerRef == nil {
					continue
				}
				node.ControllerUID = controllerRef.UID
				coreNodes = append(coreNodes, node)
			}

			if node.Role == "replicant" && host == pod.Status.PodIP {
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

func getEMQXNodesByAPI(r innerReq.RequesterInterface) ([]appsv2alpha2.EMQXNode, error) {
	url := r.GetURL("api/v5/nodes")

	resp, body, err := r.Request("GET", url, nil, nil)
	if err != nil {
		return nil, emperror.Wrapf(err, "failed to get API %s", url.String())
	}
	if resp.StatusCode != 200 {
		return nil, emperror.Errorf("failed to get API %s, status : %s, body: %s", url.String(), resp.Status, body)
	}

	nodeStatuses := []appsv2alpha2.EMQXNode{}
	if err := json.Unmarshal(body, &nodeStatuses); err != nil {
		return nil, emperror.Wrap(err, "failed to unmarshal node statuses")
	}
	return nodeStatuses, nil
}

func getNodeEvacuationStatusByAPI(r innerReq.RequesterInterface) ([]appsv2alpha2.NodeEvacuationStatus, error) {
	url := r.GetURL("api/v5/load_rebalance/global_status")
	resp, body, err := r.Request("GET", url, nil, nil)
	if err != nil {
		return nil, emperror.Wrapf(err, "failed to get API %s", url.String())
	}
	if resp.StatusCode != 200 {
		return nil, emperror.Errorf("failed to get API %s, status : %s, body: %s", url.String(), resp.Status, body)
	}

	nodeEvacuationStatuses := []appsv2alpha2.NodeEvacuationStatus{}
	data := gjson.GetBytes(body, "evacuations")
	if err := json.Unmarshal([]byte(data.Raw), &nodeEvacuationStatuses); err != nil {
		return nil, emperror.Wrap(err, "failed to unmarshal node statuses")
	}
	return nodeEvacuationStatuses, nil
}
