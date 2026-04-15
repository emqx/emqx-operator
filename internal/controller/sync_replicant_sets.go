package controller

import (
	"fmt"
	"slices"

	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v3alpha1"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	"github.com/emqx/emqx-operator/internal/emqx/api"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
)

type syncReplicantSets struct {
	*EMQXReconciler
}

// replicantAdmission mirrors coreAdmission: a pure decision plus the candidate pod when relevant.
type replicantAdmission struct {
	Action admissionAction
	Reason string
}

// replicantPodAdmission pairs a pod with its admission decision, preserving iteration order.
type replicantPodAdmission struct {
	Admission replicantAdmission
	Pod       *corev1.Pod
}

func (s *syncReplicantSets) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	updateRs := r.state.updateReplicantSet(instance)
	currentRs := r.state.currentReplicantSet(instance)
	if updateRs == nil || currentRs == nil {
		return subResult{}
	}
	if updateRs.UID != currentRs.UID {
		r.log.V(1).Info("rolling replicantSet update",
			"replicaSet", klog.KObj(updateRs),
			"outdatedPods", len(r.state.outdatedReplicantPods(instance)),
		)
		return s.migrateSet(r, instance, updateRs)
	}

	// Steady state: handle scale-up and scale-down.
	desiredReplicas := instance.Spec.NumReplicantReplicas()
	currentReplicas := ptr.Deref(updateRs.Spec.Replicas, 0)

	if currentReplicas < desiredReplicas {
		r.log.V(1).Info("scaling up replicantSet",
			"replicaSet", klog.KObj(updateRs),
			"from", currentReplicas,
			"to", desiredReplicas,
		)
		return s.scaleUp(r, updateRs, desiredReplicas)
	}

	if currentReplicas > desiredReplicas {
		r.log.V(1).Info("scaling down replicantSet",
			"replicaSet", klog.KObj(updateRs),
			"from", currentReplicas,
			"to", desiredReplicas,
		)
		return s.scaleDown(r, instance, updateRs, currentReplicas, desiredReplicas)
	}

	// Steady state: clean up stale artifacts.
	err := s.ensureConsistency(r, instance, updateRs)
	if err != nil {
		return reconcileError(emperror.Wrap(err, "failed to restore replicant consistency"))
	}

	return subResult{}
}

// scaleUp sets the update ReplicaSet replica count to the desired value.
func (s *syncReplicantSets) scaleUp(
	r *reconcileRound,
	rs *appsv1.ReplicaSet,
	desiredReplicas int32,
) subResult {
	rs.Spec.Replicas = ptr.To(desiredReplicas)
	err := s.Client.Update(r.ctx, rs)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to scale up replicantSet")}
	}
	return subResult{}
}

// scaleDown removes up to maxUnavailable excess replicant pods per reconcile iteration
// with evacuation gating, then decrements the ReplicaSet replica count.
func (s *syncReplicantSets) scaleDown(
	r *reconcileRound,
	instance *crd.EMQX,
	updateRs *appsv1.ReplicaSet,
	currentReplicas int32,
	desiredReplicas int32,
) subResult {
	pods := r.state.podsManagedBy(updateRs)
	sortByName(pods)

	// Phase 1: admit up to maxUnavailable candidates for removal.
	// At least 1 removal should be allowed if MaxUnavailable is 0.
	excessReplicas := max(0, currentReplicas-desiredReplicas)
	maxUnavailable := max(1, instance.Spec.NumMaxUnavailableReplicantReplicas())
	excessUnavailable := max(0, desiredReplicas-updateRs.Status.AvailableReplicas)
	budget := max(0, min(maxUnavailable-excessUnavailable, excessReplicas))
	admissions := s.evaluateReplicantAdmissions(instance, pods, budget)
	for _, pa := range admissions {
		err := s.onReplicantAdmission(r, instance, pa.Pod, pa.Admission)
		if err != nil {
			return reconcileError(err)
		}
	}

	// Phase 2: set replicas to the count of non-removed pods, but no less than desired number.
	activeReplicas := int32(0)
	for _, pod := range pods {
		if s.podIsActiveReplicant(pod) {
			activeReplicas += 1
		}
	}
	newReplicas := max(activeReplicas, instance.Spec.NumReplicantReplicas())
	if newReplicas < currentReplicas {
		updateRs.Spec.Replicas = ptr.To(newReplicas)
		if err := s.Client.Update(r.ctx, updateRs); err != nil {
			return reconcileError(emperror.Wrap(err, "failed to scale down replicantSet"))
		}
	}

	return subResult{}
}

