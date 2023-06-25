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

	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

func TestNextStatusForInit(t *testing.T) {
	emqx := &appsv2alpha2.EMQX{}
	emqxStatusMachine := newEMQXStatusMachine(emqx)
	assert.Equal(t, emqxStatusMachine.initialized, emqxStatusMachine.currentStatus)
	assert.Equal(t, appsv2alpha2.Initialized, emqxStatusMachine.emqx.Status.Conditions[0].Type)
}

func TestNextStatusForCreate(t *testing.T) {
	existedSts := &appsv1.StatefulSet{}
	existedRs := &appsv1.ReplicaSet{}
	emqx := &appsv2alpha2.EMQX{
		Spec: appsv2alpha2.EMQXSpec{
			Image: "emqx/emqx:latest",
		},
		Status: appsv2alpha2.EMQXStatus{
			Conditions: []metav1.Condition{
				{
					Type:   appsv2alpha2.Initialized,
					Status: metav1.ConditionTrue,
				},
			},
		},
	}

	emqxStatusMachine := newEMQXStatusMachine(emqx)
	assert.Equal(t, emqxStatusMachine.initialized, emqxStatusMachine.currentStatus)

	emqxStatusMachine.NextStatus(existedSts, existedRs)
	assert.Equal(t, emqxStatusMachine.coreNodesProgressing, emqxStatusMachine.currentStatus)
	assert.Equal(t, appsv2alpha2.CoreNodesProgressing, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
}

func TestNextStatusForCoreUpdate(t *testing.T) {
	t.Run("change image", func(t *testing.T) {
		existedSts := &appsv1.StatefulSet{}
		existedRs := &appsv1.ReplicaSet{}

		emqx := &appsv2alpha2.EMQX{
			Spec: appsv2alpha2.EMQXSpec{
				Image: "emqx/emqx:5.0",
			},
			Status: appsv2alpha2.EMQXStatus{
				CurrentImage: "emqx/emqx:latest",
				Conditions: []metav1.Condition{
					{
						Type:   appsv2alpha2.CoreNodesProgressing,
						Status: metav1.ConditionTrue,
					},
				},
			},
		}

		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.coreNodesProgressing, emqxStatusMachine.currentStatus)

		emqxStatusMachine.NextStatus(existedSts, existedRs)
		assert.Equal(t, emqxStatusMachine.coreNodesProgressing, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha2.CoreNodesProgressing, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})

	t.Run("next status", func(t *testing.T) {
		existedSts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Generation: 1,
			},
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Image: "emqx/emqx:latest"},
						},
					},
				},
			},
			Status: appsv1.StatefulSetStatus{
				Replicas:        1,
				CurrentRevision: "fake",
			},
		}
		existedRs := &appsv1.ReplicaSet{}

		emqx := &appsv2alpha2.EMQX{
			Spec: appsv2alpha2.EMQXSpec{
				Image: "emqx/emqx:latest",
			},
			Status: appsv2alpha2.EMQXStatus{
				CurrentImage: "emqx/emqx:latest",
				Conditions: []metav1.Condition{
					{
						Type:   appsv2alpha2.CoreNodesProgressing,
						Status: metav1.ConditionTrue,
					},
				},
				CoreNodesStatus: appsv2alpha2.EMQXNodesStatus{
					Replicas: 1,
				},
			},
		}

		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.coreNodesProgressing, emqxStatusMachine.currentStatus)

		// statefulSet not update
		emqxStatusMachine.NextStatus(existedSts, existedRs)
		assert.Equal(t, emqxStatusMachine.coreNodesProgressing, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha2.CoreNodesProgressing, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)

		// statefulSet already update, but not ready
		existedSts.Status.ObservedGeneration = 1
		emqxStatusMachine.NextStatus(existedSts, existedRs)
		assert.Equal(t, emqxStatusMachine.coreNodesProgressing, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha2.CoreNodesProgressing, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)

		// statefulSet is ready, but emqx nodes not ready
		existedSts.Status.ReadyReplicas = 1
		existedSts.Status.UpdatedReplicas = 1
		existedSts.Status.UpdateRevision = "fake"
		emqxStatusMachine.NextStatus(existedSts, existedRs)
		assert.Equal(t, emqxStatusMachine.coreNodesProgressing, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha2.CoreNodesProgressing, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)

		// emqx core node is ready
		emqx.Status.CoreNodesStatus.ReadyReplicas = 1
		emqxStatusMachine.NextStatus(existedSts, existedRs)
		assert.Equal(t, emqxStatusMachine.codeNodesReady, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha2.CodeNodesReady, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})
}

