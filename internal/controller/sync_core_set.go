package controller

import (
	"context"
	"fmt"
	"slices"

	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v3beta1"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	"github.com/emqx/emqx-operator/internal/emqx/api"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	// Pod removal blocked; reason explains why.
	admissionWait admissionAction = iota
	// Pod may be removed right now.
	admissionRemove
	// Pod needs evacuation before it can be removed.
	admissionEvacuate
)

type admissionCause string

const (
	coreScaleDown     admissionCause = "scale down"
	coreRollingUpdate admissionCause = "rolling update"
)

type coreAdmission struct {
	Action admissionAction
	Reason string
	Cause  admissionCause
}

func (a coreAdmission) wait(reason string, args ...any) coreAdmission {
	return a.withActionReason(admissionWait, reason, args...)
}

func (a coreAdmission) remove(reason string, args ...any) coreAdmission {
	return a.withActionReason(admissionRemove, reason, args...)
}

func (a coreAdmission) evacuate(reason string, args ...any) coreAdmission {
	return a.withActionReason(admissionEvacuate, reason, args...)
}

func (a coreAdmission) withActionReason(action admissionAction, reason string, args ...any) coreAdmission {
	a.Action = action
	if len(args) > 0 {
		a.Reason = fmt.Sprintf(reason, args...)
	} else {
		a.Reason = reason
	}
	return a
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
			return s.onCoreAdmission(r, instance, candidate, coreAdmission{
				Action: admissionWait,
				Reason: "current replicantSet still migrating",
				Cause:  coreRollingUpdate,
			})
		}
	}

	admission := checkCorePodRemoval(r, instance, candidate, coreRollingUpdate)
	return s.onCoreAdmission(r, instance, candidate, admission)
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
	if candidate != nil {
		admission = checkCorePodRemoval(r, instance, candidate, coreScaleDown)
	} else {
		admission = coreAdmission{
			Action: admissionRemove,
			Reason: "already terminated",
			Cause:  coreScaleDown,
		}
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
		return s.onCoreAdmission(r, instance, candidate, admission)
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
		if node == nil || node.Role != crd.RoleCore || node.PodName == "" {
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
// When cause is coreScaleDown, the pod is being permanently removed: data would
// be lost so there are extra safety checks (e.g. DS replication site).
func checkCorePodRemoval(
	r *reconcileRound,
	instance *crd.EMQX,
	pod *corev1.Pod,
	cause admissionCause,
) coreAdmission {
	status := &instance.Status
	admission := coreAdmission{Cause: cause}

	// Disallow removing pod if other cores just recently became ready.
	numAvailableCores := int32(0)
	for _, p := range r.state.listPods(podsManagedBy{r.state.coreSet()}, podsAlive{}) {
		if p.GetUID() != pod.GetUID() && util.IsPodAvailable(p, instance.Spec.CoreTemplate.Spec.MinReadySeconds) {
			numAvailableCores++
		}
	}
	if numAvailableCores < instance.Spec.NumCoreReplicas()-1 {
		return admission.wait("cores are not available yet")
	}

	// If a pod is already being deleted, return it.
	if pod.DeletionTimestamp != nil {
		return admission.wait("pod %s deletion in progress", pod.Name)
	}

	// Disallow permanently removing the pod that is still a DS replication site.
	if cause == coreScaleDown {
		dsCondition := util.FindPodCondition(pod, crd.DSReplicationSite)
		if dsCondition != nil && dsCondition.Status != corev1.ConditionFalse {
			return admission.wait("pod %s is still a DS replication site", pod.Name)
		}
	}

	nodeInfo := status.FindNodeByPodName(pod.Name, crd.RoleCore)

	if nodeInfo == nil {
		return admission.remove("node is out of cluster")
	}

	if nodeInfo.Status == api.NodeStatusStopped {
		return admission.remove("node is already stopped")
	}

	evacuation := status.FindNodeEvacuation(nodeInfo.Name)
	if evacuation != nil && evacuation.State != api.EvacuationStateProhibiting {
		return admission.wait("node evacuation in progress")
	}

	if nodeInfo.Sessions > 0 && instance.Spec.IsEvacuationEnabled() {
		if instance.Spec.NumCoreReplicas() == 1 && !instance.Spec.HasReplicants() {
			return admission.remove("node %s has active sessions nowhere to evacuate", nodeInfo.Name)
		}
		return admission.evacuate("node %s has active sessions", nodeInfo.Name)
	}

	return admission.remove("node is safe to stop")
}

// onCoreAdmission performs the side effects implied by a coreAdmission.
func (s *syncCoreSet) onCoreAdmission(
	r *reconcileRound,
	instance *crd.EMQX,
	candidate *corev1.Pod,
	admission coreAdmission,
) subResult {
	switch admission.Action {
	case admissionRemove:
		r.log.V(1).Info("removing core pod",
			"reason", admission.Reason,
			"pod", klog.KObj(candidate),
			"statefulSet", klog.KObj(r.state.coreSet()),
			"cause", admission.Cause,
		)
		if admission.Cause == coreScaleDown {
			if err := attachScaleDownRetirementFinalizer(r.ctx, s.Client, candidate); err != nil {
				return reconcileError(err)
			}
		}
		if err := s.Client.Delete(r.ctx, candidate); err != nil {
			return reconcileError(emperror.Wrap(err, "failed to delete core pod"))
		}
	case admissionWait:
		r.log.V(1).Info("removal of core pod postponed",
			"reason", admission.Reason,
			"pod", klog.KObj(candidate),
			"statefulSet", klog.KObj(r.state.coreSet()),
			"cause", admission.Cause,
		)
	case admissionEvacuate:
		err := s.startEvacuation(r, instance, candidate)
		if err != nil {
			return subResult{err: emperror.WrapWithDetails(err,
				"failed to start node evacuation",
				"pod", klog.KObj(candidate),
				"statefulSet", klog.KObj(r.state.coreSet()),
				"cause", admission.Cause,
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
	nodeInfo := instance.Status.FindNodeByPodName(pod.Name, crd.RoleCore)
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
//   - pods in the "update" replicant set, if it has at least 1 node up and running,
//   - pods in any replicant set otherwise.
func migrationTargetNodes(r *reconcileRound, instance *crd.EMQX) []string {
	targets := []string{}
	fallback := []string{}
	if instance.Spec.HasReplicants() {
		updateReplicantSet := r.state.updateReplicantSet(instance)
		if updateReplicantSet == nil {
			return targets
		}
		for _, node := range instance.Status.ReplicantNodes {
			pod := r.state.podWithName(node.PodName)
			if pod == nil {
				continue
			}
			if util.IsPodManagedBy(pod, updateReplicantSet) {
				targets = append(targets, node.Name)
			}
			fallback = append(fallback, node.Name)
		}
		if len(targets) == 0 {
			return fallback
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

func attachScaleDownRetirementFinalizer(ctx context.Context, k8sClient client.Client, pod *corev1.Pod) error {
	if controllerutil.AddFinalizer(pod, crd.FinalizerScaleDownRetirement) {
		if err := k8sClient.Update(ctx, pod); err != nil {
			return emperror.Wrap(err, "failed to attach retirement finalizer")
		}
	}
	return nil
}

func removeScaleDownRetirementFinalizer(ctx context.Context, k8sClient client.Client, pod *corev1.Pod) error {
	if controllerutil.RemoveFinalizer(pod, crd.FinalizerScaleDownRetirement) {
		if err := k8sClient.Update(ctx, pod); err != nil {
			return emperror.Wrap(err, "failed to remove retirement finalizer")
		}
	}
	return nil
}

func isPodScaleDownRetiring(pod *corev1.Pod) bool {
	return pod.DeletionTimestamp != nil &&
		controllerutil.ContainsFinalizer(pod, crd.FinalizerScaleDownRetirement)
}
