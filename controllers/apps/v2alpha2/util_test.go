package v2alpha2

import (
	"testing"
	"time"

	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestCanBeScaledDown(t *testing.T) {
	t.Run("event list is empty, current replicaSet is not available, can not scale down", func(t *testing.T) {
		assert.False(t, canBeScaledDown(&appsv2alpha2.EMQX{}, appsv2alpha2.CodeNodesReady, []*corev1.Event{}))
	})

	t.Run("event list is empty, initialDelaySeconds not ready, can not scale down", func(t *testing.T) {
		emqx := &appsv2alpha2.EMQX{
			Spec: appsv2alpha2.EMQXSpec{
				UpdateStrategy: appsv2alpha2.UpdateStrategy{
					InitialDelaySeconds: 999999999,
				},
			},
			Status: appsv2alpha2.EMQXStatus{
				Conditions: []metav1.Condition{
					{Type: appsv2alpha2.CodeNodesReady, Status: metav1.ConditionTrue},
				},
			},
		}
		assert.False(t, canBeScaledDown(emqx, appsv2alpha2.CodeNodesReady, []*corev1.Event{}))
	})

	t.Run("event list is empty, initialDelaySeconds is ready, can scale down", func(t *testing.T) {
		emqx := &appsv2alpha2.EMQX{
			Spec: appsv2alpha2.EMQXSpec{
				UpdateStrategy: appsv2alpha2.UpdateStrategy{
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

		assert.True(t, canBeScaledDown(emqx, appsv2alpha2.CodeNodesReady, []*corev1.Event{}))
	})

	t.Run("event list not empty, current replicaSet is not available, can not scale down", func(t *testing.T) {
		assert.False(t, canBeScaledDown(&appsv2alpha2.EMQX{}, appsv2alpha2.CodeNodesReady, []*corev1.Event{
			{
				LastTimestamp: metav1.Time{Time: time.Now().AddDate(0, 0, 1)},
			},
		}))
	})

	t.Run("event list is not empty, initialDelaySeconds is ready, waitTakeover not ready, can not scale down", func(t *testing.T) {
		emqx := &appsv2alpha2.EMQX{
			Spec: appsv2alpha2.EMQXSpec{
				UpdateStrategy: appsv2alpha2.UpdateStrategy{
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

		assert.False(t, canBeScaledDown(emqx, appsv2alpha2.CodeNodesReady, eventList))
	})

	t.Run("event list is not empty,initialDelaySeconds is ready, waitTakeover is ready, can scale down", func(t *testing.T) {
		emqx := &appsv2alpha2.EMQX{
			Spec: appsv2alpha2.EMQXSpec{
				UpdateStrategy: appsv2alpha2.UpdateStrategy{
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

		assert.True(t, canBeScaledDown(emqx, appsv2alpha2.CodeNodesReady, eventList))
	})
}

func TestHandlerStatefulSetList(t *testing.T) {
	t.Run("filter not ready statefulSet", func(t *testing.T) {
		list := &appsv1.StatefulSetList{
			Items: []appsv1.StatefulSet{
				{
					Spec: appsv1.StatefulSetSpec{
						Replicas: pointer.Int32(0),
					},
					Status: appsv1.StatefulSetStatus{
						Replicas: 0,
					},
				},
				{
					Spec: appsv1.StatefulSetSpec{
						Replicas: pointer.Int32(1),
					},
					Status: appsv1.StatefulSetStatus{
						Replicas:      1,
						ReadyReplicas: 0,
					},
				},
				{
					Spec: appsv1.StatefulSetSpec{
						Replicas: pointer.Int32(1),
					},
					Status: appsv1.StatefulSetStatus{
						Replicas:      1,
						ReadyReplicas: 1,
					},
				},
			},
		}
		assert.Len(t, handlerStatefulSetList(list), 1)
	})

	t.Run("sort statefulSet list", func(t *testing.T) {
		list := &appsv1.StatefulSetList{
			Items: []appsv1.StatefulSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "emqx-1",
						CreationTimestamp: metav1.Time{Time: time.Now().AddDate(0, 0, 1)},
					},
					Spec: appsv1.StatefulSetSpec{
						Replicas: pointer.Int32(1),
					},
					Status: appsv1.StatefulSetStatus{
						Replicas:      1,
						ReadyReplicas: 1,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "emqx-0",
						CreationTimestamp: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
					},
					Spec: appsv1.StatefulSetSpec{
						Replicas: pointer.Int32(1),
					},
					Status: appsv1.StatefulSetStatus{
						Replicas:      1,
						ReadyReplicas: 1,
					},
				},
			},
		}

		var l []string
		for _, d := range handlerStatefulSetList(list) {
			l = append(l, d.DeepCopy().Name)
		}
		assert.ElementsMatch(t, []string{"emqx-0", "emqx-1"}, l)
	})
}

func TestHandlerReplicaSetList(t *testing.T) {
	t.Run("filter not ready replicaSet", func(t *testing.T) {
		list := &appsv1.ReplicaSetList{
			Items: []appsv1.ReplicaSet{
				{
					Spec: appsv1.ReplicaSetSpec{
						Replicas: pointer.Int32(0),
					},
					Status: appsv1.ReplicaSetStatus{
						Replicas: 0,
					},
				},
				{
					Spec: appsv1.ReplicaSetSpec{
						Replicas: pointer.Int32(1),
					},
					Status: appsv1.ReplicaSetStatus{
						Replicas:      1,
						ReadyReplicas: 0,
					},
				},
				{
					Spec: appsv1.ReplicaSetSpec{
						Replicas: pointer.Int32(1),
					},
					Status: appsv1.ReplicaSetStatus{
						Replicas:      1,
						ReadyReplicas: 1,
					},
				},
			},
		}
		assert.Len(t, handlerReplicaSetList(list), 1)
	})

	t.Run("sort replicaSet list", func(t *testing.T) {
		list := &appsv1.ReplicaSetList{
			Items: []appsv1.ReplicaSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "emqx-1",
						CreationTimestamp: metav1.Time{Time: time.Now().AddDate(0, 0, 1)},
					},
					Spec: appsv1.ReplicaSetSpec{
						Replicas: pointer.Int32(1),
					},
					Status: appsv1.ReplicaSetStatus{
						Replicas:      1,
						ReadyReplicas: 1,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "emqx-0",
						CreationTimestamp: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
					},
					Spec: appsv1.ReplicaSetSpec{
						Replicas: pointer.Int32(1),
					},
					Status: appsv1.ReplicaSetStatus{
						Replicas:      1,
						ReadyReplicas: 1,
					},
				},
			},
		}

		var l []string
		for _, d := range handlerReplicaSetList(list) {
			l = append(l, d.DeepCopy().Name)
		}
		assert.ElementsMatch(t, []string{"emqx-0", "emqx-1"}, l)
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