func TestNextStatusForCodeNodesReady(t *testing.T) {
	t.Run("change image", func(t *testing.T) {
		existedSts := &appsv1.StatefulSet{}
		existedRs := &appsv1.ReplicaSet{}

		emqx := &appsv2alpha2.EMQX{
			Spec: appsv2alpha2.EMQXSpec{
				Image: "emqx/emqx:5.0",
			},
			Status: appsv2alpha2.EMQXStatus{
				CurrentImage: "emqx/emqx:latest",
				Conditions: []metav1.Condition{
					{
						Type:   appsv2alpha2.CodeNodesReady,
						Status: metav1.ConditionTrue,
					},
				},
			},
		}

		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.codeNodesReady, emqxStatusMachine.currentStatus)

		emqxStatusMachine.NextStatus(existedSts, existedRs)
		assert.Equal(t, emqxStatusMachine.coreNodesProgressing, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha2.CoreNodesProgressing, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})

	t.Run("next status", func(t *testing.T) {
		existedSts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				UID: types.UID("fake"),
			},
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Image: "emqx/emqx:latest"},
						},
					},
				},
			},
			Status: appsv1.StatefulSetStatus{
				Replicas:        1,
				CurrentRevision: "fake",
			},
		}
		existedRs := &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				UID: types.UID("fake"),
			},
			Spec: appsv1.ReplicaSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Image: "emqx/emqx:latest"},
						},
					},
				},
			},
			Status: appsv1.ReplicaSetStatus{
				Replicas: 1,
			},
		}

		emqx := &appsv2alpha2.EMQX{
			Spec: appsv2alpha2.EMQXSpec{
				Image: "emqx/emqx:latest",
			},
			Status: appsv2alpha2.EMQXStatus{
				CurrentImage: "emqx/emqx:latest",
				Conditions: []metav1.Condition{
					{
						Type:   appsv2alpha2.CodeNodesReady,
						Status: metav1.ConditionTrue,
					},
				},
				CoreNodesStatus: appsv2alpha2.EMQXNodesStatus{
					Replicas: 1,
				},
				ReplicantNodesStatus: &appsv2alpha2.EMQXNodesStatus{
					Replicas: 1,
				},
			},
		}

		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.codeNodesReady, emqxStatusMachine.currentStatus)

		// statefulSet and replicaSet not ready
		emqxStatusMachine.NextStatus(existedSts, existedRs)
		assert.Equal(t, emqxStatusMachine.codeNodesReady, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha2.CodeNodesReady, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)

		// statefulSet is ready, but replicaSet not ready
		existedSts.Status.ReadyReplicas = 1
		existedSts.Status.UpdatedReplicas = 1
		existedSts.Status.UpdateRevision = "fake"
		emqxStatusMachine.NextStatus(existedSts, existedRs)
		assert.Equal(t, emqxStatusMachine.codeNodesReady, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha2.CodeNodesReady, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)

		// statefulSet and replicaSet is ready, but emqx nodes not ready
		existedRs.Status.ReadyReplicas = 1
		emqxStatusMachine.NextStatus(existedSts, existedRs)
		assert.Equal(t, emqxStatusMachine.codeNodesReady, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha2.CodeNodesReady, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)

		// emqx nodes is ready
		emqx.Status.CoreNodesStatus.ReadyReplicas = 1
		emqx.Status.ReplicantNodesStatus.ReadyReplicas = 1
		emqxStatusMachine.NextStatus(existedSts, existedRs)
		assert.Equal(t, emqxStatusMachine.ready, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha2.Ready, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})

}

