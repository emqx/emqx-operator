package controller

import (
	"testing"
	"time"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCorePodRetirementReadyOverrides(t *testing.T) {
	now := time.Now()

	for _, tc := range []struct {
		name          string
		labels        map[string]string
		deletedAt     time.Time
		apiStatus     metav1.ConditionStatus
		apiTransition time.Time
		coreNodes     []crd.EMQXNode
		ready         bool
		reason        string
	}{
		{
			name:      "force label",
			labels:    map[string]string{crd.LabelForceRetirement: "true"},
			deletedAt: now,
			ready:     true,
			reason:    "retirement forced",
		},
		{
			name:      "fallback timeout reached",
			deletedAt: now.Add(-corePodForcedRetirementTimeout[condFallback] - time.Second),
			ready:     true,
			reason:    "timeout exceeded",
		},
		{
			name:          "API unavailable timeout reached",
			deletedAt:     now.Add(-2 * corePodForcedRetirementTimeout[condEMQXAPIUnavailable]),
			apiStatus:     metav1.ConditionFalse,
			apiTransition: now.Add(-2 * corePodForcedRetirementTimeout[condEMQXAPIUnavailable]),
			ready:         true,
			reason:        "API unavailability timeout exceeded",
		},
		{
			name:          "API unavailability predates deletion",
			deletedAt:     now.Add(-corePodForcedRetirementTimeout[condEMQXAPIUnavailable] / 2),
			apiStatus:     metav1.ConditionFalse,
			apiTransition: now.Add(-2 * corePodForcedRetirementTimeout[condEMQXAPIUnavailable]),
			reason:        "cluster membership state is unknown",
		},
		{
			name:          "deletion predates API unavailability",
			deletedAt:     now.Add(-2 * corePodForcedRetirementTimeout[condEMQXAPIUnavailable]),
			apiStatus:     metav1.ConditionFalse,
			apiTransition: now.Add(-corePodForcedRetirementTimeout[condEMQXAPIUnavailable] / 2),
			reason:        "cluster membership state is unknown",
		},
		{
			name:          "API available",
			deletedAt:     now.Add(-2 * corePodForcedRetirementTimeout[condEMQXAPIUnavailable]),
			apiStatus:     metav1.ConditionTrue,
			apiTransition: now.Add(-2 * corePodForcedRetirementTimeout[condEMQXAPIUnavailable]),
			coreNodes:     []crd.EMQXNode{{Name: "emqx@emqx", PodName: "emqx-1", Status: "running"}},
			reason:        "DS cluster state is not loaded",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			instance := &crd.EMQX{}
			instance.Status.CoreNodes = tc.coreNodes
			if tc.apiStatus != "" {
				instance.Status.Conditions = []metav1.Condition{{
					Type:               crd.EMQXAPIAvailable,
					Status:             tc.apiStatus,
					LastTransitionTime: metav1.NewTime(tc.apiTransition),
				}}
			}
			r := &retireCorePods{}
			pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
				Name:              "emqx-0",
				Labels:            tc.labels,
				DeletionTimestamp: &metav1.Time{Time: tc.deletedAt},
			}}
			ready, reason := r.corePodRetirementReady(&reconcileRound{}, instance, pod)
			assert.Equal(t, tc.ready, ready)
			assert.Contains(t, reason, tc.reason)
		})
	}
}
