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

func TestIsRuning(t *testing.T) {
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
	status := &EMQXStatus{
		Conditions: []Condition{
			{
				Type:   ClusterCreating,
				Status: v1.ConditionFalse,
			},
		},
	}
	c := Condition{
		Type:   ClusterCreating,
		Status: v1.ConditionTrue,
	}
	status.SetCondition(c)
	conditions := status.Conditions

	assert.Equal(t, 1, len(conditions))
	assert.Equal(t, c.LastTransitionTime, conditions[0].LastTransitionTime)
	assert.NotEqual(t, c.LastUpdateAt, conditions[0].LastUpdateAt)
	assert.NotEqual(t, c.LastUpdateTime, conditions[0].LastUpdateTime)

	c = Condition{
		Type:   ClusterCoreUpdating,
		Status: v1.ConditionTrue,
	}
	status.SetCondition(c)
	conditions = status.Conditions

	assert.Equal(t, 2, len(conditions))
	assert.NotEqual(t, c.LastTransitionTime, conditions[0].LastTransitionTime)
	assert.NotEqual(t, c.LastUpdateAt, conditions[0].LastUpdateAt)
	assert.NotEqual(t, c.LastUpdateTime, conditions[0].LastUpdateTime)
}
