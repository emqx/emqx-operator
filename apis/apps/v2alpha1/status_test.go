/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v2alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestIsCoreNodesUpdating(t *testing.T) {
	status := EMQXStatus{
		Conditions: []Condition{
			{
				Type:   ClusterCoreUpdating,
				Status: v1.ConditionTrue,
			},
		},
	}

	got := status.IsCoreNodesUpdating()
	assert.True(t, got)

	status = EMQXStatus{}
	got = status.IsCoreNodesUpdating()
	assert.False(t, got)

	status = EMQXStatus{
		Conditions: []Condition{
			{
				Type:   ClusterRunning,
				Status: v1.ConditionTrue,
			},
			{
				Type:   ClusterCoreReady,
				Status: v1.ConditionTrue,
			},
		},
	}
	got = status.IsCoreNodesUpdating()
	assert.False(t, got)
}

func TestIsCoreNodesReady(t *testing.T) {
	status := EMQXStatus{
		Conditions: []Condition{
			{
				Type:   ClusterCoreReady,
				Status: v1.ConditionTrue,
			},
		},
	}

	got := status.IsCoreNodesReady()
	assert.True(t, got)

	status = EMQXStatus{}
	got = status.IsCoreNodesReady()
	assert.False(t, got)

	status = EMQXStatus{
		Conditions: []Condition{
			{
				Type:   ClusterRunning,
				Status: v1.ConditionTrue,
			},
			{
				Type:   ClusterCoreReady,
				Status: v1.ConditionTrue,
			},
		},
	}
	got = status.IsCoreNodesReady()
	assert.False(t, got)
}

func TestIsRunning(t *testing.T) {
	status := EMQXStatus{
		Conditions: []Condition{
			{
				Type:   ClusterRunning,
				Status: v1.ConditionTrue,
			},
		},
	}

	got := status.IsRunning()
	assert.True(t, got)

	status = EMQXStatus{}
	got = status.IsRunning()
	assert.False(t, got)

	status = EMQXStatus{
		Conditions: []Condition{
			{
				Type:   ClusterRunning,
				Status: v1.ConditionFalse,
			},
		},
	}
	got = status.IsRunning()
	assert.False(t, got)
}

func TestIsCreating(t *testing.T) {
	status := &EMQXStatus{
		Conditions: []Condition{
			{
				Type:   ClusterCreating,
				Status: v1.ConditionTrue,
			},
		},
	}

	got := status.IsCreating()
	assert.True(t, got)

	status = &EMQXStatus{}
	got = status.IsCreating()
	assert.False(t, got)

	status = &EMQXStatus{
		Conditions: []Condition{
			{
				Type:   ClusterCreating,
				Status: v1.ConditionFalse,
			},
		},
	}
	got = status.IsCreating()
	assert.False(t, got)
}

func TestIndexCondition(t *testing.T) {
	status := &EMQXStatus{
		Conditions: []Condition{
			{
				Type:   ClusterCoreUpdating,
				Status: v1.ConditionTrue,
			},
			{
				Type:   ClusterCreating,
				Status: v1.ConditionFalse,
			},
		},
	}
	idx := indexCondition(status, ClusterCoreUpdating)
	assert.Equal(t, 0, idx)

	idx = indexCondition(status, ClusterCreating)
	assert.Equal(t, 1, idx)
}

func TestSetCondition(t *testing.T) {
	c0 := Condition{
		Type:   ClusterCreating,
		Status: v1.ConditionFalse,
	}

	c1 := Condition{
		Type:   ClusterRunning,
		Status: v1.ConditionFalse,
	}

	c2 := Condition{
		Type:   ClusterRunning,
		Status: v1.ConditionTrue,
	}

	c3 := c2

	t.Run("add condition", func(t *testing.T) {
		status := &EMQXStatus{}

		status.SetCondition(c0)
		assert.Equal(t, 1, len(status.Conditions))

		c0 = status.Conditions[0]
		assert.NotEmpty(t, c0.LastTransitionTime)
		assert.NotEmpty(t, c0.LastUpdateTime)
		assert.NotEmpty(t, c0.LastUpdateAt)
	})

	t.Run("add different condition type", func(t *testing.T) {
		status := &EMQXStatus{}

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
		status := &EMQXStatus{}

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
		status := &EMQXStatus{}

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
