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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type status interface {
	nextStatus(*appsv1.StatefulSet, *appsv1.ReplicaSet)
}

type emqxStatusMachine struct {
	emqx *appsv2alpha2.EMQX

	init status

	// EMQX cluster status
	initialized          status
	coreNodesProgressing status
	codeNodesReady       status
	ready                status

	currentStatus status
}

func newEMQXStatusMachine(emqx *appsv2alpha2.EMQX) *emqxStatusMachine {
	emqxStatusMachine := &emqxStatusMachine{
		emqx: emqx,
	}

	initStatus := &initStatus{emqxStatusMachine: emqxStatusMachine}
	initializedStatus := &initializedStatus{emqxStatusMachine: emqxStatusMachine}
	coreNodesProgressingStatus := &coreNodesProgressingStatus{emqxStatusMachine: emqxStatusMachine}
	codeNodesReadyStatus := &codeNodesReadyStatus{emqxStatusMachine: emqxStatusMachine}
	readyStatus := &readyStatus{emqxStatusMachine: emqxStatusMachine}

	emqxStatusMachine.init = initStatus
	emqxStatusMachine.initialized = initializedStatus
	emqxStatusMachine.coreNodesProgressing = coreNodesProgressingStatus
	emqxStatusMachine.codeNodesReady = codeNodesReadyStatus
	emqxStatusMachine.ready = readyStatus
	emqxStatusMachine.setCurrentStatus(emqx)

	return emqxStatusMachine
}

func (s *emqxStatusMachine) setCurrentStatus(emqx *appsv2alpha2.EMQX) {
	if emqx.Status.Conditions == nil {
		s.currentStatus = s.init
	}

	condition := emqx.Status.GetLastTrueCondition()
	if condition == nil {
		return
	}

	switch condition.Type {
	case appsv2alpha2.Initialized:
		s.currentStatus = s.initialized
	case appsv2alpha2.CoreNodesProgressing:
		s.currentStatus = s.coreNodesProgressing
	case appsv2alpha2.CodeNodesReady:
		s.currentStatus = s.codeNodesReady
	case appsv2alpha2.Ready:
		s.currentStatus = s.ready
	default:
		panic("unknown condition type")
	}
}

func (s *emqxStatusMachine) UpdateNodeCount(emqxNodes []appsv2alpha2.EMQXNode) {
	s.emqx.Status.CoreNodeStatus.Replicas = *s.emqx.Spec.CoreTemplate.Spec.Replicas
	s.emqx.Status.ReplicantNodeStatus.Replicas = *s.emqx.Spec.ReplicantTemplate.Spec.Replicas

	s.emqx.Status.CoreNodeStatus.ReadyReplicas = int32(0)
	s.emqx.Status.ReplicantNodeStatus.ReadyReplicas = int32(0)

	if emqxNodes != nil {
		s.emqx.Status.SetEMQXNodes(emqxNodes)

		for _, node := range emqxNodes {
			if node.NodeStatus == "running" {
				if node.Role == "core" {
					s.emqx.Status.CoreNodeStatus.ReadyReplicas++
				}
				if node.Role == "replicant" {
					s.emqx.Status.ReplicantNodeStatus.ReadyReplicas++
				}
			}
		}
	}
}

func (s *emqxStatusMachine) NextStatus(existedSts *appsv1.StatefulSet, existedRs *appsv1.ReplicaSet) {
	s.currentStatus.nextStatus(existedSts, existedRs)
}

func (s *emqxStatusMachine) GetEMQX() *appsv2alpha2.EMQX {
	return s.emqx
}

type initStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *initStatus) nextStatus(_ *appsv1.StatefulSet, _ *appsv1.ReplicaSet) {
	s.emqxStatusMachine.emqx.Status.SetCondition(metav1.Condition{
		Type:    appsv2alpha2.Initialized,
		Status:  metav1.ConditionTrue,
		Reason:  "Initialized",
		Message: "initialized EMQX cluster",
	})
	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
}

type initializedStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *initializedStatus) nextStatus(_ *appsv1.StatefulSet, _ *appsv1.ReplicaSet) {
	s.emqxStatusMachine.emqx.Status.CurrentImage = s.emqxStatusMachine.emqx.Spec.Image
	s.emqxStatusMachine.emqx.Status.SetCondition(metav1.Condition{
		Type:    appsv2alpha2.CoreNodesProgressing,
		Status:  metav1.ConditionTrue,
		Reason:  "CoreNodesProgressing",
		Message: "Updating core nodes in cluster",
	})
	s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2alpha2.CodeNodesReady)
	s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2alpha2.Ready)

	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
}

type coreNodesProgressingStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *coreNodesProgressingStatus) nextStatus(existedSts *appsv1.StatefulSet, existedRs *appsv1.ReplicaSet) {
	if s.emqxStatusMachine.emqx.Status.CurrentImage != s.emqxStatusMachine.emqx.Spec.Image {
		s.emqxStatusMachine.initialized.nextStatus(existedSts, existedRs)
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
	if s.emqxStatusMachine.emqx.Status.CoreNodeStatus.ReadyReplicas != s.emqxStatusMachine.emqx.Status.CoreNodeStatus.Replicas {
		return
	}
	s.emqxStatusMachine.emqx.Status.SetCondition(metav1.Condition{
		Type:    appsv2alpha2.CodeNodesReady,
		Status:  metav1.ConditionTrue,
		Reason:  "CodeNodesReady",
		Message: "Core nodes is ready",
	})
	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
}

type codeNodesReadyStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *codeNodesReadyStatus) nextStatus(existedSts *appsv1.StatefulSet, existedRs *appsv1.ReplicaSet) {
	if s.emqxStatusMachine.emqx.Status.CurrentImage != s.emqxStatusMachine.emqx.Spec.Image {
		s.emqxStatusMachine.initialized.nextStatus(existedSts, existedRs)
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

	// replicaSet is ready
	if existedRs.UID == "" ||
		existedRs.Spec.Template.Spec.Containers[0].Image != s.emqxStatusMachine.emqx.Spec.Image ||
		existedRs.Status.ReadyReplicas != existedRs.Status.Replicas {
		return
	}

	// emqx nodes is ready
	if s.emqxStatusMachine.emqx.Status.CoreNodeStatus.ReadyReplicas != s.emqxStatusMachine.emqx.Status.CoreNodeStatus.Replicas ||
		s.emqxStatusMachine.emqx.Status.ReplicantNodeStatus.ReadyReplicas != s.emqxStatusMachine.emqx.Status.ReplicantNodeStatus.Replicas {
		return
	}

	s.emqxStatusMachine.emqx.Status.SetCondition(metav1.Condition{
		Type:    appsv2alpha2.Ready,
		Status:  metav1.ConditionTrue,
		Reason:  "Ready",
		Message: "Cluster is ready",
	})
	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
}

type readyStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *readyStatus) nextStatus(existedSts *appsv1.StatefulSet, existedRs *appsv1.ReplicaSet) {
	if s.emqxStatusMachine.emqx.Status.CurrentImage != s.emqxStatusMachine.emqx.Spec.Image {
		s.emqxStatusMachine.initialized.nextStatus(existedSts, existedRs)
		return
	}

	if s.emqxStatusMachine.emqx.Status.ReplicantNodeStatus.ReadyReplicas != s.emqxStatusMachine.emqx.Status.ReplicantNodeStatus.Replicas {
		s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2alpha2.Ready)
	}

	if s.emqxStatusMachine.emqx.Status.CoreNodeStatus.ReadyReplicas != s.emqxStatusMachine.emqx.Status.CoreNodeStatus.Replicas {
		s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2alpha2.Ready)
		s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2alpha2.CodeNodesReady)
	}

	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
}
