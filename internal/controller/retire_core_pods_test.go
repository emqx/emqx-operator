package controller

import (
	"testing"
	"time"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCoreOrdinalRetirementReadyTimeouts(t *testing.T) {
	now := time.Now()

	for _, tc := range []struct {
		name          string
		startedAt     time.Time
		apiStatus     metav1.ConditionStatus
		apiTransition time.Time
		coreNodes     []crd.EMQXNode
		ready         bool
		reason        string
	}{
		{
			name:      "fallback timeout reached",
			startedAt: now.Add(-corePodForcedRetirementTimeout[condFallback] - time.Second),
			ready:     true,
			reason:    "timeout exceeded",
		},
		{
			name:          "API unavailable timeout reached",
			startedAt:     now.Add(-2 * corePodForcedRetirementTimeout[condEMQXAPIUnavailable]),
			apiStatus:     metav1.ConditionFalse,
			apiTransition: now.Add(-2 * corePodForcedRetirementTimeout[condEMQXAPIUnavailable]),
			ready:         true,
			reason:        "API unavailability timeout exceeded",
		},
		{
			name:          "API unavailability predates retirement",
			startedAt:     now.Add(-corePodForcedRetirementTimeout[condEMQXAPIUnavailable] / 2),
			apiStatus:     metav1.ConditionFalse,
			apiTransition: now.Add(-2 * corePodForcedRetirementTimeout[condEMQXAPIUnavailable]),
			reason:        "cluster membership state is unknown",
		},
		{
			name:          "retirement predates API unavailability",
			startedAt:     now.Add(-2 * corePodForcedRetirementTimeout[condEMQXAPIUnavailable]),
			apiStatus:     metav1.ConditionFalse,
			apiTransition: now.Add(-corePodForcedRetirementTimeout[condEMQXAPIUnavailable] / 2),
			reason:        "cluster membership state is unknown",
		},
		{
			name:          "API available",
			startedAt:     now.Add(-2 * corePodForcedRetirementTimeout[condEMQXAPIUnavailable]),
			apiStatus:     metav1.ConditionTrue,
			apiTransition: now.Add(-2 * corePodForcedRetirementTimeout[condEMQXAPIUnavailable]),
			coreNodes:     []crd.EMQXNode{{Name: "emqx@emqx", PodName: "emqx-core-1", Status: "running"}},
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
			reconciler := &retireCorePods{}
			coreSet := &appsv1.StatefulSet{}
			decision := reconciler.corePodRetirementReady(
				&reconcileRound{
					state:          &reconcileState{coreSets: []*appsv1.StatefulSet{coreSet}},
					coreRetirement: &coreSetRetirement{watermark: 1, updatedAt: tc.startedAt},
				},
				instance,
				"emqx-core-4",
			)
			assert.Equal(t, tc.ready, decision.ready)
			assert.Contains(t, decision.reason, tc.reason)
		})
	}
}
