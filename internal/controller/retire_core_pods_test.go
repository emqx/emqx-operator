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
		name      string
		labels    map[string]string
		deletedAt time.Time
		ready     bool
		reason    string
	}{
		{
			name:      "force label",
			labels:    map[string]string{crd.LabelForceRetirement: "true"},
			deletedAt: now,
			ready:     true,
			reason:    "retirement forced",
		},
		{
			name:      "timeout reached",
			deletedAt: now.Add(-corePodRetirementTimeout - time.Second),
			ready:     true,
			reason:    "timeout exceeded",
		},
		{
			name:      "timeout pending",
			deletedAt: now,
			reason:    "DS cluster state is not loaded",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
				Labels:            tc.labels,
				DeletionTimestamp: &metav1.Time{Time: tc.deletedAt},
			}}
			ready, reason := (&retireCorePods{}).corePodRetirementReady(
				&reconcileRound{},
				&crd.EMQX{},
				pod,
			)
			assert.Equal(t, tc.ready, ready)
			assert.Contains(t, reason, tc.reason)
		})
	}
}
