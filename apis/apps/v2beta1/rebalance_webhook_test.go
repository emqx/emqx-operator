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

package v2beta1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRebalanceValidateCreate(t *testing.T) {
	rebalance := Rebalance{
		Spec: RebalanceSpec{
			InstanceName: "test",
			RebalanceStrategy: RebalanceStrategy{
				ConnEvictRate: 10,
			},
		},
	}
	_, err := rebalance.ValidateCreate()
	assert.NoError(t, err)

	t.Run("valid RelConnThreshold must be float", func(t *testing.T) {
		var err error
		r := rebalance.DeepCopy()
		r.Spec.RebalanceStrategy.RelConnThreshold = "test"
		_, err = r.ValidateCreate()
		assert.ErrorContains(t, err, "must be float64")

		r = rebalance.DeepCopy()
		r.Spec.RebalanceStrategy.RelConnThreshold = "1.2"
		_, err = r.ValidateCreate()
		assert.NoError(t, err)
	})

	t.Run("valid RelSessThreshold must be float", func(t *testing.T) {
		var err error
		r := rebalance.DeepCopy()
		r.Spec.RebalanceStrategy.RelSessThreshold = "test-0"
		_, err = r.ValidateCreate()
		assert.ErrorContains(t, err, "must be float64")

		r = rebalance.DeepCopy()
		r.Spec.RebalanceStrategy.RelSessThreshold = "1.2"
		_, err = r.ValidateCreate()
		assert.NoError(t, err)
	})

	t.Run("valid RelSessThreshold and RelConnThreshold must be float64", func(t *testing.T) {
		var err error
		r := rebalance.DeepCopy()
		r.Spec.RebalanceStrategy.RelConnThreshold = "1.2"
		r.Spec.RebalanceStrategy.RelSessThreshold = "test"
		_, err = r.ValidateCreate()
		assert.ErrorContains(t, err, "must be float64")

		r.Spec.RebalanceStrategy.RelConnThreshold = "test"
		r.Spec.RebalanceStrategy.RelSessThreshold = "1.2"
		_, err = r.ValidateCreate()
		assert.ErrorContains(t, err, "must be float64")

		r.Spec.RebalanceStrategy.RelConnThreshold = "1.2"
		r.Spec.RebalanceStrategy.RelSessThreshold = "1.2"
		_, err = r.ValidateCreate()
		assert.NoError(t, err)
	})
}

func TestRebalanceValidateUpdate(t *testing.T) {
	rebalance := Rebalance{
		Spec: RebalanceSpec{
			InstanceName: "test",
			RebalanceStrategy: RebalanceStrategy{
				ConnEvictRate: 10,
			},
		},
	}

	t.Run("valid update instanceName ", func(t *testing.T) {
		var err error
		old := rebalance.DeepCopy()
		_, err = rebalance.ValidateUpdate(old)
		assert.NoError(t, err)

		old = rebalance.DeepCopy()
		old.SetGeneration(1)
		_, err = rebalance.ValidateUpdate(old)
		assert.ErrorContains(t, err, "the Rebalance spec don't allow update")
	})

	t.Run("valid other field instead of spec ", func(t *testing.T) {
		var err error
		old := rebalance.DeepCopy()
		old.Finalizers = []string{"test", "test-0"}
		_, err = rebalance.ValidateUpdate(old)
		assert.NoError(t, err)

		old = rebalance.DeepCopy()
		old.DeletionTimestamp = &v1.Time{Time: time.Now()}
		_, err = rebalance.ValidateUpdate(old)
		assert.NoError(t, err)
	})
}
