/*
Copyright 2026.

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

package v3beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func TestRolloutMaxUnavailable(t *testing.T) {
	emqx := &EMQX{
		Spec: EMQXSpec{
			ReplicantTemplate: &EMQXReplicantTemplate{
				Spec: EMQXReplicantTemplateSpec{Replicas: ptr.To(int32(10))},
			},
		},
	}
	assert.Equal(t, int32(1), emqx.Spec.NumMaxUnavailableReplicantReplicas())

	emqx.Spec.UpdateStrategy.Replicants = &ReplicantsUpdateStrategy{
		MaxUnavailable: ptr.To(intstr.FromString("50%")),
	}
	assert.Equal(t, int32(5), emqx.Spec.NumMaxUnavailableReplicantReplicas())
}

func TestRolloutMaxSurge(t *testing.T) {
	emqx := &EMQX{
		Spec: EMQXSpec{
			ReplicantTemplate: &EMQXReplicantTemplate{
				Spec: EMQXReplicantTemplateSpec{Replicas: ptr.To(int32(3))},
			},
		},
	}
	assert.Equal(t, int32(0), emqx.Spec.NumMaxSurgeReplicantReplicas())

	emqx.Spec.UpdateStrategy.Replicants = &ReplicantsUpdateStrategy{
		MaxSurge: ptr.To(intstr.FromInt(2)),
	}
	assert.Equal(t, int32(2), emqx.Spec.NumMaxSurgeReplicantReplicas())
}

func TestServiceTemplateIsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		template *ServiceTemplate
		enabled  bool
	}{
		{name: "absent template", template: nil, enabled: true},
		{name: "enabled omitted", template: &ServiceTemplate{}, enabled: true},
		{name: "explicitly enabled", template: &ServiceTemplate{Enabled: ptr.To(true)}, enabled: true},
		{name: "explicitly disabled", template: &ServiceTemplate{Enabled: ptr.To(false)}, enabled: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.enabled, tt.template.IsEnabled())
		})
	}
}
