package v1beta3

import (
	"testing"
	"time"

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

func TestIsPluginInitialized(t *testing.T) {
	status := &Status{
		Conditions: []Condition{
			{
				Type:   ConditionPluginInitialized,
				Status: v1.ConditionTrue,
			},
			{
				Type:   ConditionRunning,
				Status: v1.ConditionTrue,
			},
		},
	}

	got := status.IsPluginInitialized()
	assert.True(t, got)

	status = &Status{}
	got = status.IsPluginInitialized()
	assert.False(t, got)

	status = &Status{
		Conditions: []Condition{
			{
				Type:   ConditionPluginInitialized,
				Status: v1.ConditionFalse,
			},
		},
	}
	got = status.IsPluginInitialized()
	assert.False(t, got)
}

func TestIndexCondition(t *testing.T) {
	status := &Status{
		Conditions: []Condition{
			{
				Type:   ConditionPluginInitialized,
				Status: v1.ConditionTrue,
			},
			{
				Type:   ConditionRunning,
				Status: v1.ConditionFalse,
			},
		},
	}
	idx := indexCondition(status, ConditionPluginInitialized)
	assert.Equal(t, 0, idx)

	idx = indexCondition(status, ConditionRunning)
	assert.Equal(t, 1, idx)
}

func TestSetCondition(t *testing.T) {
	c0 := Condition{
		Type:   ConditionPluginInitialized,
		Status: v1.ConditionFalse,
	}

	c1 := Condition{
		Type:   ConditionRunning,
		Status: v1.ConditionFalse,
	}

	c2 := Condition{
		Type:   ConditionRunning,
		Status: v1.ConditionTrue,
	}

	c3 := c2

	t.Run("add condition", func(t *testing.T) {
		status := &Status{}

		status.SetCondition(c0)
		assert.Equal(t, 1, len(status.Conditions))

		c0 = status.Conditions[0]
		assert.NotEmpty(t, c0.LastTransitionTime)
		assert.NotEmpty(t, c0.LastUpdateTime)
		assert.NotEmpty(t, c0.LastUpdateAt)
	})

	t.Run("add different condition type", func(t *testing.T) {
		status := &Status{}

		status.SetCondition(c0)
		c0 = status.Conditions[0]
		time.Sleep(time.Millisecond * time.Duration(1500))
		status.SetCondition(c1)
		c1 = status.Conditions[0]

		assert.Equal(t, 2, len(status.Conditions))
		assert.NotEmpty(t, c1.LastTransitionTime)
		assert.NotEqual(t, c0.LastTransitionTime, c1.LastTransitionTime)
		assert.NotEmpty(t, c1.LastUpdateTime)
		assert.NotEqual(t, c0.LastUpdateTime, c1.LastUpdateTime)
		assert.NotEmpty(t, c1.LastUpdateAt)
		assert.NotEqual(t, c0.LastUpdateAt, c1.LastUpdateAt)
	})

	t.Run("add same condition type, but different condition status", func(t *testing.T) {
		status := &Status{}

		status.SetCondition(c1)
		c1 = status.Conditions[0]
		time.Sleep(time.Millisecond * time.Duration(1500))
		status.SetCondition(c2)
		c2 = status.Conditions[0]

		assert.Equal(t, 1, len(status.Conditions))
		assert.NotEmpty(t, c2.LastTransitionTime)
		assert.NotEqual(t, c1.LastTransitionTime, c2.LastTransitionTime)
		assert.NotEmpty(t, c2.LastUpdateTime)
		assert.NotEqual(t, c1.LastUpdateTime, c2.LastUpdateTime)
		assert.NotEmpty(t, c2.LastUpdateAt)
		assert.NotEqual(t, c1.LastUpdateAt, c2.LastUpdateAt)
	})

	t.Run("add same condition type and same condition status", func(t *testing.T) {
		status := &Status{}

		status.SetCondition(c2)
		c2 = status.Conditions[0]
		time.Sleep(time.Millisecond * time.Duration(1500))
		status.SetCondition(c3)
		c3 = status.Conditions[0]

		assert.Equal(t, 1, len(status.Conditions))
		assert.NotEmpty(t, c3.LastTransitionTime)
		assert.Equal(t, c2.LastTransitionTime, c3.LastTransitionTime)
		assert.NotEmpty(t, c3.LastUpdateTime)
		assert.NotEqual(t, c2.LastUpdateTime, c3.LastUpdateTime)
		assert.NotEmpty(t, c3.LastUpdateAt)
		assert.NotEqual(t, c2.LastUpdateAt, c3.LastUpdateAt)
	})
}
