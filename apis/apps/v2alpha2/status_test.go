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

package v2alpha2

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetNodes(t *testing.T) {
	status := &EMQXStatus{}

	nodes := []EMQXNode{
		{
			Node:   "emqx-0",
			Role:   "core",
			Uptime: 10000,
		},
		{
			Node:   "emqx-1",
			Role:   "core",
			Uptime: 10,
		},
		{
			Node:   "emqx-2",
			Role:   "replicant",
			Uptime: 10000,
		},
		{
			Node:   "emqx-3",
			Role:   "replicant",
			Uptime: 10,
		},
	}
	status.SetNodes(nodes)

	assert.Equal(t, int32(2), status.CoreNodeStatus.ReadyReplicas)
	assert.Equal(t, []EMQXNode{
		{
			Node:   "emqx-1",
			Role:   "core",
			Uptime: 10,
		},
		{
			Node:   "emqx-0",
			Role:   "core",
			Uptime: 10000,
		},
	}, status.CoreNodeStatus.Nodes)

	assert.Equal(t, int32(2), status.ReplicantNodeStatus.ReadyReplicas)
	assert.Equal(t, []EMQXNode{
		{
			Node:   "emqx-3",
			Role:   "replicant",
			Uptime: 10,
		},
		{
			Node:   "emqx-2",
			Role:   "replicant",
			Uptime: 10000,
		},
	}, status.ReplicantNodeStatus.Nodes)
}

func TestSetCondition(t *testing.T) {
	c0 := metav1.Condition{
		Type:   Initialized,
		Status: metav1.ConditionFalse,
	}

	c1 := metav1.Condition{
		Type:   Ready,
		Status: metav1.ConditionFalse,
	}

	c2 := metav1.Condition{
		Type:   Ready,
		Status: metav1.ConditionTrue,
	}

	c3 := c2

	t.Run("add condition", func(t *testing.T) {
		status := &EMQXStatus{}

		status.SetCondition(c0)
		assert.Equal(t, 1, len(status.Conditions))

		c0 = status.Conditions[0]
		assert.NotEmpty(t, c0.LastTransitionTime)
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
	})
}

func TestGetLastTrueCondition(t *testing.T) {
	status := &EMQXStatus{
		Conditions: []metav1.Condition{
			{
				Type:   CoreNodesProgressing,
				Status: metav1.ConditionFalse,
			},
			{
				Type:   Initialized,
				Status: metav1.ConditionTrue,
			},
		},
	}

	c := status.GetLastTrueCondition()
	assert.Equal(t, Initialized, c.Type)
}

func TestGetCondition(t *testing.T) {
	status := &EMQXStatus{
		Conditions: []metav1.Condition{
			{
				Type:   CoreNodesProgressing,
				Status: metav1.ConditionTrue,
			},
			{
				Type:   Initialized,
				Status: metav1.ConditionFalse,
			},
		},
	}

	var pos int

	pos, _ = status.GetCondition(CoreNodesProgressing)
	assert.Equal(t, 0, pos)

	pos, _ = status.GetCondition(Initialized)
	assert.Equal(t, 1, pos)
}

func TestIsConditionTrue(t *testing.T) {
	status := &EMQXStatus{
		Conditions: []metav1.Condition{
			{
				Type:   Initialized,
				Status: metav1.ConditionTrue,
			},
			{
				Type:   Ready,
				Status: metav1.ConditionFalse,
			},
		},
	}

	assert.True(t, status.IsConditionTrue(Initialized))
	assert.False(t, status.IsConditionTrue("Nothing"))
	assert.False(t, status.IsConditionTrue(Ready))
}

func TestRemoveCondition(t *testing.T) {
	status := &EMQXStatus{
		Conditions: []metav1.Condition{
			{
				Type:   Initialized,
				Status: metav1.ConditionTrue,
			},
			{
				Type:   Ready,
				Status: metav1.ConditionFalse,
			},
		},
	}
	status.RemoveCondition(Initialized)
	assert.Equal(t, 1, len(status.Conditions))
}
