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

func TestEnterpriseAddCondition(t *testing.T) {
	status := &EmqxBrokerStatus{}

	status.AddCondition(ConditionInitResourceReady, v1.ConditionTrue, "fake", "fake")
	assert.Equal(t, 1, len(status.Conditions))
}