// ensureConsistency removes stale scale-down artifacts from "update" replicant set pods
// when no scale-down is active. This handles the case where a scale-down was interrupted
// by the user scaling back up: pods may still carry AnnotationScalingDown and PodDeletionCost
// annotations, and replicant evacuations may still be running.
func (s *syncReplicantSets) ensureConsistency(
	r *reconcileRound,
	instance *crd.EMQX,
	updateRs *appsv1.ReplicaSet,
) error {
	for _, pod := range r.state.podsManagedBy(updateRs) {
		// 1. Check if pod has stale scale-down annotations.
		dirty := s.removeStaleReplicantAnnotations(pod)
		if !dirty {
			continue
		}

		// 2. Stop any ongoing node evacuation.
		stopped, err := s.stopStaleReplicantEvacuation(r, instance, pod)
		if err != nil {
			return err
		}
		if stopped {
			r.log.V(1).Info("stopped stale replicant node evacuation",
				"replicaSet", klog.KObj(updateRs),
				"pod", klog.KObj(pod),
			)
		}

		// 3. Remove stale annotations.
		err = s.Client.Update(r.ctx, pod)
		if err == nil {
			r.log.V(1).Info("removed stale replicant pod annotations",
				"replicaSet", klog.KObj(updateRs),
				"pod", klog.KObj(pod),
			)
		} else {
			return emperror.Wrap(err, "failed to remove stale replicant pod annotations")
		}
	}
	return nil
}

// removeStaleReplicantAnnotations strips AnnotationScalingDown and PodDeletionCost from
// pod that was marked during a now-cancelled scale-down.
func (s *syncReplicantSets) removeStaleReplicantAnnotations(pod *corev1.Pod) bool {
	dirty := false
	if _, ok := pod.Annotations[crd.AnnotationScalingDown]; ok {
		delete(pod.Annotations, crd.AnnotationScalingDown)
		dirty = true
	}
	if _, ok := pod.Annotations[corev1.PodDeletionCost]; ok {
		delete(pod.Annotations, corev1.PodDeletionCost)
		dirty = true
	}
	return dirty
}

// stopStaleReplicantEvacuation stops evacuations on replicant node belonging
// to the specified pod.
func (s *syncReplicantSets) stopStaleReplicantEvacuation(
	r *reconcileRound,
	instance *crd.EMQX,
	pod *corev1.Pod,
) (bool, error) {
	nodeInfo := instance.Status.FindNodeByPodName(pod.Name, roleReplicant)
	if nodeInfo == nil {
		return false, emperror.Errorf("missing replicant %s node information", pod.Name)
	}
	if evacuation := instance.Status.FindNodeEvacuation(nodeInfo.Name); evacuation != nil {
		err := api.StopEvacuation(r.oldestCoreRequester(), nodeInfo.Name)
		if err == nil {
			s.EventRecorder.Event(
				instance,
				corev1.EventTypeNormal,
				"NodeEvacuation",
				fmt.Sprintf("Node %s evacuation stopped", nodeInfo.Name),
			)
			return true, nil
		} else {
			return true, err
		}
	}
	return false, nil
}

// migrateSet drives a replicant template migration: surge capacity on the new ReplicaSet,
// then for the topmost drain batch across all outdated pods, evacuations and safe removals.
func (s *syncReplicantSets) migrateSet(
	r *reconcileRound,
	instance *crd.EMQX,
	updateRs *appsv1.ReplicaSet,
) subResult {
	// Phase 1:
	// Scale the update ReplicaSet toward available maxSurge budget.
	err := s.surgeScaleSet(r, instance, updateRs)
	if err != nil {
		return reconcileError(emperror.Wrap(err, "failed to surge scale up replicantSet"))
	}

	// Phase 2:
	// Start migrating outdated replicants up to maxUnavailable allowance.
	admissions := s.outdatedReplicantAdmissions(r, instance)
	for _, pa := range admissions {
		err = s.onReplicantAdmission(r, instance, pa.Pod, pa.Admission)
		if err != nil {
			return reconcileError(err)
		}
	}

	// Phase 3:
	// Scale down affected outdated replicant sets.
	err = s.scaleDownOutdatedReplicantSets(r, instance)
	if err != nil {
		return reconcileError(err)
	}

	return subResult{}
}

