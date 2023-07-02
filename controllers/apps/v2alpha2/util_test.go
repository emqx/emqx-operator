package v2alpha2

import (
	"testing"
	"time"

	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCheckInitialDelaySecondsReady(t *testing.T) {
	assert.False(t, checkInitialDelaySecondsReady(&appsv2alpha2.EMQX{}))

	assert.False(t, checkInitialDelaySecondsReady(&appsv2alpha2.EMQX{
		Spec: appsv2alpha2.EMQXSpec{
			UpdateStrategy: appsv2alpha2.UpdateStrategy{
				InitialDelaySeconds: 999999999,
			},
		},
		Status: appsv2alpha2.EMQXStatus{
			Conditions: []metav1.Condition{
				{
					Type:               appsv2alpha2.Available,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Time{Time: time.Now()},
				},
			},
		},
	}))

	assert.True(t, checkInitialDelaySecondsReady(&appsv2alpha2.EMQX{
		Spec: appsv2alpha2.EMQXSpec{
			UpdateStrategy: appsv2alpha2.UpdateStrategy{
				InitialDelaySeconds: 0,
			},
		},
		Status: appsv2alpha2.EMQXStatus{
			Conditions: []metav1.Condition{
				{
					Type:               appsv2alpha2.Available,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
				},
			},
		},
	}))
}

func TestCheckWaitTakeoverReady(t *testing.T) {
	t.Run("event list is empty", func(t *testing.T) {
		assert.True(t, checkWaitTakeoverReady(&appsv2alpha2.EMQX{}, []*corev1.Event{}))
	})

	t.Run("event list is not empty, waitTakeover not ready", func(t *testing.T) {
		emqx := &appsv2alpha2.EMQX{
			Spec: appsv2alpha2.EMQXSpec{
				UpdateStrategy: appsv2alpha2.UpdateStrategy{
					InitialDelaySeconds: 0,
					EvacuationStrategy: appsv2alpha2.EvacuationStrategy{
						WaitTakeover: 999999999,
					},
				},
			},
		}

		eventList := []*corev1.Event{
			{
				LastTimestamp: metav1.Time{Time: time.Now().AddDate(0, 0, 1)},
			},
		}

		assert.False(t, checkWaitTakeoverReady(emqx, eventList))
	})

	t.Run("event list is not empty, waitTakeover is ready", func(t *testing.T) {
		emqx := &appsv2alpha2.EMQX{
			Spec: appsv2alpha2.EMQXSpec{
				UpdateStrategy: appsv2alpha2.UpdateStrategy{
					InitialDelaySeconds: 0,
					EvacuationStrategy: appsv2alpha2.EvacuationStrategy{
						WaitTakeover: 0,
					},
				},
			},
		}

		eventList := []*corev1.Event{
			{
				LastTimestamp: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
			},
		}

		assert.True(t, checkWaitTakeoverReady(emqx, eventList))
	})
}

func TestHandlerEventList(t *testing.T) {
	t.Run("filter event", func(t *testing.T) {
		list := &corev1.EventList{
			Items: []corev1.Event{
				{
					Reason: "SuccessfulCreate",
				},
				{
					Reason: "SuccessfulDelete",
				},
			},
		}
		assert.Len(t, handlerEventList(list), 1)
	})

	t.Run("sort event list", func(t *testing.T) {
		list := &corev1.EventList{
			Items: []corev1.Event{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "emqx-1",
					},
					LastTimestamp: metav1.Time{Time: time.Now().AddDate(0, 0, 1)},
					Reason:        "SuccessfulDelete",
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "emqx-0",
					},
					LastTimestamp: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
					Reason:        "SuccessfulDelete",
				},
			},
		}

		var l []string
		for _, e := range handlerEventList(list) {
			l = append(l, e.DeepCopy().Name)
		}
		assert.ElementsMatch(t, []string{"emqx-0", "emqx-1"}, l)
	})
}
