package v1beta4

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestEnterpriseIsRuning(t *testing.T) {
	status := EmqxEnterpriseStatus{
		Conditions: []Condition{
			{
				Type:   ConditionRunning,
				Status: v1.ConditionTrue,
			},
		},
	}

	got := status.IsRunning()
	assert.True(t, got)

	status = EmqxEnterpriseStatus{}
	got = status.IsRunning()
	assert.False(t, got)

	status = EmqxEnterpriseStatus{
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

func TestEnterpriseIsInitResourceReady(t *testing.T) {
	status := &EmqxEnterpriseStatus{
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

	status = &EmqxEnterpriseStatus{}
	got = status.IsInitResourceReady()
	assert.False(t, got)

	status = &EmqxEnterpriseStatus{
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

func TestEnterpriseSetCondition(t *testing.T) {
	status := &EmqxEnterpriseStatus{
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