func (s *syncReplicantSets) surgeScaleSet(
	r *reconcileRound,
	instance *crd.EMQX,
	updateRs *appsv1.ReplicaSet,
) error {
	specReplicas := instance.Spec.NumReplicantReplicas()
	maxReplicas := specReplicas + instance.Spec.NumMaxSurgeReplicantReplicas()
	numReplicas := ptr.Deref(updateRs.Spec.Replicas, 0)
	numOutdatedReplicas := int32(0)
	outdatedSets := r.state.outdatedReplicantSets(instance)
	for _, rs := range outdatedSets {
		numOutdatedReplicas += ptr.Deref(rs.Spec.Replicas, 0)
	}
	numAllowedReplicas := min(maxReplicas-numOutdatedReplicas, specReplicas)
	if numAllowedReplicas > numReplicas {
		updateRs.Spec.Replicas = ptr.To(numAllowedReplicas)
		err := s.Client.Update(r.ctx, updateRs)
		if err != nil {
			return err
		}
	}
	return nil
}

// onReplicantAdmission performs the side effects implied by a replicantAdmission.
// Every admitted pod is marked with AnnotationScalingDown so that on subsequent
// reconcile iterations it bypasses the maxUnavailable budget.
func (s *syncReplicantSets) onReplicantAdmission(
	r *reconcileRound,
	instance *crd.EMQX,
	pod *corev1.Pod,
	admission replicantAdmission,
) error {
	annotationsDirty := util.AttachPodAnnotation(pod, crd.AnnotationScalingDown, "true")
	switch admission.Action {
	case admissionRemove:
		r.log.V(1).Info("scheduling replicant pod removal",
			"reason", admission.Reason,
			"pod", klog.KObj(pod),
		)
		annotationsDirty = util.AttachPodAnnotation(pod, corev1.PodDeletionCost, "-99999") || annotationsDirty
	case admissionWait:
		r.log.V(1).Info("removal of replicant pod postponed",
			"reason", admission.Reason,
			"pod", klog.KObj(pod),
		)
	case admissionEvacuate:
		r.log.V(1).Info("starting replicant pod evacuation",
			"reason", admission.Reason,
			"pod", klog.KObj(pod),
		)
	}
	// 1. Commit the pod annotations.
	err := s.updatePodAnnotations(r, pod, annotationsDirty)
	if err != nil {
		return err
	}
	// 2. Run any side effects.
	switch admission.Action {
	case admissionEvacuate:
		err := s.startEvacuation(r, instance, pod)
		if err != nil {
			return emperror.WrapWithDetails(err,
				"failed to start node evacuation",
				"pod", klog.KObj(pod),
			)
		}
	default:
	}
	return nil
}

func (s *syncReplicantSets) updatePodAnnotations(r *reconcileRound, pod *corev1.Pod, dirty bool) error {
	if dirty {
		err := s.Client.Update(r.ctx, pod)
		if err != nil {
			return emperror.Wrap(err, "failed to annotate replicant pod")
		}
	}
	return nil
}

// scaleDownOutdatedReplicantSets subtracts one replica per pod scheduled for removal from that
// pod's owning ReplicaSet.
func (s *syncReplicantSets) scaleDownOutdatedReplicantSets(r *reconcileRound, instance *crd.EMQX) error {
	for _, rs := range r.state.outdatedReplicantSets(instance) {
		numReplicas := int32(0)
		specReplicas := ptr.Deref(rs.Spec.Replicas, 0)
		if specReplicas == 0 {
			continue
		}
		for _, pod := range r.state.podsManagedBy(rs) {
			if s.podIsActiveReplicant(pod) {
				numReplicas += 1
			}
		}
		if numReplicas < specReplicas {
			rs.Spec.Replicas = ptr.To(numReplicas)
			err := s.Client.Update(r.ctx, rs)
			if err != nil {
				return emperror.WrapWithDetails(
					err,
					"scaling down outdated replicantSet failed",
					"replicantSet", rs.Name,
				)
			}
		}
	}
	return nil
}

// outdatedReplicantAdmissions returns admissions, up to a budget allowed by maxUnavailable, for
// the topmost outdated pods.
func (s *syncReplicantSets) outdatedReplicantAdmissions(
	r *reconcileRound,
	instance *crd.EMQX,
) []replicantPodAdmission {
	specReplicas := instance.Spec.NumReplicantReplicas()
	maxUnavailable := instance.Spec.NumMaxUnavailableReplicantReplicas()
	extraAvailable := r.state.numAvailableReplicants() - specReplicas
	budget := maxUnavailable + extraAvailable
	outdatedPods := r.state.outdatedReplicantPods(instance)
	return s.evaluateReplicantAdmissions(instance, outdatedPods, budget)
}

