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

package v1beta4

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestSetFailed(t *testing.T) {
	t.Run("condition type must be Failed", func(t *testing.T) {
		r := &Rebalance{}
		c := RebalanceCondition{}
		c.Type = RebalanceCompleted
		assert.ErrorContains(t, r.Status.SetFailed(c), "condition type must be Failed")
	})

	t.Run("set successfully", func(t *testing.T) {
		r := &Rebalance{}
		c := RebalanceCondition{
			Type: RebalanceFailed,
		}
		assert.Nil(t, r.Status.SetFailed(c))
		assert.Equal(t, RebalancePhaseFailed, r.Status.Phase)
		assert.Equal(t, c.Type, r.Status.Conditions[0].Type)
	})
}

func TestSetCompleted(t *testing.T) {
	t.Run("rebalance job is not in processing", func(t *testing.T) {
		r := &Rebalance{}
		c := RebalanceCondition{}
		assert.ErrorContains(t, r.Status.SetCompleted(c), "rebalance job is not in processing")
	})
	t.Run("condition type must be Completed", func(t *testing.T) {
		r := &Rebalance{
			Status: RebalanceStatus{
				Phase: RebalancePhaseProcessing,
			},
		}
		c := RebalanceCondition{}
		assert.ErrorContains(t, r.Status.SetCompleted(c), "condition type must be Completed")
	})

	t.Run("set successfully", func(t *testing.T) {
		r := &Rebalance{
			Status: RebalanceStatus{
				Phase: RebalancePhaseProcessing,
			},
		}
		c := RebalanceCondition{
			Type: RebalanceCompleted,
		}

		assert.Nil(t, r.Status.SetCompleted(c))
		assert.Equal(t, RebalancePhaseCompleted, r.Status.Phase)
		assert.Equal(t, c.Type, r.Status.Conditions[0].Type)
	})
}

func TestSetProcessing(t *testing.T) {
	t.Run("rebalance job has been failed", func(t *testing.T) {
		r := &Rebalance{
			Status: RebalanceStatus{
				Phase: RebalancePhaseFailed,
			},
		}
		c := RebalanceCondition{}
		assert.ErrorContains(t, r.Status.SetProcessing(c), "rebalance job has been failed")
	})

	t.Run("rebalance job has been completed", func(t *testing.T) {
		r := &Rebalance{
			Status: RebalanceStatus{
				Phase: RebalancePhaseCompleted,
			},
		}
		c := RebalanceCondition{}
		assert.ErrorContains(t, r.Status.SetProcessing(c), "rebalance job has been completed")
	})

	t.Run("condition type must be Processing", func(t *testing.T) {
		r := &Rebalance{
			Status: RebalanceStatus{
				Phase: RebalancePhaseProcessing,
			},
		}
		c := RebalanceCondition{}
		assert.ErrorContains(t, r.Status.SetProcessing(c), "condition type must be Processing")
	})

	t.Run("set successfully", func(t *testing.T) {
		r := &Rebalance{
			Status: RebalanceStatus{
				Phase: RebalancePhaseProcessing,
			},
		}
		c := RebalanceCondition{
			Type: RebalanceProcessing,
		}

		assert.Nil(t, r.Status.SetProcessing(c))
		assert.Equal(t, RebalancePhaseProcessing, r.Status.Phase)
		assert.Equal(t, c.Type, r.Status.Conditions[0].Type)
	})
}

func TestSetRebalanceCondition(t *testing.T) {
	t.Run("add condition", func(t *testing.T) {
		r := &Rebalance{
			Status: RebalanceStatus{},
		}

		c0 := RebalanceCondition{
			Type:   RebalanceCompleted,
			Status: v1.ConditionTrue,
		}

		r.Status.SetCondition(c0)
		assert.Equal(t, 1, len(r.Status.Conditions))

		assert.NotEmpty(t, r.Status.Conditions[0].LastTransitionTime)
		assert.NotEmpty(t, r.Status.Conditions[0].LastUpdateTime)
	})

	t.Run("add different condition type", func(t *testing.T) {
		r := &Rebalance{
			Status: RebalanceStatus{},
		}

		c0 := RebalanceCondition{
			Type:   RebalanceCompleted,
			Status: v1.ConditionTrue,
		}

		c1 := RebalanceCondition{
			Type:   RebalanceProcessing,
			Status: v1.ConditionTrue,
		}

		r.Status.SetCondition(c0)
		c0 = r.Status.Conditions[0]
		time.Sleep(time.Millisecond * time.Duration(1500))
		r.Status.SetCondition(c1)
		c1 = r.Status.Conditions[0]

		assert.Equal(t, 2, len(r.Status.Conditions))
		assert.NotEmpty(t, c0.LastTransitionTime)
		assert.NotEqual(t, c0.LastTransitionTime, c1.LastTransitionTime)
		assert.NotEmpty(t, c1.LastUpdateTime)
		assert.NotEqual(t, c0.LastUpdateTime, c1.LastUpdateTime)
		assert.NotEqual(t, c0.Type, c1.Type)

		c0 = r.Status.Conditions[0]
		c1 = r.Status.Conditions[1]
		assert.False(t, c0.LastUpdateTime.Before(&c1.LastUpdateTime))
	})

	t.Run("add same condition type, but different condition status", func(t *testing.T) {
		r := &Rebalance{
			Status: RebalanceStatus{},
		}

		c1 := RebalanceCondition{
			Type:   RebalanceProcessing,
			Status: v1.ConditionTrue,
		}

		c2 := RebalanceCondition{
			Type:   RebalanceProcessing,
			Status: v1.ConditionFalse,
		}

		r.Status.SetCondition(c1)
		c1 = r.Status.Conditions[0]
		time.Sleep(time.Millisecond * time.Duration(1500))
		r.Status.SetCondition(c2)
		c2 = r.Status.Conditions[0]

		assert.Equal(t, 1, len(r.Status.Conditions))
		assert.NotEmpty(t, c2.LastTransitionTime)
		assert.NotEqual(t, c1.LastTransitionTime, c2.LastTransitionTime)
		assert.NotEmpty(t, c2.LastUpdateTime)
		assert.NotEqual(t, c1.LastUpdateTime, c2.LastUpdateTime)
		assert.Equal(t, c1.Type, c2.Type)

	})

	t.Run("add same condition type and same condition status", func(t *testing.T) {
		r := &Rebalance{
			Status: RebalanceStatus{},
		}

		c1 := RebalanceCondition{
			Type:   RebalanceProcessing,
			Status: v1.ConditionTrue,
		}

		c3 := c1
		r.Status.SetCondition(c1)
		c1 = r.Status.Conditions[0]
		time.Sleep(time.Millisecond * time.Duration(1500))
		r.Status.SetCondition(c3)
		c3 = r.Status.Conditions[0]

		assert.Equal(t, 1, len(r.Status.Conditions))
		assert.NotEmpty(t, c3.LastTransitionTime)
		assert.Equal(t, c1.LastTransitionTime, c3.LastTransitionTime)
		assert.NotEmpty(t, c3.LastUpdateTime)
		assert.NotEqual(t, c1.LastUpdateTime, c3.LastUpdateTime)
		assert.Equal(t, c1.Type, c3.Type)
	})
}
