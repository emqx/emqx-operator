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

	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

var instance = &appsv2beta1.EMQX{
	Spec: appsv2beta1.EMQXSpec{
		CoreTemplate: appsv2beta1.EMQXCoreTemplate{
			Spec: appsv2beta1.EMQXCoreTemplateSpec{
				EMQXReplicantTemplateSpec: appsv2beta1.EMQXReplicantTemplateSpec{
					Replicas: pointer.Int32Ptr(3),
				},
			},
		},
		ReplicantTemplate: &appsv2beta1.EMQXReplicantTemplate{
			Spec: appsv2beta1.EMQXReplicantTemplateSpec{
				Replicas: pointer.Int32Ptr(3),
			},
		},
	},
	Status: appsv2beta1.EMQXStatus{
		CoreNodesStatus: appsv2beta1.EMQXNodesStatus{
			Replicas: 3,
		},
		ReplicantNodesStatus: &appsv2beta1.EMQXNodesStatus{
			Replicas: 3,
		},
	},
}

func TestNewStatusMachine(t *testing.T) {
	t.Run("initialized", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.initialized, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2beta1.Initialized, emqxStatusMachine.emqx.Status.Conditions[0].Type)
	})

	t.Run("coreNodesProgressing", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Status.Conditions = []metav1.Condition{
			{Type: appsv2beta1.CoreNodesProgressing, Status: metav1.ConditionTrue},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.coreNodesProgressing, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2beta1.CoreNodesProgressing, emqxStatusMachine.emqx.Status.Conditions[0].Type)
	})

	t.Run("coreNodesReady", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Status.Conditions = []metav1.Condition{
			{Type: appsv2beta1.CoreNodesReady, Status: metav1.ConditionTrue},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.coreNodesReady, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2beta1.CoreNodesReady, emqxStatusMachine.emqx.Status.Conditions[0].Type)
	})

	t.Run("replicantNodesProgressing", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Status.Conditions = []metav1.Condition{
			{Type: appsv2beta1.ReplicantNodesProgressing, Status: metav1.ConditionTrue},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.replicantNodesProgressing, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2beta1.ReplicantNodesProgressing, emqxStatusMachine.emqx.Status.Conditions[0].Type)
	})

	t.Run("replicantNodesReady", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Status.Conditions = []metav1.Condition{
			{Type: appsv2beta1.ReplicantNodesReady, Status: metav1.ConditionTrue},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.replicantNodesReady, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2beta1.ReplicantNodesReady, emqxStatusMachine.emqx.Status.Conditions[0].Type)
	})

	t.Run("available", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Status.Conditions = []metav1.Condition{
			{Type: appsv2beta1.Available, Status: metav1.ConditionTrue},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.available, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2beta1.Available, emqxStatusMachine.emqx.Status.Conditions[0].Type)
	})

	t.Run("ready", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Status.Conditions = []metav1.Condition{
			{Type: appsv2beta1.Ready, Status: metav1.ConditionTrue},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.ready, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2beta1.Ready, emqxStatusMachine.emqx.Status.Conditions[0].Type)
	})
}

