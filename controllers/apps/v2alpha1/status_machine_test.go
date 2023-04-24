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

	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCheckNodeCount(t *testing.T) {
	replicas := int32(1)

	t.Run("no replicant nodes", func(t *testing.T) {
		emqx := &appsv2alpha1.EMQX{}
		emqx.Spec.CoreTemplate.Spec.Replicas = &replicas

		emqxNodes := []appsv2alpha1.EMQXNode{
			{
				Role:       "core",
				NodeStatus: "running",
			},
			{
				Role:       "fake role",
				NodeStatus: "stop",
			},
		}

		emqxStatusMachine := newEMQXStatusMachine(emqx)
		emqxStatusMachine.CheckNodeCount(emqxNodes)
		assert.Equal(t, emqxStatusMachine.GetEMQX().Status.CoreNodeReplicas, int32(1))
		assert.Equal(t, emqxStatusMachine.GetEMQX().Status.CoreNodeReadyReplicas, int32(1))
		assert.Equal(t, emqxStatusMachine.GetEMQX().Status.ReplicantNodeReplicas, int32(0))
		assert.Equal(t, emqxStatusMachine.GetEMQX().Status.ReplicantNodeReadyReplicas, int32(0))
	})

	t.Run("have replicant nodes", func(t *testing.T) {
		emqx := &appsv2alpha1.EMQX{}
		emqx.Spec.CoreTemplate.Spec.Replicas = &replicas
		emqx.Spec.ReplicantTemplate.Spec.Replicas = &replicas

		emqxNodes := []appsv2alpha1.EMQXNode{
			{
				Role:       "core",
				NodeStatus: "running",
			},
			{
				Role:       "replicant",
				NodeStatus: "running",
			},
			{
				Role:       "fake role",
				NodeStatus: "stop",
			},
		}

		emqxStatusMachine := newEMQXStatusMachine(emqx)
		emqxStatusMachine.CheckNodeCount(emqxNodes)
		assert.Equal(t, emqxStatusMachine.GetEMQX().Status.CoreNodeReplicas, int32(1))
		assert.Equal(t, emqxStatusMachine.GetEMQX().Status.CoreNodeReadyReplicas, int32(1))
		assert.Equal(t, emqxStatusMachine.GetEMQX().Status.ReplicantNodeReplicas, int32(1))
		assert.Equal(t, emqxStatusMachine.GetEMQX().Status.ReplicantNodeReadyReplicas, int32(1))
	})
}

