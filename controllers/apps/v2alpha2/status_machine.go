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
	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type status interface {
	nextStatus(*appsv1.StatefulSet, *appsv1.Deployment)
}

type emqxStatusMachine struct {
	emqx *appsv2alpha2.EMQX

	init         status
	creating     status
	coreUpdating status
	coreReady    status
	running      status

	currentStatus status
}

func newEMQXStatusMachine(emqx *appsv2alpha2.EMQX) *emqxStatusMachine {
	emqxStatusMachine := &emqxStatusMachine{
		emqx: emqx,
	}

	initStatus := &initStatus{emqxStatusMachine: emqxStatusMachine}
	createStatus := &createStatus{emqxStatusMachine: emqxStatusMachine}
	coreUpdateStatus := &coreUpdateStatus{emqxStatusMachine: emqxStatusMachine}
	coreReadyStatus := &coreReadyStatus{emqxStatusMachine: emqxStatusMachine}
	runningStatus := &runningStatus{emqxStatusMachine: emqxStatusMachine}

	emqxStatusMachine.init = initStatus
	emqxStatusMachine.creating = createStatus
	emqxStatusMachine.coreUpdating = coreUpdateStatus
	emqxStatusMachine.coreReady = coreReadyStatus
	emqxStatusMachine.running = runningStatus
	emqxStatusMachine.setCurrentStatus(emqx)

	return emqxStatusMachine
}

func (s *emqxStatusMachine) setCurrentStatus(emqx *appsv2alpha2.EMQX) {
	if emqx.Status.Conditions == nil {
		s.currentStatus = s.init
	}

	if emqx.Status.IsCreating() {
		s.currentStatus = s.creating
	}

	if emqx.Status.IsCoreNodesUpdating() {
		s.currentStatus = s.coreUpdating
	}

	if emqx.Status.IsCoreNodesReady() {
		s.currentStatus = s.coreReady
	}

	if emqx.Status.IsRunning() {
		s.currentStatus = s.running
	}
}

func (s *emqxStatusMachine) UpdateNodeCount(emqxNodes []appsv2alpha2.EMQXNode) {
	s.emqx.Status.CoreNodeReplicas = *s.emqx.Spec.CoreTemplate.Spec.Replicas
	s.emqx.Status.ReplicantNodeReplicas = *s.emqx.Spec.ReplicantTemplate.Spec.Replicas

	s.emqx.Status.CoreNodeReadyReplicas = int32(0)
	s.emqx.Status.ReplicantNodeReadyReplicas = int32(0)

	if emqxNodes != nil {
		s.emqx.Status.SetEMQXNodes(emqxNodes)

		for _, node := range emqxNodes {
			if node.NodeStatus == "running" {
				if node.Role == "core" {
					s.emqx.Status.CoreNodeReadyReplicas++
				}
				if node.Role == "replicant" {
					s.emqx.Status.ReplicantNodeReadyReplicas++
				}
			}
		}
	}
}

func (s *emqxStatusMachine) NextStatus(existedSts *appsv1.StatefulSet, existedDeploy *appsv1.Deployment) {
	s.currentStatus.nextStatus(existedSts, existedDeploy)
}

func (s *emqxStatusMachine) GetEMQX() *appsv2alpha2.EMQX {
	return s.emqx
}

type initStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *initStatus) nextStatus(_ *appsv1.StatefulSet, _ *appsv1.Deployment) {
	condition := appsv2alpha2.NewCondition(
		appsv2alpha2.ClusterCreating,
		corev1.ConditionTrue,
		"ClusterCreating",
		"Creating EMQX cluster",
	)
	s.emqxStatusMachine.emqx.Status.SetCondition(*condition)
	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
}

type createStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *createStatus) nextStatus(_ *appsv1.StatefulSet, _ *appsv1.Deployment) {
	s.emqxStatusMachine.emqx.Status.CurrentImage = s.emqxStatusMachine.emqx.Spec.Image
	condition := appsv2alpha2.NewCondition(
		appsv2alpha2.ClusterCoreUpdating,
		corev1.ConditionTrue,
		"ClusterCoreUpdating",
		"Updating core nodes in cluster",
	)
	s.emqxStatusMachine.emqx.Status.SetCondition(*condition)
	s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2alpha2.ClusterCoreReady)
	s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2alpha2.ClusterRunning)

	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
}

type coreUpdateStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *coreUpdateStatus) nextStatus(existedSts *appsv1.StatefulSet, existedDeploy *appsv1.Deployment) {
	if s.emqxStatusMachine.emqx.Status.CurrentImage != s.emqxStatusMachine.emqx.Spec.Image {
		s.emqxStatusMachine.creating.nextStatus(existedSts, existedDeploy)
		return
	}

	if existedSts == nil {
		return
	}

	// statefulSet already updated
	if existedSts.Spec.Template.Spec.Containers[0].Image != s.emqxStatusMachine.emqx.Spec.Image ||
		existedSts.Status.ObservedGeneration != existedSts.Generation {
		return
	}
	// statefulSet is ready
	if existedSts.Status.UpdateRevision != existedSts.Status.CurrentRevision ||
		existedSts.Status.UpdatedReplicas != existedSts.Status.Replicas ||
		existedSts.Status.ReadyReplicas != existedSts.Status.Replicas {
		return
	}
	// core nodes is ready
	if s.emqxStatusMachine.emqx.Status.CoreNodeReadyReplicas != s.emqxStatusMachine.emqx.Status.CoreNodeReplicas {
		return
	}
	condition := appsv2alpha2.NewCondition(
		appsv2alpha2.ClusterCoreReady,
		corev1.ConditionTrue,
		"ClusterCoreReady",
		"Core nodes is ready",
	)
	s.emqxStatusMachine.emqx.Status.SetCondition(*condition)
	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
}

type coreReadyStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *coreReadyStatus) nextStatus(existedSts *appsv1.StatefulSet, existedDeploy *appsv1.Deployment) {
	if s.emqxStatusMachine.emqx.Status.CurrentImage != s.emqxStatusMachine.emqx.Spec.Image {
		s.emqxStatusMachine.creating.nextStatus(existedSts, existedDeploy)
		return
	}

	// statefulSet is ready
	if existedSts.UID == "" ||
		existedSts.Spec.Template.Spec.Containers[0].Image != s.emqxStatusMachine.emqx.Spec.Image ||
		existedSts.Status.ReadyReplicas != existedSts.Status.Replicas ||
		existedSts.Status.UpdatedReplicas != existedSts.Status.Replicas ||
		existedSts.Status.UpdateRevision != existedSts.Status.CurrentRevision {
		return
	}

	// deployment is ready
	if existedDeploy.UID == "" ||
		existedDeploy.Spec.Template.Spec.Containers[0].Image != s.emqxStatusMachine.emqx.Spec.Image ||
		existedDeploy.Status.UpdatedReplicas != existedDeploy.Status.Replicas ||
		existedDeploy.Status.ReadyReplicas != existedDeploy.Status.Replicas {
		return
	}

	// emqx nodes is ready
	if s.emqxStatusMachine.emqx.Status.CoreNodeReplicas != s.emqxStatusMachine.emqx.Status.CoreNodeReadyReplicas ||
		s.emqxStatusMachine.emqx.Status.ReplicantNodeReplicas != s.emqxStatusMachine.emqx.Status.ReplicantNodeReadyReplicas {
		return
	}

	condition := appsv2alpha2.NewCondition(
		appsv2alpha2.ClusterRunning,
		corev1.ConditionTrue,
		"ClusterRunning",
		"Cluster is running",
	)
	s.emqxStatusMachine.emqx.Status.SetCondition(*condition)
	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
}

type runningStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *runningStatus) nextStatus(existedSts *appsv1.StatefulSet, existedDeploy *appsv1.Deployment) {
	if s.emqxStatusMachine.emqx.Status.CurrentImage != s.emqxStatusMachine.emqx.Spec.Image {
		s.emqxStatusMachine.creating.nextStatus(existedSts, existedDeploy)
		return
	}

	if s.emqxStatusMachine.emqx.Status.ReplicantNodeReadyReplicas != s.emqxStatusMachine.emqx.Status.ReplicantNodeReplicas {
		s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2alpha2.ClusterRunning)
	}

	if s.emqxStatusMachine.emqx.Status.CoreNodeReadyReplicas != s.emqxStatusMachine.emqx.Status.CoreNodeReplicas {
		s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2alpha2.ClusterRunning)
		s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2alpha2.ClusterCoreReady)
	}

	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
}
