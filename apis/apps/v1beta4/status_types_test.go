package v1beta4

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestSetCondition(t *testing.T) {
	conditions := []Condition{}

	c1 := Condition{
		Type:   ConditionInitResourceReady,
		Status: v1.ConditionFalse,
	}
	got1 := addCondition(conditions, c1)
	assert.Len(t, got1, 1)
	assert.NotNil(t, got1[0].LastUpdateAt)
	assert.NotNil(t, got1[0].LastUpdateTime)
	assert.NotNil(t, got1[0].LastTransitionTime)

	c2 := c1
	c2.Status = v1.ConditionTrue

	got2 := addCondition(got1, c2)
	assert.Len(t, got2, 1)
	assert.NotNil(t, got2[0].LastUpdateAt)
	assert.NotNil(t, got2[0].LastUpdateTime)
	assert.Equal(t, got1[0].LastTransitionTime, got2[0].LastTransitionTime)

	c3 := Condition{
		Type:   ConditionRunning,
		Status: v1.ConditionTrue,
	}
	got3 := addCondition(got2, c3)
	assert.Len(t, got3, 2)

	assert.Equal(t, ConditionRunning, got3[0].Type)
	assert.Equal(t, ConditionInitResourceReady, got3[1].Type)
}

func TestIndexCondition(t *testing.T) {
	conditions := []Condition{
		{
			Type:   ConditionInitResourceReady,
			Status: v1.ConditionTrue,
		},
		{
			Type:   ConditionRunning,
			Status: v1.ConditionFalse,
		},
	}
	idx := indexCondition(conditions, ConditionInitResourceReady)
	assert.Equal(t, 0, idx)

	idx = indexCondition(conditions, ConditionRunning)
	assert.Equal(t, 1, idx)
}
