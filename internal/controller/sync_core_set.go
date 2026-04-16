package controller

import (
	"fmt"
	"slices"

	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v3alpha1"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	"github.com/emqx/emqx-operator/internal/emqx/api"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

// Responsibilities:
// - Scaling the core set up according to specified number of replicas.
// - Scaling the core set down safely according to specified number of replicas.
//   - Safety: availability is maintained.
//     Loss of availability of a single, next-in-line replica is tolerated.
//   - Safety: connections and sessions are evacuated first.
//   - Safety: existing DS shard replicas prevent scaling until all replicas are migrated.
//     See `dsUpdateReplicaSets` reconciler.
//
// - Conducting safe pod-by-pod in-place rolling update of the core set.
//   - Safety: availability is maintained.
//   - Safety: connections and sessions are evacuated first.
//
// - Terminating evacuations on updated cores.
//
// NOTE
// As evacuation state is expected to survive pod recreation, evacuations need to be
// terminated manually. Currently, this reconciler can not tell the difference between
// evacuations started by itself and manually by the user, latter will be stopped on the
// next reconcile.
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

func (s *syncCoreSet) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	coreSet := r.state.coreSet()
	if coreSet == nil {
		return reconcilePostpone()
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
		return s.scaleDown(r, instance, currentReplicas)
	}

	// Stop evacuations of any updated pods.
	// This is done irrespective of whether Node Evacuation is enabled or not,
	// to avoid ending up in transient state if it was disabled mid-update.
	err := s.updateEvacuationState(r, instance)
	if err != nil {
		return reconcileError(emperror.Wrap(err, "failed to update evacuation state"))
	}

	// Handle rolling update: replace outdated pods one at a time, highest ordinal first.
	return s.rollingUpdate(r, instance)
}

