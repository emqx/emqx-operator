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

func (s *syncReplicantSets) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	updateRs := r.state.updateReplicantSet(instance)
	currentRs := r.state.currentReplicantSet(instance)
	if updateRs == nil || currentRs == nil {
		return subResult{}
	}
	if updateRs.UID != currentRs.UID {
		return s.migrateSet(r, instance, updateRs)
	}
	return subResult{}
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
	admissions := s.topmostReplicantAdmissions(r, instance)
	for pod, admission := range admissions {
		err = s.onReplicantAdmission(r, instance, pod, admission)
		if err != nil {
			return reconcileError(err)
		}
	}

	// Phase 3:
	// Scale down affected outdated replicant sets.
	err = s.scaleDownReplicantSets(r, instance)
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
func (s *syncReplicantSets) onReplicantAdmission(
	r *reconcileRound,
	instance *crd.EMQX,
	pod *corev1.Pod,
	admission replicantAdmission,
) error {
	switch admission.Action {
	case admissionRemove:
		r.log.V(1).Info("scheduling replicant pod removal",
			"reason", admission.Reason,
			"pod", klog.KObj(pod),
		)
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}
		pod.Annotations[corev1.PodDeletionCost] = "-99999"
		if err := s.Client.Update(r.ctx, pod); err != nil {
			return emperror.Wrap(err, "failed to annotate replicant pod")
		}
	case admissionWait:
		r.log.V(1).Info("removal of replicant pod postponed",
			"reason", admission.Reason,
			"pod", klog.KObj(pod),
		)
	case admissionEvacuate:
		err := s.startEvacuation(r, instance, pod)
		if err != nil {
			return emperror.WrapWithDetails(err,
				"failed to start node evacuation",
				"pod", klog.KObj(pod),
			)
		}
	}
	return nil
}

// scaleDownReplicantSets subtracts one replica per pod scheduled for removal from that
// pod's owning ReplicaSet.
func (s *syncReplicantSets) scaleDownReplicantSets(r *reconcileRound, instance *crd.EMQX) error {
	for _, rs := range r.state.outdatedReplicantSets(instance) {
		numReplicas := int32(0)
		specReplicas := ptr.Deref(rs.Spec.Replicas, 0)
		if specReplicas == 0 {
			continue
		}
		for _, pod := range r.state.podsManagedBy(rs) {
			if pod.DeletionTimestamp != nil {
				continue
			}
			if _, ok := pod.Annotations[corev1.PodDeletionCost]; ok {
				continue
			}
			numReplicas += 1
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

// topmostReplicantAdmissions returns up to maxUnavailable admissions for the oldest eligible
// outdated replicant pods (evacuate and/or remove).
func (s *syncReplicantSets) topmostReplicantAdmissions(
	r *reconcileRound,
	instance *crd.EMQX,
) map[*corev1.Pod]replicantAdmission {
	batch := map[*corev1.Pod]replicantAdmission{}

	specReplicas := instance.Spec.NumReplicantReplicas()
	maxUnavailable := instance.Spec.NumMaxUnavailableReplicantReplicas()
	extraAvailable := r.state.numAvailableReplicants() - specReplicas
	numAllowedUnavailable := int(maxUnavailable + extraAvailable)

	// No budget left, wait for better times.
	if numAllowedUnavailable <= 0 {
		return batch
	}

	outdatedPods := r.state.outdatedReplicantPods(instance)
	for _, pod := range outdatedPods {
		// Batch contains enough admissions to fit into maxUnavailable allowance.
		if len(batch) >= numAllowedUnavailable {
			break
		}
		batch[pod] = checkReplicantPodRemoval(instance, pod)
	}

	return batch
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

	// if scaleDownNode == nil {
	// 	return replicantAdmission{}, emperror.Errorf("node is missing for pod %s", scaleDownPod.Name)
	// }

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
