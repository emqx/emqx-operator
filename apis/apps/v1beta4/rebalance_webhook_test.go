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

	assert.NoError(t, rebalance.ValidateCreate())

	r := rebalance.DeepCopy()
	r.Spec.RebalanceStrategy.RelConnThreshold = "test"
	assert.ErrorContains(t, r.ValidateCreate(), "invalid syntax")

	r = rebalance.DeepCopy()
	r.Spec.RebalanceStrategy.RelSessThreshold = "test-0"
	assert.ErrorContains(t, r.ValidateCreate(), "invalid syntax")

	r = rebalance.DeepCopy()
	r.Spec.RebalanceStrategy.RelSessThreshold = "1.2"
	assert.NoError(t, r.ValidateCreate())

	r = rebalance.DeepCopy()
	r.Spec.RebalanceStrategy.RelConnThreshold = "1.2"
	assert.NoError(t, r.ValidateCreate())

	r = rebalance.DeepCopy()
	r.Spec.RebalanceStrategy.RelConnThreshold = "1.2"
	r.Spec.RebalanceStrategy.RelSessThreshold = "test"
	assert.ErrorContains(t, r.ValidateCreate(), "invalid syntax")

	r = rebalance.DeepCopy()
	assert.NoError(t, r.ValidateCreate())
}

func TestRebalanceValidateUpdate(t *testing.T) {
	rebalance := Rebalance{
		ObjectMeta: v1.ObjectMeta{
			Annotations: map[string]string{
				"test": "rebalance",
			},
		},
		Spec: RebalanceSpec{
			InstanceName: "test",
			RebalanceStrategy: RebalanceStrategy{
				ConnEvictRate: 10,
			},
		},
	}

	old := rebalance.DeepCopy()
	old.Spec.InstanceName = "test-0"
	assert.ErrorContains(t, rebalance.ValidateUpdate(old), "the Rebalance don't allow update")

	old = rebalance.DeepCopy()
	old.Annotations = map[string]string{
		"test":   "rebalance",
		"test-0": "rebalance",
	}
	assert.NoError(t, rebalance.ValidateUpdate(old))

	old = rebalance.DeepCopy()
	old.Annotations = map[string]string{
		"test":   "rebalance",
		"test-0": "rebalance",
	}
	old.Spec.InstanceName = "test-0"
	assert.ErrorContains(t, rebalance.ValidateUpdate(old), "the Rebalance don't allow update")
}