func TestNextStatusForInit(t *testing.T) {
	emqx := instance.DeepCopy()
	emqx.Status.Conditions = []metav1.Condition{
		{
			Type:   appsv2beta1.Initialized,
			Status: metav1.ConditionTrue,
		},
	}

	emqxStatusMachine := newEMQXStatusMachine(emqx)
	emqxStatusMachine.NextStatus()
	assert.Equal(t, emqxStatusMachine.coreNodesProgressing, emqxStatusMachine.currentStatus)
	assert.Equal(t, appsv2beta1.CoreNodesProgressing, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
}

func TestNextStatusForCoreNodeProgressing(t *testing.T) {
	t.Run("still status when core nodes not ready", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Status.Conditions = []metav1.Condition{
			{
				Type:   appsv2beta1.CoreNodesProgressing,
				Status: metav1.ConditionTrue,
			},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)

		emqx.Status.CoreNodesStatus.UpdateReplicas = 0
		emqxStatusMachine.NextStatus()

		assert.Equal(t, emqxStatusMachine.coreNodesProgressing, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2beta1.CoreNodesProgressing, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})

	t.Run("next status", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Status.Conditions = []metav1.Condition{
			{
				Type:   appsv2beta1.CoreNodesProgressing,
				Status: metav1.ConditionTrue,
			},
		}

		emqxStatusMachine := newEMQXStatusMachine(emqx)
		emqx.Status.CoreNodesStatus.UpdateReplicas = emqx.Status.CoreNodesStatus.Replicas

		emqxStatusMachine.NextStatus()
		assert.Equal(t, emqxStatusMachine.coreNodesReady, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2beta1.CoreNodesReady, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})
}

func TestNextStatusForCodeNodesReady(t *testing.T) {
	t.Run("next status when replicant template is not nil", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Status.Conditions = []metav1.Condition{
			{
				Type:   appsv2beta1.CoreNodesReady,
				Status: metav1.ConditionTrue,
			},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)
		emqxStatusMachine.NextStatus()
		assert.Equal(t, emqxStatusMachine.replicantNodesProgressing, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2beta1.ReplicantNodesProgressing, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})

	t.Run("next status when replicant template is nil", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Status.Conditions = []metav1.Condition{
			{
				Type:   appsv2beta1.CoreNodesReady,
				Status: metav1.ConditionTrue,
			},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)

		emqx.Spec.ReplicantTemplate = nil
		emqxStatusMachine.NextStatus()
		assert.Equal(t, emqxStatusMachine.available, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2beta1.Available, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})
}

func TestNextStatusForReplicantNodeProgressing(t *testing.T) {
	t.Run("replicant template is nil, need roll back to initialized next status", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Spec.ReplicantTemplate = nil
		emqx.Status.Conditions = []metav1.Condition{
			{
				Type:   appsv2beta1.ReplicantNodesProgressing,
				Status: metav1.ConditionTrue,
			},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)

		emqxStatusMachine.NextStatus()
		assert.Equal(t, emqxStatusMachine.coreNodesProgressing, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2beta1.CoreNodesProgressing, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})

	t.Run("still status when replicant nodes not ready", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Status.Conditions = []metav1.Condition{
			{
				Type:   appsv2beta1.ReplicantNodesProgressing,
				Status: metav1.ConditionTrue,
			},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)
		emqx.Status.ReplicantNodesStatus.UpdateReplicas = 0

		emqxStatusMachine.NextStatus()
		assert.Equal(t, emqxStatusMachine.replicantNodesProgressing, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2beta1.ReplicantNodesProgressing, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})

	t.Run("next status", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Status.Conditions = []metav1.Condition{
			{
				Type:   appsv2beta1.ReplicantNodesProgressing,
				Status: metav1.ConditionTrue,
			},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)
		emqx.Status.ReplicantNodesStatus.UpdateReplicas = emqx.Status.ReplicantNodesStatus.Replicas

		emqxStatusMachine.NextStatus()
		assert.Equal(t, emqxStatusMachine.replicantNodesReady, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2beta1.ReplicantNodesReady, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})
}

func TestNextStatusForReplicantNodesReady(t *testing.T) {
	t.Run("replicant template is nil, need roll back to initialized next status", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Spec.ReplicantTemplate = nil
		emqx.Status.Conditions = []metav1.Condition{
			{Type: appsv2beta1.ReplicantNodesReady, Status: metav1.ConditionTrue},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)

		emqxStatusMachine.NextStatus()
		assert.Equal(t, emqxStatusMachine.coreNodesProgressing, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2beta1.CoreNodesProgressing, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})

	t.Run("next status", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Status.Conditions = []metav1.Condition{
			{
				Type:   appsv2beta1.ReplicantNodesReady,
				Status: metav1.ConditionTrue,
			},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)
		emqx.Status.ReplicantNodesStatus.UpdateReplicas = emqx.Status.ReplicantNodesStatus.Replicas

		emqxStatusMachine.NextStatus()
		assert.Equal(t, emqxStatusMachine.available, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2beta1.Available, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})
}

func TestNextStatusForAvailable(t *testing.T) {
	t.Run("still status when core nodes ready replicas not equal replicas", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Status.Conditions = []metav1.Condition{
			{
				Type:   appsv2beta1.Available,
				Status: metav1.ConditionTrue,
			},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)
		emqx.Status.CoreNodesStatus.ReadyReplicas = 5

		emqxStatusMachine.NextStatus()
		assert.Equal(t, emqxStatusMachine.available, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2beta1.Available, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})

	t.Run("still status when replicant nodes ready replicas not equal replicas", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Status.Conditions = []metav1.Condition{
			{
				Type:   appsv2beta1.Available,
				Status: metav1.ConditionTrue,
			},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)
		emqx.Status.ReplicantNodesStatus.ReadyReplicas = 5

		emqxStatusMachine.NextStatus()
		assert.Equal(t, emqxStatusMachine.available, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2beta1.Available, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})

	t.Run("next status", func(t *testing.T) {
		emqx := instance.DeepCopy()
		emqx.Status.Conditions = []metav1.Condition{
			{
				Type:   appsv2beta1.Available,
				Status: metav1.ConditionTrue,
			},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)
		emqx.Status.CoreNodesStatus.ReadyReplicas = emqx.Status.CoreNodesStatus.Replicas
		emqx.Status.ReplicantNodesStatus.ReadyReplicas = emqx.Status.ReplicantNodesStatus.Replicas

		emqxStatusMachine.NextStatus()
		assert.Equal(t, emqxStatusMachine.ready, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2beta1.Ready, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})
}
