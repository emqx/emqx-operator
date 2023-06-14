package v2alpha2

import (
	"testing"
	"time"

	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCanBeScaledDownRs(t *testing.T) {
	t.Run("event list is empty, current replicaSet is not available, can not scale down", func(t *testing.T) {
		assert.False(t, canBeScaledDownRs(&appsv2alpha2.EMQX{}, &appsv1.ReplicaSet{}, []*corev1.Event{}))
	})

	t.Run("event list is empty, initialDelaySeconds not ready, can not scale down", func(t *testing.T) {
		emqx := &appsv2alpha2.EMQX{
			Spec: appsv2alpha2.EMQXSpec{
				BlueGreenUpdate: appsv2alpha2.BlueGreenUpdate{
					InitialDelaySeconds: 999999999,
				},
			},
			Status: appsv2alpha2.EMQXStatus{
				Conditions: []metav1.Condition{
					{Type: appsv2alpha2.CodeNodesReady, Status: metav1.ConditionTrue},
				},
			},
		}
		assert.False(t, canBeScaledDownRs(emqx, &appsv1.ReplicaSet{}, []*corev1.Event{}))
	})

	t.Run("event list is empty, initialDelaySeconds is ready, can scale down", func(t *testing.T) {
		emqx := &appsv2alpha2.EMQX{
			Spec: appsv2alpha2.EMQXSpec{
				BlueGreenUpdate: appsv2alpha2.BlueGreenUpdate{
					InitialDelaySeconds: 1,
				},
			},
			Status: appsv2alpha2.EMQXStatus{
				Conditions: []metav1.Condition{
					{
						Type:               appsv2alpha2.CodeNodesReady,
						Status:             metav1.ConditionTrue,
						LastTransitionTime: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
					},
				},
			},
		}

		assert.True(t, canBeScaledDownRs(emqx, &appsv1.ReplicaSet{}, []*corev1.Event{}))
	})

	t.Run("event list not empty, current replicaSet is not available, can not scale down", func(t *testing.T) {
		assert.False(t, canBeScaledDownRs(&appsv2alpha2.EMQX{}, &appsv1.ReplicaSet{}, []*corev1.Event{
			{
				LastTimestamp: metav1.Time{Time: time.Now().AddDate(0, 0, 1)},
			},
		}))
	})

	t.Run("event list is not empty, initialDelaySeconds is ready, waitTakeover not ready, can not scale down", func(t *testing.T) {
		emqx := &appsv2alpha2.EMQX{
			Spec: appsv2alpha2.EMQXSpec{
				BlueGreenUpdate: appsv2alpha2.BlueGreenUpdate{
					InitialDelaySeconds: 1,
					EvacuationStrategy: appsv2alpha2.EvacuationStrategy{
						WaitTakeover: 999999999,
					},
				},
			},
			Status: appsv2alpha2.EMQXStatus{
				Conditions: []metav1.Condition{
					{
						Type:               appsv2alpha2.CodeNodesReady,
						Status:             metav1.ConditionTrue,
						LastTransitionTime: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
					},
				},
			},
		}

		eventList := []*corev1.Event{
			{
				LastTimestamp: metav1.Time{Time: time.Now().AddDate(0, 0, 1)},
			},
		}

		assert.False(t, canBeScaledDownRs(emqx, &appsv1.ReplicaSet{}, eventList))
	})

	t.Run("event list is not empty,initialDelaySeconds is ready, waitTakeover is ready, can scale down", func(t *testing.T) {
		emqx := &appsv2alpha2.EMQX{
			Spec: appsv2alpha2.EMQXSpec{
				BlueGreenUpdate: appsv2alpha2.BlueGreenUpdate{
					InitialDelaySeconds: 1,
					EvacuationStrategy: appsv2alpha2.EvacuationStrategy{
						WaitTakeover: 1,
					},
				},
			},
			Status: appsv2alpha2.EMQXStatus{
				Conditions: []metav1.Condition{
					{
						Type:               appsv2alpha2.CodeNodesReady,
						Status:             metav1.ConditionTrue,
						LastTransitionTime: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
					},
				},
			},
		}

		eventList := []*corev1.Event{
			{
				LastTimestamp: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
			},
		}

		assert.True(t, canBeScaledDownRs(emqx, &appsv1.ReplicaSet{}, eventList))
	})
}
