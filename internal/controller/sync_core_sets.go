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

type syncCoreSets struct {
	*EMQXReconciler
}

type scaleDownCore struct {
	Pod    *corev1.Pod
	Reason string
}

func (s *syncCoreSets) reconcile(r *reconcileRound, instance *crdv2.EMQX) subResult {
	updateSts := r.state.updateCoreSet(instance)
	currentSts := r.state.currentCoreSet(instance)
	if updateSts == nil || currentSts == nil {
		return subResult{}
	}
	if updateSts.UID != currentSts.UID {
		return s.migrateSet(r, instance, currentSts)
	}
	return s.scaleSet(r, instance, currentSts)
}

// Orchestrates gradual scale down of the old statefulSet, by migrating workloads to the new statefulSet.
func (s *syncCoreSets) migrateSet(
	r *reconcileRound,
	instance *crdv2.EMQX,
	current *appsv1.StatefulSet,
) subResult {
	admission, err := s.chooseScaleDownCore(r, instance, current)
	if err != nil {
		return subResult{err: err}
	}
	if admission.Pod != nil {
		r.log.V(1).Info("migrating coreSet", "pod", klog.KObj(admission.Pod), "statefulSet", klog.KObj(current))

		*current.Spec.Replicas = *current.Spec.Replicas - 1
		if err := s.Client.Update(r.ctx, current); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to scale down old statefulSet")}
		}
	}
	if admission.Reason != "" {
		r.log.V(1).Info("migrate coreSet skipped", "reason", admission.Reason, "statefulSet", klog.KObj(current))
	}
	return subResult{}
}

// Scale up or down the existing statefulSet.
func (s *syncCoreSets) scaleSet(
	r *reconcileRound,
	instance *crdv2.EMQX,
	current *appsv1.StatefulSet,
) subResult {
	desiredReplicas := *instance.Spec.CoreTemplate.Spec.Replicas
	currentReplicas := *current.Spec.Replicas

	if currentReplicas < desiredReplicas {
		r.log.V(1).Info("scaling up coreSet", "statefulSet", klog.KObj(current), "desiredReplicas", desiredReplicas)
		*current.Spec.Replicas = desiredReplicas
		if err := s.Client.Update(r.ctx, current); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to scale up statefulSet")}
		}
		return subResult{}
	}

	if currentReplicas > desiredReplicas {
		admission, err := s.chooseScaleDownCore(r, instance, current)
		if err != nil {
			return subResult{err: err}
		}
		if admission.Pod != nil {
			r.log.V(1).Info("scaling down coreSet", "pod", klog.KObj(admission.Pod), "statefulSet", klog.KObj(current))
			*current.Spec.Replicas = *current.Spec.Replicas - 1
			if err := s.Client.Update(r.ctx, current); err != nil {
				return subResult{err: emperror.Wrap(err, "failed to scale down statefulSet")}
			}
			return subResult{}
		}
		if admission.Reason != "" {
			r.log.V(1).Info("scale down coreSet skipped", "reason", admission.Reason, "statefulSet", klog.KObj(current))
		}
	}

	return subResult{}
}

func (s *syncCoreSets) chooseScaleDownCore(
	r *reconcileRound,
	instance *crdv2.EMQX,
	current *appsv1.StatefulSet,
) (scaleDownCore, error) {
	// Disallow scaling down the statefulSet if replcants replicaSet is still updating.
	status := &instance.Status
	if instance.Spec.HasReplicants() {
		if status.ReplicantNodesStatus.CurrentRevision != status.ReplicantNodesStatus.UpdateRevision {
			return scaleDownCore{Reason: "replicant replicaSet is still updating"}, nil
		}
	}

	if !checkInitialDelaySecondsReady(instance) {
		return scaleDownCore{Reason: "instance is not ready"}, nil
	}

	if len(status.NodeEvacuationsStatus) > 0 {
		if status.NodeEvacuationsStatus[0].State != "prohibiting" {
			return scaleDownCore{Reason: "node evacuation is still in progress"}, nil
		}
	}

	// List the pods managed by the current coreSet.
	pods := r.state.podsManagedBy(current)
	sortByName(pods)

	// No more pods, no need to scale down.
	if len(pods) == 0 {
		return scaleDownCore{Reason: "no more pods"}, nil
	}

	// Get the pod to be scaled down next.
	scaleDownPod := pods[len(pods)-1]

	// Disallow scaling down the pod that is already being deleted.
	if scaleDownPod.DeletionTimestamp != nil {
		return scaleDownCore{Reason: fmt.Sprintf("pod %s deletion in progress", scaleDownPod.Name)}, nil
	}

	// Disallow scaling down the pod that is still a DS replication site.
	dsCondition := util.FindPodCondition(scaleDownPod, crdv2.DSReplicationSite)
	if dsCondition != nil && dsCondition.Status != corev1.ConditionFalse {
		return scaleDownCore{Reason: fmt.Sprintf("pod %s is still a DS replication site", scaleDownPod.Name)}, nil
	}

	// Get the node info of the pod to be scaled down.
	var scaleDownNode *crdv2.EMQXNode
	for _, node := range instance.Status.CoreNodes {
		if node.PodName == scaleDownPod.Name {
			scaleDownNode = &node
			break
		}
	}

	// If the cluster lacks information about the node, there's very likely nothing to migrate.
	if scaleDownNode == nil {
		return scaleDownCore{Pod: scaleDownPod, Reason: "node is out of cluster"}, nil
	}

	// Scale down the node that is already stopped.
	if scaleDownNode.Status == "stopped" {
		return scaleDownCore{Pod: scaleDownPod, Reason: "node is already stopped"}, nil
	}

	// Disallow scaling down the node that has at least one session.
	if scaleDownNode.Sessions > 0 {
		nodeName := scaleDownNode.Name
		strategy := instance.Spec.UpdateStrategy.EvacuationStrategy
		migrateTo := migrationTargetNodes(r, instance)
		if len(migrateTo) == 0 {
			return scaleDownCore{Reason: fmt.Sprintf("no nodes to migrate %s to", nodeName)}, nil
		}
		err := api.StartEvacuation(r.oldestCoreRequester(), strategy, migrateTo, nodeName)
		if err != nil {
			return scaleDownCore{}, emperror.Wrap(err, "failed to start node evacuation")
		}
		s.EventRecorder.Event(instance, corev1.EventTypeNormal, "NodeEvacuation", fmt.Sprintf("Node %s evacuation started", nodeName))
		return scaleDownCore{Reason: fmt.Sprintf("node %s needs to be evacuated", nodeName)}, nil
	}

	return scaleDownCore{Pod: scaleDownPod}, nil
}

// Returns the list of nodes to migrate workloads to.
func migrationTargetNodes(r *reconcileRound, instance *crdv2.EMQX) []string {
	targets := []string{}
	if instance.Spec.HasReplicants() {
		for _, node := range instance.Status.ReplicantNodes {
			pod := r.state.podWithName(node.PodName)
			if pod != nil && r.state.partOfUpdateSet(pod, instance) {
				targets = append(targets, node.Name)
			}
		}
	} else {
		for _, node := range instance.Status.CoreNodes {
			pod := r.state.podWithName(node.PodName)
			if pod != nil && r.state.partOfUpdateSet(pod, instance) {
				targets = append(targets, node.Name)
			}
		}
	}
	return targets
}
