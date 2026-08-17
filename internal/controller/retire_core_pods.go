package controller

import (
	"fmt"
	"time"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type forcedRetirementCondition int

const (
	condEMQXAPIUnavailable forcedRetirementCondition = iota
	condFallback
)

var corePodForcedRetirementTimeout = map[forcedRetirementCondition]time.Duration{
	// condEMQXAPIUnavailable applies after both ordinal retirement and continuous EMQX API unavailability have begun.
	condEMQXAPIUnavailable: time.Minute,
	// condFallback defines the unconditional hard retirement deadline.
	// The hard fallback timeout bypasses membership, DS, and PVC checks,
	// but still requires the pod to be absent.
	condFallback: time.Minute * 10,
}

// retireCorePods normally advances the reusable ordinal watermark after the
// retiring core has left EMQX membership and DS metadata, and its pod and PVCs
// are gone.
type retireCorePods struct {
	*EMQXReconciler
}

func (s *retireCorePods) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	coreSet := r.state.coreSet()
	if coreSet == nil {
		return subResult{}
	}

	currentReplicas := util.NumReplicas(coreSet)
	if r.coreRetirement.watermark == currentReplicas {
		return subResult{}
	}

	ordinal := int(r.coreRetirement.watermark - 1)
	podName := fmt.Sprintf("%s-%d", coreSet.Name, ordinal)
	podObject := client.ObjectKey{Namespace: coreSet.Namespace, Name: podName}

	if pod := r.state.podWithName(podName); pod != nil {
		return reconcilePostpone()
	}

	decision := s.corePodRetirementReady(r, instance, podName)
	if decision.err != nil {
		return reconcileError(decision.err)
	}
	if !decision.ready {
		r.log.V(1).Info("core pod retirement pending", "reason", decision.reason, "pod", podObject)
		return reconcilePostpone()
	}

	r.coreRetirement.watermark -= 1
	util.AttachAnnotations(coreSet, r.coreRetirement.annotations())
	if err := s.Client.Update(r.ctx, coreSet); err != nil {
		return reconcileError(fmt.Errorf("failed to advance reusable core pod ordinal watermark: %w", err))
	}

	logArgs := []any{"pod", podObject}
	if decision.reason != "" {
		logArgs = append(logArgs, "reason", decision.reason)
		s.EventRecorder.Eventf(
			instance,
			corev1.EventTypeWarning,
			"CorePodForceRetired",
			"Core pod %s retirement guardrails bypassed: %s",
			podName,
			decision.reason,
		)
	} else {
		s.EventRecorder.Eventf(
			instance,
			corev1.EventTypeNormal,
			"CorePodRetired",
			"Core pod %s retired and its ordinal is reusable",
			podName,
		)
	}
	r.log.V(1).Info("core pod ordinal made reusable", logArgs...)
	return subResult{}
}

type retirementDecision struct {
	ready  bool
	reason string
	err    error
}

func retirementReady(reason string) retirementDecision {
	return retirementDecision{true, reason, nil}
}

func retirementPending(reason string) retirementDecision {
	return retirementDecision{false, reason, nil}
}

func (s *retireCorePods) corePodRetirementReady(
	r *reconcileRound,
	instance *crd.EMQX,
	podName string,
) retirementDecision {
	now := time.Now()
	updatedAt := r.coreRetirement.updatedAt

	fallbackTimeout := corePodForcedRetirementTimeout[condFallback]
	fallbackDeadline := updatedAt.Add(fallbackTimeout)
	if now.After(fallbackDeadline) {
		deadlineString := fallbackDeadline.UTC().Format(time.RFC3339)
		return retirementReady(fmt.Sprintf("fallback timeout exceeded at %s", deadlineString))
	}

	hasPVCs, err := s.corePodHasPVCs(r, podName)
	if err != nil {
		return retirementDecision{false, "", err}
	}
	if hasPVCs {
		return retirementPending("pod still has live PVCs")
	}

	apiCondition := instance.Status.GetCondition(crd.EMQXAPIAvailable)
	if apiCondition != nil && apiCondition.Status == metav1.ConditionFalse {
		unavailableSince := apiCondition.LastTransitionTime.Time
		if updatedAt.After(unavailableSince) {
			unavailableSince = updatedAt
		}
		apiTimeout := corePodForcedRetirementTimeout[condEMQXAPIUnavailable]
		apiDeadline := unavailableSince.Add(apiTimeout)
		if now.After(apiDeadline) {
			deadlineString := apiDeadline.UTC().Format(time.RFC3339)
			return retirementReady(fmt.Sprintf("EMQX API unavailability timeout exceeded at %s", deadlineString))
		}
	}

	if !instance.Status.HasClusterMembership() {
		return retirementPending("cluster membership state is unknown")
	}
	if node := instance.Status.FindNodeByPodName(podName, crd.RoleCore); node != nil {
		return retirementPending(fmt.Sprintf("node %s is still present in cluster status", node.Name))
	}
	if r.dsCluster == nil {
		return retirementPending("DS cluster state is not loaded")
	}
	if site := r.dsCluster.FindSite(constructNodeName(podName, instance)); site != nil {
		return retirementPending(fmt.Sprintf("DS site %s for node %s is still present", site.ID, site.Node))
	}

	return retirementReady("")
}

func (s *retireCorePods) corePodHasPVCs(r *reconcileRound, podName string) (bool, error) {
	coreSet := r.state.coreSet()
	for _, claim := range coreSet.Spec.VolumeClaimTemplates {
		pvc := &corev1.PersistentVolumeClaim{}
		key := client.ObjectKey{
			Name:      fmt.Sprintf("%s-%s", claim.Name, podName),
			Namespace: coreSet.Namespace,
		}
		err := s.Client.Get(r.ctx, key, pvc)
		switch {
		case err == nil:
			return true, nil
		case k8sErrors.IsNotFound(err):
			continue
		default:
			return true, fmt.Errorf("failed to get core pod PVC %s: %w", key.Name, err)
		}
	}
	return false, nil
}
