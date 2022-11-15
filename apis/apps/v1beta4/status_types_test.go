package v1beta4

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestIsRuning(t *testing.T) {
	status := Status{
		Conditions: []Condition{
			{
				Type:   ConditionRunning,
				Status: v1.ConditionTrue,
			},
		},
	}

	got := status.IsRunning()
	assert.True(t, got)

	status = Status{}
	got = status.IsRunning()
	assert.False(t, got)

	status = Status{
		Conditions: []Condition{
			{
				Type:   ConditionRunning,
				Status: v1.ConditionFalse,
			},
		},
	}
	got = status.IsRunning()
	assert.False(t, got)
}

func TestIsInitResourceReady(t *testing.T) {
	status := &Status{
		Conditions: []Condition{
			{
				Type:   ConditionInitResourceReady,
				Status: v1.ConditionTrue,
			},
			{
				Type:   ConditionRunning,
				Status: v1.ConditionTrue,
			},
		},
	}

	got := status.IsInitResourceReady()
	assert.False(t, got)

	status = &Status{}
	got = status.IsInitResourceReady()
	assert.False(t, got)

	status = &Status{
		Conditions: []Condition{
			{
				Type:   ConditionInitResourceReady,
				Status: v1.ConditionTrue,
			},
		},
	}
	got = status.IsInitResourceReady()
	assert.True(t, got)
}

func TestIndexCondition(t *testing.T) {
	status := &Status{
		Conditions: []Condition{
			{
				Type:   ConditionInitResourceReady,
				Status: v1.ConditionTrue,
			},
			{
				Type:   ConditionRunning,
				Status: v1.ConditionFalse,
			},
		},
	}
	idx := indexCondition(status, ConditionInitResourceReady)
	assert.Equal(t, 0, idx)

	idx = indexCondition(status, ConditionRunning)
	assert.Equal(t, 1, idx)
}

func TestSetCondition(t *testing.T) {
	status := &Status{
		Conditions: []Condition{
			{
				Type:   ConditionInitResourceReady,
				Status: v1.ConditionFalse,
			},
		},
	}
	c := Condition{
		Type:   ConditionInitResourceReady,
		Status: v1.ConditionTrue,
	}
	status.SetCondition(c)
	conditions := status.GetConditions()

	assert.Equal(t, 1, len(conditions))
	assert.Equal(t, c.LastTransitionTime, conditions[0].LastTransitionTime)
	assert.NotEqual(t, c.LastUpdateAt, conditions[0].LastUpdateAt)
	assert.NotEqual(t, c.LastUpdateTime, conditions[0].LastUpdateTime)

	c = Condition{
		Type:   ConditionRunning,
		Status: v1.ConditionTrue,
	}
	status.SetCondition(c)
	conditions = status.GetConditions()

	assert.Equal(t, 2, len(conditions))
	assert.NotEqual(t, c.LastTransitionTime, conditions[0].LastTransitionTime)
	assert.NotEqual(t, c.LastUpdateAt, conditions[0].LastUpdateAt)
	assert.NotEqual(t, c.LastUpdateTime, conditions[0].LastUpdateTime)
}