func TestNextStatusForCoreReady(t *testing.T) {
	t.Run("change image", func(t *testing.T) {
		existedSts := &appsv1.StatefulSet{}
		existedRs := &appsv1.ReplicaSet{}

		emqx := &appsv2alpha2.EMQX{
			Spec: appsv2alpha2.EMQXSpec{
				Image: "emqx/emqx:5.0",
			},
			Status: appsv2alpha2.EMQXStatus{
				CurrentImage: "emqx/emqx:latest",
				Conditions: []metav1.Condition{
					{
						Type:   appsv2alpha2.Ready,
						Status: metav1.ConditionTrue,
					},
				},
			},
		}

		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.ready, emqxStatusMachine.currentStatus)

		emqxStatusMachine.NextStatus(existedSts, existedRs)
		assert.Equal(t, emqxStatusMachine.coreNodesProgressing, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha2.CoreNodesProgressing, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})

	t.Run("replicant nodes not ready", func(t *testing.T) {
		existedSts := &appsv1.StatefulSet{}
		existedRs := &appsv1.ReplicaSet{}
		emqx := &appsv2alpha2.EMQX{
			Spec: appsv2alpha2.EMQXSpec{
				Image: "emqx/emqx:latest",
				ReplicantTemplate: &appsv2alpha2.EMQXReplicantTemplate{
					Spec: appsv2alpha2.EMQXReplicantTemplateSpec{
						Replicas: pointer.Int32(1),
					},
				},
			},
			Status: appsv2alpha2.EMQXStatus{
				CurrentImage: "emqx/emqx:latest",
				Conditions: []metav1.Condition{
					{
						Type:   appsv2alpha2.Ready,
						Status: metav1.ConditionTrue,
					},
					{
						Type:   appsv2alpha2.CodeNodesReady,
						Status: metav1.ConditionTrue,
					},
				},
				ReplicantNodesStatus: &appsv2alpha2.EMQXNodesStatus{
					Replicas:      1,
					ReadyReplicas: 0,
				},
			},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.ready, emqxStatusMachine.currentStatus)

		emqxStatusMachine.NextStatus(existedSts, existedRs)
		assert.Equal(t, emqxStatusMachine.codeNodesReady, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha2.CodeNodesReady, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})

	t.Run("replicant nodes not ready", func(t *testing.T) {
		existedSts := &appsv1.StatefulSet{}
		existedRs := &appsv1.ReplicaSet{}
		emqx := &appsv2alpha2.EMQX{
			Spec: appsv2alpha2.EMQXSpec{
				Image: "emqx/emqx:latest",
			},
			Status: appsv2alpha2.EMQXStatus{
				CurrentImage: "emqx/emqx:latest",
				Conditions: []metav1.Condition{
					{
						Type:   appsv2alpha2.Ready,
						Status: metav1.ConditionTrue,
					},
					{
						Type:   appsv2alpha2.CodeNodesReady,
						Status: metav1.ConditionTrue,
					},
					{
						Type:   appsv2alpha2.CoreNodesProgressing,
						Status: metav1.ConditionTrue,
					},
				},
				CoreNodesStatus: appsv2alpha2.EMQXNodesStatus{
					Replicas:      1,
					ReadyReplicas: 0,
				},
				ReplicantNodesStatus: &appsv2alpha2.EMQXNodesStatus{
					Replicas:      1,
					ReadyReplicas: 0,
				},
			},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.ready, emqxStatusMachine.currentStatus)

		emqxStatusMachine.NextStatus(existedSts, existedRs)
		assert.Equal(t, emqxStatusMachine.coreNodesProgressing, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha2.CoreNodesProgressing, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})

	t.Run("next status", func(t *testing.T) {
		existedSts := &appsv1.StatefulSet{}
		existedRs := &appsv1.ReplicaSet{}
		emqx := &appsv2alpha2.EMQX{
			Spec: appsv2alpha2.EMQXSpec{
				Image: "emqx/emqx:latest",
			},
			Status: appsv2alpha2.EMQXStatus{
				CurrentImage: "emqx/emqx:latest",
				Conditions: []metav1.Condition{
					{
						Type:   appsv2alpha2.Ready,
						Status: metav1.ConditionTrue,
					},
				},
			},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.ready, emqxStatusMachine.currentStatus)

		emqxStatusMachine.NextStatus(existedSts, existedRs)
		assert.Equal(t, emqxStatusMachine.ready, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha2.Ready, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})
}
