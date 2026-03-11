package controller

import (
	"fmt"
	"slices"

	emperror "emperror.dev/errors"
	crdv2 "github.com/emqx/emqx-operator/api/v2"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	"github.com/emqx/emqx-operator/internal/emqx/api"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

type syncCoreSet struct {
	*EMQXReconciler
}

type admissionAction int

const (
	// Pod may be removed right now.
	admissionRemove admissionAction = iota
	// Pod removal blocked; reason explains why.
	admissionWait
	// Pod needs evacuation before it can be removed.
	admissionEvacuate
)

type coreAdmission struct {
	Action admissionAction
	Reason string
}

func (s *syncCoreSet) reconcile(r *reconcileRound, instance *crdv2.EMQX) subResult {
	coreSet := r.state.coreSet()
	if coreSet == nil {
		return subResult{}
	}

	desiredReplicas := instance.Spec.NumCoreReplicas()
	currentReplicas := util.NumReplicas(coreSet)

	// Handle scale-up: simply update the StatefulSet replica count.
	if currentReplicas < desiredReplicas {
		r.log.V(1).Info("scaling up coreSet",
			"statefulSet", klog.KObj(coreSet),
			"from", currentReplicas,
			"to", desiredReplicas,
		)
		return s.scaleUp(r, desiredReplicas)
	}

	// Handle scale-down: remove highest-ordinal pod with evacuation gating.
	if currentReplicas > desiredReplicas {
		r.log.V(1).Info("scaling down coreSet",
			"statefulSet", klog.KObj(coreSet),
			"from", currentReplicas,
			"to", desiredReplicas,
		)
		return s.scaleDown(r, instance)
	}

	// Handle rolling update: replace outdated pods one at a time, highest ordinal first.
	return s.rollingUpdate(r, instance)
}

// rollingUpdate detects outdated core pods and replaces them one at a time,
// starting from the highest ordinal. Each pod is evacuated before deletion.
func (s *syncCoreSet) rollingUpdate(r *reconcileRound, instance *crdv2.EMQX) subResult {
	coreSet := r.state.coreSet()

	var outdated []*corev1.Pod
	for _, pod := range r.state.podsManagedBy(coreSet) {
		if !r.state.partOfCoreSetLatestRevision(pod) {
			outdated = append(outdated, pod)
		}
	}

	// Sort by name descending to delete highest ordinal first.
	sortByName(outdated)

	if len(outdated) == 0 {
		return subResult{}
	}

	r.log.V(1).Info("rolling coreSet update",
		"statefulSet", klog.KObj(coreSet),
		"outdatedPods", len(outdated),
	)

	candidate := outdated[len(outdated)-1]
	admission := checkCorePodRemoval(instance, candidate, false)

	return s.onCoreAdmission(r, instance, candidate, admission, "rollingUpdate")
}

func (s *syncCoreSet) scaleUp(r *reconcileRound, desiredReplicas int32) subResult {
	coreSet := r.state.coreSet()
	coreSet.Spec.Replicas = &desiredReplicas
	err := s.Client.Update(r.ctx, coreSet)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to scale up coreSet")}
	}
	return subResult{}
}

// scaleDown removes the highest-ordinal pod with evacuation gating, then
// decrements the StatefulSet replica count.
func (s *syncCoreSet) scaleDown(r *reconcileRound, instance *crdv2.EMQX) subResult {
	pods := r.state.podsManagedBy(r.state.coreSet())
	sortByName(pods)

	if len(pods) == 0 {
		return subResult{}
	}

	// Scale down the highest-ordinal pod.
	candidate := pods[len(pods)-1]
	admission := checkCorePodRemoval(instance, candidate, true)

	return s.onCoreAdmission(r, instance, candidate, admission, "scaleDown")
}