// evaluateReplicantAdmissions returns admissions for the given candidate pods.
// Pods already annotated with scaling-down bypass the budget: they were committed
// to in a previous reconcile iteration. The budget only limits how many *unannotated*
// pods can be admitted per iteration.
// Candidates are evaluated in order, the caller controls which pods to consider:
// * outdated pods for migration,
// * "update" set's replicant pods for scale-down, etc.
func (s *syncReplicantSets) evaluateReplicantAdmissions(
	instance *crd.EMQX,
	candidates []*corev1.Pod,
	budget int32,
) []replicantPodAdmission {
	batch := []replicantPodAdmission{}
	budgetUsed := int32(0)
	for _, pod := range candidates {
		// Evaluate admission:
		admission := replicantPodAdmission{
			Pod:       pod,
			Admission: checkReplicantPodRemoval(instance, pod),
		}
		// Consume the budget:
		budgetUsed += 1
		if _, ok := pod.Annotations[crd.AnnotationScalingDown]; ok {
			// Pod was already committed to in a previous reconcile.
			// Including it so the controller can progress it toward removal.
			batch = append(batch, admission)
			continue
		}
		// Otherwise, see if we have budget left:
		if budgetUsed > budget {
			continue
		}
		// If we do, include the admission:
		batch = append(batch, admission)
	}
	return batch
}

// podIsActiveReplicant returns `false` if a pod is in the process of or going to be deleted,
// e.g. assigned a pod deletion cost.
func (*syncReplicantSets) podIsActiveReplicant(pod *corev1.Pod) bool {
	if pod.DeletionTimestamp != nil {
		return false
	}
	if _, ok := pod.Annotations[corev1.PodDeletionCost]; ok {
		return false
	}
	return true
}

// checkReplicantPodRemoval decides whether an outdated replicant pod may be removed or needs evacuation.
func checkReplicantPodRemoval(instance *crd.EMQX, pod *corev1.Pod) replicantAdmission {
	status := &instance.Status

	if pod.DeletionTimestamp != nil {
		return replicantAdmission{
			Action: admissionWait,
			Reason: fmt.Sprintf("pod %s deletion in progress", pod.Name),
		}
	}

	if _, ok := pod.Annotations[corev1.PodDeletionCost]; ok {
		return replicantAdmission{
			Action: admissionRemove,
			Reason: "pod already marked for deletion",
		}
	}

	// Disallow permanently removing the pod that is still a DS replication site.
	dsCondition := util.FindPodCondition(pod, crd.DSReplicationSite)
	if dsCondition != nil && dsCondition.Status != corev1.ConditionFalse {
		return replicantAdmission{
			Action: admissionWait,
			Reason: fmt.Sprintf("pod %s is still a DS replication site", pod.Name),
		}
	}

	nodeInfo := status.FindNodeByPodName(pod.Name, "replicant")
	if nodeInfo == nil {
		return replicantAdmission{Action: admissionRemove, Reason: "node is out of cluster"}
	}

	if nodeInfo.Status == api.NodeStatusStopped {
		return replicantAdmission{Action: admissionRemove, Reason: "node is already stopped"}
	}

	evacuation := status.FindNodeEvacuation(nodeInfo.Name)
	if evacuation != nil && evacuation.State != api.EvacuationStateProhibiting {
		return replicantAdmission{
			Action: admissionWait,
			Reason: fmt.Sprintf("node %s evacuation in progress", nodeInfo.Name),
		}
	}

	if nodeInfo.Sessions > 0 {
		if instance.Spec.NumReplicantReplicas() == 1 && instance.Spec.NumMaxSurgeReplicantReplicas() == 0 {
			return replicantAdmission{
				Action: admissionRemove,
				Reason: fmt.Sprintf("node %s has active sessions nowhere to evacuate", nodeInfo.Name),
			}
		}
		return replicantAdmission{
			Action: admissionEvacuate,
			Reason: fmt.Sprintf("node %s has active sessions", nodeInfo.Name),
		}
	}

	return replicantAdmission{Action: admissionRemove, Reason: "node is safe to stop"}
}

// startReplicantEvacuation calls the EMQX evacuation API for a replicant pod (side effect only).
func (s *syncReplicantSets) startEvacuation(r *reconcileRound, instance *crd.EMQX, pod *corev1.Pod) error {
	nodeInfo := instance.Status.FindNodeByPodName(pod.Name, "replicant")
	if nodeInfo == nil {
		return emperror.New("no corresponding replicant node in cluster status")
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
