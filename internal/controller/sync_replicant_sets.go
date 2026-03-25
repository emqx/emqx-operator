package controller

import (
	"fmt"

	emperror "emperror.dev/errors"
	crdv2 "github.com/emqx/emqx-operator/api/v2"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	"github.com/emqx/emqx-operator/internal/emqx/api"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

type syncReplicantSets struct {
	*EMQXReconciler
}

type scaleDownReplicant struct {
	Pod    *corev1.Pod
	Reason string
}

func (s *syncReplicantSets) reconcile(r *reconcileRound, instance *crdv2.EMQX) subResult {
	updateRs := r.state.updateReplicantSet(instance)
	currentRs := r.state.currentReplicantSet(instance)
	if updateRs == nil || currentRs == nil {
		return subResult{}
	}
	if updateRs.UID != currentRs.UID {
		return s.migrateSet(r, instance, currentRs)
	}
	return subResult{}
}

// Orchestrates gradual scale down of the old replicaSet, by migrating workloads to the new replicaSet.
func (s *syncReplicantSets) migrateSet(
	r *reconcileRound,
	instance *crdv2.EMQX,
	current *appsv1.ReplicaSet,
) subResult {
	admission, err := s.chooseScaleDownReplicant(r, instance, current)
	if err != nil {
		return subResult{err: err}
	}
	if admission.Pod != nil {
		r.log.V(1).Info("scaling down replicantSet", "pod", klog.KObj(admission.Pod), "replicaSet", klog.KObj(current))

		if admission.Pod.Annotations == nil {
			admission.Pod.Annotations = make(map[string]string)
		}

		// https://kubernetes.io/docs/concepts/workloads/controllers/replicaset/#pod-deletion-cost
		admission.Pod.Annotations["controller.kubernetes.io/pod-deletion-cost"] = "-99999"
		if err := s.Client.Update(r.ctx, admission.Pod); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to update pod deletion cost")}
		}

		// https://github.com/emqx/emqx-operator/issues/1105
		*current.Spec.Replicas = *current.Spec.Replicas - 1
		if err := s.Client.Update(r.ctx, current); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to scale down current replicantSet")}
		}
	}
	if admission.Reason != "" {
		r.log.V(1).Info("scale down replicantSet skipped", "reason", admission.Reason, "replicaSet", klog.KObj(current))
	}
	return subResult{}
}

func (s *syncReplicantSets) chooseScaleDownReplicant(
	r *reconcileRound,
	instance *crdv2.EMQX,
	current *appsv1.ReplicaSet,
) (scaleDownReplicant, error) {
	var scaleDownPod *corev1.Pod
	var scaleDownNode *crdv2.EMQXNode
	status := &instance.Status

	// Disallow scaling down the replicaSet if replicants just recently became ready.
	if !r.state.areReplicantsAvailable(instance) {
		return scaleDownReplicant{Reason: "replicants are not available yet"}, nil
	}

	// Nothing to do if the replicaSet has no pods.
	currentPods := r.state.podsManagedBy(current)
	sortByName(currentPods)
	if len(currentPods) == 0 {
		return scaleDownReplicant{Reason: "no more pods"}, nil
	}

	// If a pod is already being deleted, return it.
	for _, pod := range currentPods {
		if pod.DeletionTimestamp != nil {
			return scaleDownReplicant{Reason: fmt.Sprintf("pod %s deletion in progress", pod.Name)}, nil
		}
		if _, ok := pod.Annotations["controller.kubernetes.io/pod-deletion-cost"]; ok {
			return scaleDownReplicant{Pod: pod, Reason: "pod already marked for deletion"}, nil
		}
	}

	if len(status.NodeEvacuations) > 0 {
		evacuatingNode := status.NodeEvacuations[0]
		if evacuatingNode.State != "prohibiting" {
			return scaleDownReplicant{Reason: fmt.Sprintf("node %s evacuation in progress", evacuatingNode.NodeName)}, nil
		}
		for _, node := range status.ReplicantNodes {
			if node.Name == evacuatingNode.NodeName && node.PodName != "" {
				scaleDownNode = &node
				scaleDownPod = r.state.podWithName(node.PodName)
				break
			}
		}
		if scaleDownNode == nil {
			return scaleDownReplicant{Reason: fmt.Sprintf("evacuated node %s already deleted", evacuatingNode.NodeName)}, nil
		}
	} else {
		// If there is no node evacuation, return the oldest pod.
		scaleDownPod = currentPods[0]
		for _, node := range status.ReplicantNodes {
			if node.PodName == scaleDownPod.Name {
				scaleDownNode = &node
				break
			}
		}
		if scaleDownNode == nil {
			return scaleDownReplicant{}, emperror.Errorf("node is missing for pod %s", scaleDownPod.Name)
		}
		// If the pod is already stopped, return it.
		if scaleDownNode.Status == "stopped" {
			return scaleDownReplicant{Pod: scaleDownPod, Reason: "pod is already stopped"}, nil
		}
	}

	// Disallow scaling down the pod that is still a DS replication site.
	// While replicants are not supposed to be DS replication sites, check it for safety.
	dsCondition := util.FindPodCondition(scaleDownPod, crdv2.DSReplicationSite)
	if dsCondition != nil && dsCondition.Status != corev1.ConditionFalse {
		return scaleDownReplicant{Reason: fmt.Sprintf("pod %s is still a DS replication site", scaleDownPod.Name)}, nil
	}

	// If the pod has at least one session, start node evacuation.
	if scaleDownNode.Sessions > 0 {
		nodeName := scaleDownNode.Name
		strategy := instance.Spec.UpdateStrategy.EvacuationStrategy
		migrateTo := migrationTargetNodes(r, instance)
		if len(migrateTo) == 0 {
			return scaleDownReplicant{Reason: fmt.Sprintf("no nodes to migrate %s to", nodeName)}, nil
		}
		err := api.StartEvacuation(r.oldestCoreRequester(), strategy, migrateTo, nodeName)
		if err != nil {
			return scaleDownReplicant{}, emperror.Wrap(err, "failed to start node evacuation")
		}
		s.EventRecorder.Event(instance, corev1.EventTypeNormal, "NodeEvacuation", fmt.Sprintf("Node %s evacuation started", nodeName))
		return scaleDownReplicant{Reason: fmt.Sprintf("node %s evacuation started", nodeName)}, nil
	}

	return scaleDownReplicant{Pod: scaleDownPod}, nil
}
