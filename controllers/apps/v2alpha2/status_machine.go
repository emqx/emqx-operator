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

	// EMQX cluster status
	initialized               status
	coreNodesProgressing      status
	coreNodesReady            status
	replicantNodesProgressing status
	replicantNodesReady       status
	available                 status
	ready                     status

	currentStatus status
}

func newEMQXStatusMachine(emqx *appsv2alpha2.EMQX) *emqxStatusMachine {
	emqxStatusMachine := &emqxStatusMachine{
		emqx: emqx,
	}

	initializedStatus := &initializedStatus{emqxStatusMachine: emqxStatusMachine}
	coreNodesProgressingStatus := &coreNodesProgressingStatus{emqxStatusMachine: emqxStatusMachine}
	codeNodesReadyStatus := &codeNodesReadyStatus{emqxStatusMachine: emqxStatusMachine}
	replicantNodesProgressingStatus := &replicantNodesProgressingStatus{emqxStatusMachine: emqxStatusMachine}
	replicantNodesReadyStatus := &replicantNodesReadyStatus{emqxStatusMachine: emqxStatusMachine}
	availableStatus := &availableStatus{emqxStatusMachine: emqxStatusMachine}
	readyStatus := &readyStatus{emqxStatusMachine: emqxStatusMachine}

	emqxStatusMachine.initialized = initializedStatus
	emqxStatusMachine.coreNodesProgressing = coreNodesProgressingStatus
	emqxStatusMachine.coreNodesReady = codeNodesReadyStatus
	emqxStatusMachine.replicantNodesProgressing = replicantNodesProgressingStatus
	emqxStatusMachine.replicantNodesReady = replicantNodesReadyStatus
	emqxStatusMachine.available = availableStatus
	emqxStatusMachine.ready = readyStatus
	emqxStatusMachine.setCurrentStatus(emqx)

	return emqxStatusMachine
}

func (s *emqxStatusMachine) setCurrentStatus(emqx *appsv2alpha2.EMQX) {
	condition := emqx.Status.GetLastTrueCondition()
	if condition == nil {
		condition = &metav1.Condition{
			Type:    appsv2alpha2.Initialized,
			Status:  metav1.ConditionTrue,
			Reason:  "Initialized",
			Message: "initialized EMQX cluster",
		}
		s.emqx.Status.SetCondition(*condition)
	}

	switch condition.Type {
	case appsv2alpha2.CoreNodesProgressing:
		s.currentStatus = s.coreNodesProgressing
	case appsv2alpha2.CoreNodesReady:
		s.currentStatus = s.coreNodesReady
	case appsv2alpha2.ReplicantNodesProgressing:
		s.currentStatus = s.replicantNodesProgressing
	case appsv2alpha2.ReplicantNodesReady:
		s.currentStatus = s.replicantNodesReady
	case appsv2alpha2.Available:
		s.currentStatus = s.available
	case appsv2alpha2.Ready:
		s.currentStatus = s.ready
	default:
		s.currentStatus = s.initialized
	}
}

func (s *emqxStatusMachine) NextStatus(currentSts *appsv1.StatefulSet, currentRs *appsv1.ReplicaSet) {
	s.currentStatus.nextStatus(currentSts, currentRs)
}

func (s *emqxStatusMachine) GetEMQX() *appsv2alpha2.EMQX {
	return s.emqx
}

type initializedStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *initializedStatus) nextStatus(_ *appsv1.StatefulSet, _ *appsv1.ReplicaSet) {
	s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2alpha2.ReplicantNodesProgressing)
	s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2alpha2.ReplicantNodesReady)

	s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2alpha2.CoreNodesProgressing)
	s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2alpha2.CoreNodesReady)

	s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2alpha2.Available)
	s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2alpha2.Ready)

	s.emqxStatusMachine.emqx.Status.SetCondition(metav1.Condition{
		Type:    appsv2alpha2.CoreNodesProgressing,
		Status:  metav1.ConditionTrue,
		Reason:  "CoreNodesProgressing",
		Message: "Core nodes progressing",
	})
	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
}

type coreNodesProgressingStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *coreNodesProgressingStatus) nextStatus(currentSts *appsv1.StatefulSet, currentRs *appsv1.ReplicaSet) {
	// statefulSet is ready
	if currentSts.UID == "" ||
		currentSts.Status.ObservedGeneration != currentSts.Generation ||
		currentSts.Status.ReadyReplicas != currentSts.Status.Replicas {
		return
	}

	// core nodes is ready
	if s.emqxStatusMachine.emqx.Status.CoreNodesStatus.ReadyReplicas < s.emqxStatusMachine.emqx.Status.CoreNodesStatus.Replicas {
		return
	}

	s.emqxStatusMachine.emqx.Status.SetCondition(metav1.Condition{
		Type:    appsv2alpha2.CoreNodesReady,
		Status:  metav1.ConditionTrue,
		Reason:  "CoreNodesReady",
		Message: "Core nodes is ready",
	})
	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
}

type codeNodesReadyStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *codeNodesReadyStatus) nextStatus(currentSts *appsv1.StatefulSet, currentRs *appsv1.ReplicaSet) {
	if isExistReplicant(s.emqxStatusMachine.emqx) {
		s.emqxStatusMachine.emqx.Status.SetCondition(metav1.Condition{
			Type:    appsv2alpha2.ReplicantNodesProgressing,
			Status:  metav1.ConditionTrue,
			Reason:  appsv2alpha2.ReplicantNodesProgressing,
			Message: "Replicant nodes progressing",
		})
		s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
		return
	}

	s.emqxStatusMachine.emqx.Status.SetCondition(metav1.Condition{
		Type:    appsv2alpha2.Available,
		Status:  metav1.ConditionTrue,
		Reason:  appsv2alpha2.Available,
		Message: "Cluster is available",
	})
	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
}

type replicantNodesProgressingStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *replicantNodesProgressingStatus) nextStatus(currentSts *appsv1.StatefulSet, currentRs *appsv1.ReplicaSet) {
	if !isExistReplicant(s.emqxStatusMachine.emqx) {
		s.emqxStatusMachine.initialized.nextStatus(currentSts, currentRs)
		return
	}

	if currentRs.UID == "" ||
		currentRs.Status.ReadyReplicas != currentRs.Status.Replicas {
		return
	}

	// replicant nodes is ready
	if s.emqxStatusMachine.emqx.Status.ReplicantNodesStatus.ReadyReplicas < s.emqxStatusMachine.emqx.Status.ReplicantNodesStatus.Replicas {
		return
	}

	s.emqxStatusMachine.emqx.Status.SetCondition(metav1.Condition{
		Type:    appsv2alpha2.ReplicantNodesReady,
		Status:  metav1.ConditionTrue,
		Reason:  appsv2alpha2.ReplicantNodesReady,
		Message: "Replicant nodes ready",
	})
	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
}

type replicantNodesReadyStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *replicantNodesReadyStatus) nextStatus(currentSts *appsv1.StatefulSet, currentRs *appsv1.ReplicaSet) {
	if !isExistReplicant(s.emqxStatusMachine.emqx) {
		s.emqxStatusMachine.initialized.nextStatus(currentSts, currentRs)
		return
	}

	s.emqxStatusMachine.emqx.Status.SetCondition(metav1.Condition{
		Type:    appsv2alpha2.Available,
		Status:  metav1.ConditionTrue,
		Reason:  appsv2alpha2.Available,
		Message: "Cluster is available",
	})
	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
}

type availableStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *availableStatus) nextStatus(currentSts *appsv1.StatefulSet, currentRs *appsv1.ReplicaSet) {
	if s.emqxStatusMachine.emqx.Status.CoreNodesStatus.ReadyReplicas != s.emqxStatusMachine.emqx.Status.CoreNodesStatus.Replicas {
		return
	}
	if isExistReplicant(s.emqxStatusMachine.emqx) {
		if s.emqxStatusMachine.emqx.Status.ReplicantNodesStatus.ReadyReplicas != s.emqxStatusMachine.emqx.Status.ReplicantNodesStatus.Replicas {
			return
		}
	}

	s.emqxStatusMachine.emqx.Status.SetCondition(metav1.Condition{
		Type:    appsv2alpha2.Ready,
		Status:  metav1.ConditionTrue,
		Reason:  appsv2alpha2.Ready,
		Message: "Cluster is ready",
	})
	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
}

type readyStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *readyStatus) nextStatus(currentSts *appsv1.StatefulSet, currentRs *appsv1.ReplicaSet) {}