// checkCorePodRemoval is a pure function that decides whether a core pod can
// be safely removed. It inspects instance status and pod state but performs no
// side effects.
//
// When isPermanent is true, the pod is being permanently removed:
// data would be lost so there are extra safety checks (e.g. DS replication site).
// When isPermanent is false (rolling update), the replacement pod inherits the
// data: blocking on DS site condition would stall the rollout indefinitely.
func checkCorePodRemoval(instance *crdv2.EMQX, pod *corev1.Pod, isPermanent bool) coreAdmission {
	status := &instance.Status

	if instance.Spec.HasReplicants() {
		if status.ReplicantNodesStatus.CurrentRevision != status.ReplicantNodesStatus.UpdateRevision {
			return coreAdmission{Action: admissionWait, Reason: "replicant replicaSet is still updating"}
		}
	}

	if !checkInitialDelaySecondsReady(instance) {
		return coreAdmission{Action: admissionWait, Reason: "instance is not ready"}
	}

	if len(status.NodeEvacuations) > 0 {
		if status.NodeEvacuations[0].State != "prohibiting" {
			return coreAdmission{Action: admissionWait, Reason: "node evacuation is still in progress"}
		}
	}

	if pod.DeletionTimestamp != nil {
		return coreAdmission{Action: admissionWait, Reason: fmt.Sprintf("pod %s deletion in progress", pod.Name)}
	}

	if isPermanent {
		dsCondition := util.FindPodCondition(pod, crdv2.DSReplicationSite)
		if dsCondition != nil && dsCondition.Status != corev1.ConditionFalse {
			return coreAdmission{Action: admissionWait, Reason: fmt.Sprintf("pod %s is still a DS replication site", pod.Name)}
		}
	}

	var nodeInfo *crdv2.EMQXNode
	for _, node := range status.CoreNodes {
		if node.PodName == pod.Name {
			nodeInfo = &node
			break
		}
	}

	if nodeInfo == nil {
		return coreAdmission{Action: admissionRemove, Reason: "node is out of cluster"}
	}

	if nodeInfo.Status == "stopped" {
		return coreAdmission{Action: admissionRemove, Reason: "node is already stopped"}
	}

	if nodeInfo.Sessions > 0 {
		return coreAdmission{Action: admissionEvacuate, Reason: fmt.Sprintf("node %s has active sessions", nodeInfo.Name)}
	}

	return coreAdmission{Action: admissionRemove}
}

// onCoreAdmission performs the side effects implied by a coreAdmission.
// Currently this only handles coreAdmitEvacuate by calling the evacuation API.
func (s *syncCoreSet) onCoreAdmission(
	r *reconcileRound,
	instance *crdv2.EMQX,
	candidate *corev1.Pod,
	admission coreAdmission,
	cause string,
) subResult {
	switch admission.Action {
	case admissionRemove:
		r.log.V(1).Info("removing core pod",
			"pod", klog.KObj(candidate),
			"statefulSet", klog.KObj(r.state.coreSet()),
			"cause", cause,
		)
		if err := s.Client.Delete(r.ctx, candidate); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to delete core pod")}
		}
	case admissionWait:
		r.log.V(1).Info("removal of core pod postponed",
			"reason", admission.Reason,
			"pod", klog.KObj(candidate),
			"statefulSet", klog.KObj(r.state.coreSet()),
			"cause", cause,
		)
	case admissionEvacuate:
		err := s.startEvacuation(r, instance, candidate)
		if err != nil {
			return subResult{err: emperror.WrapWithDetails(err,
				"failed to start node evacuation",
				"pod", klog.KObj(candidate),
				"statefulSet", klog.KObj(r.state.coreSet()),
				"cause", cause,
			)}
		}
	}
	return subResult{}
}

// actOnCoreAdmission performs the side effects implied by a coreAdmission.
// Currently this only handles coreAdmitEvacuate by calling the evacuation API.
func (s *syncCoreSet) startEvacuation(
	r *reconcileRound,
	instance *crdv2.EMQX,
	pod *corev1.Pod,
) error {
	nodeInfo := instance.Status.FindNodeByPodName(pod.Name)
	if nodeInfo == nil {
		return emperror.New("no corresponding node in cluster status")
	}
	nodeName := nodeInfo.Name
	strategy := instance.Spec.UpdateStrategy.EvacuationStrategy
	migrateTo := migrationTargetNodes(r, instance)
	migrateTo = slices.DeleteFunc(
		migrateTo,
		func(e string) bool { return e == nodeInfo.Name },
	)
	if len(migrateTo) == 0 {
		s.EventRecorder.Event(
			instance,
			corev1.EventTypeWarning,
			"NodeEvacuation",
			fmt.Sprintf("Node %s evacuation skipped: no nodes to migrate to", nodeName),
		)
		return nil
	}
	err := api.StartEvacuation(r.oldestCoreRequester(), strategy, migrateTo, nodeName)
	if err != nil {
		return err
	}
	s.EventRecorder.Event(
		instance,
		corev1.EventTypeNormal,
		"NodeEvacuation",
		fmt.Sprintf("Node %s evacuation started", nodeName),
	)
	return nil
}

// migrationTargetNodes returns the list of EMQX nodes to migrate workloads to.
// For cores, targets are pods on the current (update) revision. For replicants,
// targets are pods in the update ReplicaSet.
func migrationTargetNodes(r *reconcileRound, instance *crdv2.EMQX) []string {
	targets := []string{}
	if instance.Spec.HasReplicants() {
		for _, node := range instance.Status.ReplicantNodes {
			pod := r.state.podWithName(node.PodName)
			if pod != nil && r.state.partOfUpdateReplicantSet(pod, instance) {
				targets = append(targets, node.Name)
			}
		}
	} else {
		for _, node := range instance.Status.CoreNodes {
			pod := r.state.podWithName(node.PodName)
			if pod != nil && r.state.partOfCoreSet(pod) {
				targets = append(targets, node.Name)
			}
		}
	}
	return targets
}