// rollingUpdate detects outdated core pods and replaces them one at a time,
// starting from the highest ordinal. Each pod is evacuated before deletion.
func (s *syncCoreSet) rollingUpdate(r *reconcileRound, instance *crd.EMQX) subResult {
	// Sort outdated pods by ordinal ascending; pick highest ordinal last.
	outdated := listOutdatedPods(r)
	sortByOrdinal(outdated)

	if len(outdated) == 0 {
		return subResult{}
	}

	r.log.V(1).Info("rolling coreSet update",
		"statefulSet", klog.KObj(r.state.coreSet()),
		"outdatedPods", len(outdated),
	)

	candidate := outdated[len(outdated)-1]

	if instance.Spec.HasReplicants() &&
		instance.Status.ReplicantNodesStatus.CurrentRevision != instance.Status.ReplicantNodesStatus.UpdateRevision {
		// ReplicantSet is in the process of update.
		// Keep at least one old-version core alive so current-revision replicants can rejoin.
		if len(outdated) == 1 {
			admission := coreAdmission{Action: admissionWait, Reason: "current replicantSet still migrating"}
			return s.onCoreAdmission(r, instance, candidate, admission, "rollingUpdate")
		}
	}

	admission := checkCorePodRemoval(r, instance, candidate, false)
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
func (s *syncCoreSet) scaleDown(r *reconcileRound, instance *crd.EMQX, currentReplicas int32) subResult {
	coreSet := r.state.coreSet()

	// Candidate is the highest-ordinal pod, where ordinal = currentReplicas-1.
	candidateName := fmt.Sprintf("%s-%d", coreSet.Name, currentReplicas-1)
	candidate := r.state.podWithName(candidateName)

	var admission coreAdmission
	if candidate == nil {
		admission = coreAdmission{Action: admissionRemove, Reason: "already terminated"}
	} else {
		admission = checkCorePodRemoval(r, instance, candidate, true)
	}

	if admission.Action == admissionRemove {
		// Decrement StatefulSet replica count first so the StatefulSet controller
		// won't recreate the pod after we delete it.
		newReplicas := currentReplicas - 1
		coreSet.Spec.Replicas = &newReplicas
		err := s.Client.Update(r.ctx, coreSet)
		if err != nil {
			return subResult{err: emperror.Wrap(err, "failed to decrement coreSet replicas")}
		}
	}
	if candidate != nil {
		return s.onCoreAdmission(r, instance, candidate, admission, "scaleDown")
	}
	return subResult{}
}

// Stops evacuation on nodes that are no longer need to evacuate anything:
// nodes that belong to the most recent coreSet revision.
func (s *syncCoreSet) updateEvacuationState(r *reconcileRound, instance *crd.EMQX) error {
	updateRevision := r.state.coreSet().Status.UpdateRevision
	for _, evacuation := range instance.Status.NodeEvacuations {
		if evacuation.State != api.EvacuationStateProhibiting {
			continue
		}
		node := instance.Status.FindNode(evacuation.NodeName)
		if node == nil || node.Role != "core" || node.PodName == "" {
			continue
		}
		pod := r.state.podWithName(node.PodName)
		if pod != nil && r.state.partOfCoreSetRevision(pod, updateRevision) {
			err := api.StopEvacuation(r.requester.forPod(pod), node.Name)
			if err == nil {
				s.EventRecorder.Event(
					instance,
					corev1.EventTypeNormal,
					"NodeEvacuation",
					fmt.Sprintf("Node %s evacuation stopped", node.Name),
				)
			} else {
				return err
			}
		}
	}
	return nil
}

// listOutdatedPods returns core StatefulSet pods whose pod template is not yet the
// desired one: anything not labeled with Status.UpdateRevision.
//
// We intentionally do not key off CurrentRevision alone. Pods can remain labeled
// with a revision hash that is neither CurrentRevision nor UpdateRevision (e.g.
// stuck pod after ControllerRevision history moved on). Those are still outdated.
func listOutdatedPods(r *reconcileRound) []*corev1.Pod {
	var outdated []*corev1.Pod
	coreSet := r.state.coreSet()
	updateRevision := coreSet.Status.UpdateRevision
	if updateRevision == "" || updateRevision == coreSet.Status.CurrentRevision {
		return outdated
	}
	for _, pod := range r.state.podsManagedBy(coreSet) {
		if !r.state.partOfCoreSetRevision(pod, updateRevision) {
			outdated = append(outdated, pod)
		}
	}
	return outdated
}

// checkCorePodRemoval is a pure function that decides whether a core pod can
// be safely removed. It inspects instance status and pod state but performs no
// side effects.
//
// When isPermanent is true, the pod is being permanently removed:
// data would be lost so there are extra safety checks (e.g. DS replication site).
// When isPermanent is false (rolling update), the replacement pod inherits the
// data: blocking on DS site condition would stall the rollout indefinitely.
func checkCorePodRemoval(
	r *reconcileRound,
	instance *crd.EMQX,
	pod *corev1.Pod,
	isPermanent bool,
) coreAdmission {
	status := &instance.Status

	// Disallow removing pod if other cores just recently became ready.
	numAvailableCores := int32(0)
	for _, p := range r.state.listPods(podsManagedBy{r.state.coreSet()}, podsAlive{}) {
		if p.GetUID() != pod.GetUID() && util.IsPodAvailable(p, instance.Spec.CoreTemplate.Spec.MinReadySeconds) {
			numAvailableCores++
		}
	}
	if numAvailableCores < instance.Spec.NumCoreReplicas()-1 {
		return coreAdmission{Action: admissionWait, Reason: "cores are not available yet"}
	}

	// If a pod is already being deleted, return it.
	if pod.DeletionTimestamp != nil {
		return coreAdmission{Action: admissionWait, Reason: fmt.Sprintf("pod %s deletion in progress", pod.Name)}
	}

	// Disallow permanently removing the pod that is still a DS replication site.
	if isPermanent {
		dsCondition := util.FindPodCondition(pod, crd.DSReplicationSite)
		if dsCondition != nil && dsCondition.Status != corev1.ConditionFalse {
			return coreAdmission{Action: admissionWait, Reason: fmt.Sprintf("pod %s is still a DS replication site", pod.Name)}
		}
	}

	nodeInfo := status.FindNodeByPodName(pod.Name, roleCore)

	if nodeInfo == nil {
		return coreAdmission{Action: admissionRemove, Reason: "node is out of cluster"}
	}

	if nodeInfo.Status == api.NodeStatusStopped {
		return coreAdmission{Action: admissionRemove, Reason: "node is already stopped"}
	}

	evacuation := status.FindNodeEvacuation(nodeInfo.Name)
	if evacuation != nil && evacuation.State != api.EvacuationStateProhibiting {
		return coreAdmission{Action: admissionWait, Reason: "node evacuation is still in progress"}
	}

	if nodeInfo.Sessions > 0 && instance.Spec.IsEvacuationEnabled() {
		if instance.Spec.NumCoreReplicas() == 1 && !instance.Spec.HasReplicants() {
			return coreAdmission{
				Action: admissionRemove,
				Reason: fmt.Sprintf("node %s has active sessions nowhere to evacuate", nodeInfo.Name),
			}
		}
		return coreAdmission{
			Action: admissionEvacuate,
			Reason: fmt.Sprintf("node %s has active sessions", nodeInfo.Name),
		}
	}

	return coreAdmission{Action: admissionRemove, Reason: "node is safe to stop"}
}

// onCoreAdmission performs the side effects implied by a coreAdmission.
func (s *syncCoreSet) onCoreAdmission(
	r *reconcileRound,
	instance *crd.EMQX,
	candidate *corev1.Pod,
	admission coreAdmission,
	cause string,
) subResult {
	switch admission.Action {
	case admissionRemove:
		r.log.V(1).Info("removing core pod",
			"reason", admission.Reason,
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
	instance *crd.EMQX,
	pod *corev1.Pod,
) error {
	nodeInfo := instance.Status.FindNodeByPodName(pod.Name, "core")
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
	err := api.StartEvacuation(r.requester.forPod(pod), strategy, migrateTo, nodeName)
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
// * In core-only cluster, targets are pods of the core set.
// * In core-replicant cluster, targets are:
//   - pods in the "update" replicant set, if it has at least 1 ready replica,
//   - pods in any replicant set otherwise.
func migrationTargetNodes(r *reconcileRound, instance *crd.EMQX) []string {
	targets := []string{}
	if instance.Spec.HasReplicants() {
		updateReplicantSet := r.state.updateReplicantSet(instance)
		if updateReplicantSet == nil {
			return targets
		}
		updateReady := updateReplicantSet.Status.ReadyReplicas > 0
		for _, node := range instance.Status.ReplicantNodes {
			pod := r.state.podWithName(node.PodName)
			if pod == nil {
				continue
			}
			if updateReady && util.IsPodManagedBy(pod, updateReplicantSet) {
				targets = append(targets, node.Name)
			}
			if r.state.partOfReplicantSet(pod) {
				targets = append(targets, node.Name)
			}
		}
	} else {
		for _, node := range instance.Status.CoreNodes {
			pod := r.state.podWithName(node.PodName)
			if pod == nil {
				continue
			}
			if r.state.partOfCoreSet(pod) {
				targets = append(targets, node.Name)
			}
		}
	}
	return targets
}