func TestNextStatusForInit(t *testing.T) {
	existedSts := &appsv1.StatefulSet{}
	existedDeploy := &appsv1.Deployment{}
	emqx := &appsv2alpha1.EMQX{}
	emqxStatusMachine := newEMQXStatusMachine(emqx)
	assert.Equal(t, emqxStatusMachine.init, emqxStatusMachine.currentStatus)

	emqxStatusMachine.NextStatus(existedSts, existedDeploy)
	assert.Equal(t, emqxStatusMachine.creating, emqxStatusMachine.currentStatus)
	assert.Equal(t, appsv2alpha1.ClusterCreating, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
}

func TestNextStatusForCreate(t *testing.T) {
	existedSts := &appsv1.StatefulSet{}
	existedDeploy := &appsv1.Deployment{}
	emqx := &appsv2alpha1.EMQX{
		Spec: appsv2alpha1.EMQXSpec{
			Image: "emqx/emqx:latest",
		},
		Status: appsv2alpha1.EMQXStatus{
			Conditions: []appsv2alpha1.Condition{
				{
					Type:   appsv2alpha1.ClusterCreating,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	emqxStatusMachine := newEMQXStatusMachine(emqx)
	assert.Equal(t, emqxStatusMachine.creating, emqxStatusMachine.currentStatus)

	emqxStatusMachine.NextStatus(existedSts, existedDeploy)
	assert.Equal(t, emqxStatusMachine.coreUpdating, emqxStatusMachine.currentStatus)
	assert.Equal(t, appsv2alpha1.ClusterCoreUpdating, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
}

func TestNextStatusForCoreUpdate(t *testing.T) {
	t.Run("change image", func(t *testing.T) {
		existedSts := &appsv1.StatefulSet{}
		existedDeploy := &appsv1.Deployment{}

		emqx := &appsv2alpha1.EMQX{
			Spec: appsv2alpha1.EMQXSpec{
				Image: "emqx/emqx:5.0",
			},
			Status: appsv2alpha1.EMQXStatus{
				CurrentImage: "emqx/emqx:latest",
				Conditions: []appsv2alpha1.Condition{
					{
						Type:   appsv2alpha1.ClusterCoreUpdating,
						Status: corev1.ConditionTrue,
					},
				},
			},
		}

		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.coreUpdating, emqxStatusMachine.currentStatus)

		emqxStatusMachine.NextStatus(existedSts, existedDeploy)
		assert.Equal(t, emqxStatusMachine.coreUpdating, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha1.ClusterCoreUpdating, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
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
		existedDeploy := &appsv1.Deployment{}

		emqx := &appsv2alpha1.EMQX{
			Spec: appsv2alpha1.EMQXSpec{
				Image: "emqx/emqx:latest",
			},
			Status: appsv2alpha1.EMQXStatus{
				CurrentImage: "emqx/emqx:latest",
				Conditions: []appsv2alpha1.Condition{
					{
						Type:   appsv2alpha1.ClusterCoreUpdating,
						Status: corev1.ConditionTrue,
					},
				},
				CoreNodeReplicas: 1,
			},
		}

		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.coreUpdating, emqxStatusMachine.currentStatus)

		// statefulSet not update
		emqxStatusMachine.NextStatus(existedSts, existedDeploy)
		assert.Equal(t, emqxStatusMachine.coreUpdating, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha1.ClusterCoreUpdating, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)

		// statefulSet already update, but not ready
		existedSts.Status.ObservedGeneration = 1
		emqxStatusMachine.NextStatus(existedSts, existedDeploy)
		assert.Equal(t, emqxStatusMachine.coreUpdating, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha1.ClusterCoreUpdating, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)

		// statefulSet is ready, but emqx nodes not ready
		existedSts.Status.ReadyReplicas = 1
		existedSts.Status.UpdatedReplicas = 1
		existedSts.Status.UpdateRevision = "fake"
		emqxStatusMachine.NextStatus(existedSts, existedDeploy)
		assert.Equal(t, emqxStatusMachine.coreUpdating, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha1.ClusterCoreUpdating, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)

		// emqx core node is ready
		emqx.Status.CoreNodeReadyReplicas = 1
		emqxStatusMachine.NextStatus(existedSts, existedDeploy)
		assert.Equal(t, emqxStatusMachine.coreReady, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha1.ClusterCoreReady, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})
}

func TestNextStatusForCoreReady(t *testing.T) {
	t.Run("change image", func(t *testing.T) {
		existedSts := &appsv1.StatefulSet{}
		existedDeploy := &appsv1.Deployment{}

		emqx := &appsv2alpha1.EMQX{
			Spec: appsv2alpha1.EMQXSpec{
				Image: "emqx/emqx:5.0",
			},
			Status: appsv2alpha1.EMQXStatus{
				CurrentImage: "emqx/emqx:latest",
				Conditions: []appsv2alpha1.Condition{
					{
						Type:   appsv2alpha1.ClusterCoreReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		}

		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.coreReady, emqxStatusMachine.currentStatus)

		emqxStatusMachine.NextStatus(existedSts, existedDeploy)
		assert.Equal(t, emqxStatusMachine.coreUpdating, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha1.ClusterCoreUpdating, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})

	t.Run("next status", func(t *testing.T) {
		existedSts := &appsv1.StatefulSet{
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
		existedDeploy := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Image: "emqx/emqx:latest"},
						},
					},
				},
			},
			Status: appsv1.DeploymentStatus{
				Replicas: 1,
			},
		}

		emqx := &appsv2alpha1.EMQX{
			Spec: appsv2alpha1.EMQXSpec{
				Image: "emqx/emqx:latest",
			},
			Status: appsv2alpha1.EMQXStatus{
				CurrentImage: "emqx/emqx:latest",
				Conditions: []appsv2alpha1.Condition{
					{
						Type:   appsv2alpha1.ClusterCoreReady,
						Status: corev1.ConditionTrue,
					},
				},
				CoreNodeReplicas:      1,
				ReplicantNodeReplicas: 1,
			},
		}

		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.coreReady, emqxStatusMachine.currentStatus)

		// statefulSet and deployment not ready
		emqxStatusMachine.NextStatus(existedSts, existedDeploy)
		assert.Equal(t, emqxStatusMachine.coreReady, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha1.ClusterCoreReady, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)

		// statefulSet is ready, but deployment not ready
		existedSts.UID = "fake"
		existedSts.Status.ReadyReplicas = 1
		existedSts.Status.UpdatedReplicas = 1
		existedSts.Status.UpdateRevision = "fake"
		emqxStatusMachine.NextStatus(existedSts, existedDeploy)
		assert.Equal(t, emqxStatusMachine.coreReady, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha1.ClusterCoreReady, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)

		// statefulSet and deployment is ready, but emqx nodes not ready
		existedDeploy.UID = "fake"
		existedDeploy.Status.UpdatedReplicas = 1
		existedDeploy.Status.ReadyReplicas = 1
		emqxStatusMachine.NextStatus(existedSts, existedDeploy)
		assert.Equal(t, emqxStatusMachine.coreReady, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha1.ClusterCoreReady, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)

		// emqx nodes is ready
		emqx.Status.CoreNodeReadyReplicas = 1
		emqx.Status.ReplicantNodeReadyReplicas = 1
		emqxStatusMachine.NextStatus(existedSts, existedDeploy)
		assert.Equal(t, emqxStatusMachine.running, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha1.ClusterRunning, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})

}

func TestNextStatusForCoreRunning(t *testing.T) {
	t.Run("change image", func(t *testing.T) {
		existedSts := &appsv1.StatefulSet{}
		existedDeploy := &appsv1.Deployment{}

		emqx := &appsv2alpha1.EMQX{
			Spec: appsv2alpha1.EMQXSpec{
				Image: "emqx/emqx:5.0",
			},
			Status: appsv2alpha1.EMQXStatus{
				CurrentImage: "emqx/emqx:latest",
				Conditions: []appsv2alpha1.Condition{
					{
						Type:   appsv2alpha1.ClusterRunning,
						Status: corev1.ConditionTrue,
					},
				},
			},
		}

		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.running, emqxStatusMachine.currentStatus)

		emqxStatusMachine.NextStatus(existedSts, existedDeploy)
		assert.Equal(t, emqxStatusMachine.coreUpdating, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha1.ClusterCoreUpdating, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})

	t.Run("replicant nodes not ready", func(t *testing.T) {
		existedSts := &appsv1.StatefulSet{}
		existedDeploy := &appsv1.Deployment{}
		emqx := &appsv2alpha1.EMQX{
			Spec: appsv2alpha1.EMQXSpec{
				Image: "emqx/emqx:latest",
			},
			Status: appsv2alpha1.EMQXStatus{
				CurrentImage: "emqx/emqx:latest",
				Conditions: []appsv2alpha1.Condition{
					{
						Type:   appsv2alpha1.ClusterRunning,
						Status: corev1.ConditionTrue,
					},
					{
						Type:   appsv2alpha1.ClusterCoreReady,
						Status: corev1.ConditionTrue,
					},
				},
				ReplicantNodeReplicas:      1,
				ReplicantNodeReadyReplicas: 0,
			},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.running, emqxStatusMachine.currentStatus)

		emqxStatusMachine.NextStatus(existedSts, existedDeploy)
		assert.Equal(t, emqxStatusMachine.coreReady, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha1.ClusterCoreReady, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})

	t.Run("replicant nodes not ready", func(t *testing.T) {
		existedSts := &appsv1.StatefulSet{}
		existedDeploy := &appsv1.Deployment{}
		emqx := &appsv2alpha1.EMQX{
			Spec: appsv2alpha1.EMQXSpec{
				Image: "emqx/emqx:latest",
			},
			Status: appsv2alpha1.EMQXStatus{
				CurrentImage: "emqx/emqx:latest",
				Conditions: []appsv2alpha1.Condition{
					{
						Type:   appsv2alpha1.ClusterRunning,
						Status: corev1.ConditionTrue,
					},
					{
						Type:   appsv2alpha1.ClusterCoreReady,
						Status: corev1.ConditionTrue,
					},
					{
						Type:   appsv2alpha1.ClusterCoreUpdating,
						Status: corev1.ConditionTrue,
					},
				},
				ReplicantNodeReplicas:      1,
				ReplicantNodeReadyReplicas: 0,
				CoreNodeReplicas:           1,
				CoreNodeReadyReplicas:      0,
			},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.running, emqxStatusMachine.currentStatus)

		emqxStatusMachine.NextStatus(existedSts, existedDeploy)
		assert.Equal(t, emqxStatusMachine.coreUpdating, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha1.ClusterCoreUpdating, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})

	t.Run("next status", func(t *testing.T) {
		existedSts := &appsv1.StatefulSet{}
		existedDeploy := &appsv1.Deployment{}
		emqx := &appsv2alpha1.EMQX{
			Spec: appsv2alpha1.EMQXSpec{
				Image: "emqx/emqx:latest",
			},
			Status: appsv2alpha1.EMQXStatus{
				CurrentImage: "emqx/emqx:latest",
				Conditions: []appsv2alpha1.Condition{
					{
						Type:   appsv2alpha1.ClusterRunning,
						Status: corev1.ConditionTrue,
					},
				},
			},
		}
		emqxStatusMachine := newEMQXStatusMachine(emqx)
		assert.Equal(t, emqxStatusMachine.running, emqxStatusMachine.currentStatus)

		emqxStatusMachine.NextStatus(existedSts, existedDeploy)
		assert.Equal(t, emqxStatusMachine.running, emqxStatusMachine.currentStatus)
		assert.Equal(t, appsv2alpha1.ClusterRunning, emqxStatusMachine.GetEMQX().Status.Conditions[0].Type)
	})
}
