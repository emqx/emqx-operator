package controller

import (
	"fmt"
	"time"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type forcedRetirementCondition int

const (
	condEMQXAPIUnavailable forcedRetirementCondition = iota
	condFallback
)

// Kubernetes sets DeletionTimestamp to the scheduled end of graceful termination,
// so retirement timeouts measured from it include the pod's grace period.
var corePodForcedRetirementTimeout = map[forcedRetirementCondition]time.Duration{
	// causeEMQXAPIUnavailable applies after both pod deletion and continuous EMQX API unavailability have begun.
	condEMQXAPIUnavailable: time.Minute,
	// causeFallback defines the unconditional hard retirement deadline.
	condFallback: time.Minute * 10,
}

// retireCorePods releases scale-down pod finalizers after other reconcilers
// have removed the old node identity from EMQX membership and DS metadata.
type retireCorePods struct {
	*EMQXReconciler
}

func (s *retireCorePods) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	coreSet := r.state.coreSet()
	if coreSet == nil {
		return subResult{}
	}

	for _, pod := range r.state.podsManagedBy(coreSet) {
		// No finalizer attached, skip:
		if !controllerutil.ContainsFinalizer(pod, crd.FinalizerScaleDownRetirement) {
			continue
		}

		// Has finalizer but no deletion was requested, reconcile:
		if pod.DeletionTimestamp == nil {
			// If this replica is in the range of desired number of replicas, drop the finalizer:
			ordinal := util.PodOrdinal(pod.Name)
			if ordinal >= 0 && ordinal < int(instance.Spec.NumCoreReplicas()) {
				err := removeScaleDownRetirementFinalizer(r.ctx, s.Client, pod)
				if err != nil {
					return subResult{err: err}
				}
			}
			continue
		}

		ready, reason := s.corePodRetirementReady(r, instance, pod)
		if !ready {
			r.log.V(1).Info("core pod retirement pending",
				"reason", reason,
				"pod", klog.KObj(pod),
			)
			continue
		}

		err := removeScaleDownRetirementFinalizer(r.ctx, s.Client, pod)
		if err == nil {
			logArgs := []any{"pod", klog.KObj(pod)}
			if reason != "" {
				logArgs = append(logArgs, "reason", reason)
				s.EventRecorder.Eventf(
					instance,
					corev1.EventTypeWarning,
					"CorePodForceRetired",
					"Core pod %s retirement guardrails bypassed: %s",
					pod.Name,
					reason,
				)
			} else {
				s.EventRecorder.Eventf(
					instance,
					corev1.EventTypeNormal,
					"CorePodRetired",
					"Core pod %s retired",
					pod.Name,
				)
			}
			r.log.V(1).Info("core pod retired", logArgs...)
		} else {
			return subResult{err: err}
		}
	}

	return subResult{}
}

func (*retireCorePods) corePodRetirementReady(r *reconcileRound, instance *crd.EMQX, pod *corev1.Pod) (bool, string) {
	now := time.Now()

	if pod.Labels[crd.LabelForceRetirement] == "true" {
		return true, "retirement forced"
	}

	fallbackTimeout := corePodForcedRetirementTimeout[condFallback]
	fallbackDeadline := pod.DeletionTimestamp.Add(fallbackTimeout)
	if now.After(fallbackDeadline) {
		deadlineString := fallbackDeadline.UTC().Format(time.RFC3339)
		return true, fmt.Sprintf("fallback timeout exceeded at %s", deadlineString)
	}

	_, apiCondition := instance.Status.GetCondition(crd.EMQXAPIAvailable)
	if apiCondition != nil && apiCondition.Status == metav1.ConditionFalse {
		unavailableSince := apiCondition.LastTransitionTime.Time
		if pod.DeletionTimestamp.After(unavailableSince) {
			unavailableSince = pod.DeletionTimestamp.Time
		}
		apiTimeout := corePodForcedRetirementTimeout[condEMQXAPIUnavailable]
		apiDeadline := unavailableSince.Add(apiTimeout)
		if now.After(apiDeadline) {
			deadlineString := apiDeadline.UTC().Format(time.RFC3339)
			return true, fmt.Sprintf("EMQX API unavailability timeout exceeded at %s", deadlineString)
		}
	}

	if instance.Status.CoreNodes == nil {
		return false, "cluster membership state is unknown"
	}

	node := instance.Status.FindNodeByPodName(pod.Name, crd.RoleCore)
	if node != nil {
		return false, fmt.Sprintf("node %s is still present in cluster status", node.Name)
	}

	if r.dsCluster == nil {
		return false, "DS cluster state is not loaded"
	}

	site := r.dsCluster.FindSite(constructNodeName(pod.Name, instance))
	if site != nil {
		return false, fmt.Sprintf("DS site %s for node %s is still present", site.ID, site.Node)
	}

	return true, ""
}
