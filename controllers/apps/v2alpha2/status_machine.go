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

	initializedStatus := &initializedStatus{emqxStatusMachine: emqxStatusMachine}
	coreNodesProgressingStatus := &coreNodesProgressingStatus{emqxStatusMachine: emqxStatusMachine}
	codeNodesReadyStatus := &codeNodesReadyStatus{emqxStatusMachine: emqxStatusMachine}
	readyStatus := &readyStatus{emqxStatusMachine: emqxStatusMachine}

	emqxStatusMachine.initialized = initializedStatus
	emqxStatusMachine.coreNodesProgressing = coreNodesProgressingStatus
	emqxStatusMachine.codeNodesReady = codeNodesReadyStatus
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
	case appsv2alpha2.CodeNodesReady:
		s.currentStatus = s.codeNodesReady
	case appsv2alpha2.Ready:
		s.currentStatus = s.ready
	default:
		s.currentStatus = s.initialized
	}
}

func (s *emqxStatusMachine) NextStatus(existedSts *appsv1.StatefulSet, existedRs *appsv1.ReplicaSet) {
	s.currentStatus.nextStatus(existedSts, existedRs)
}

func (s *emqxStatusMachine) GetEMQX() *appsv2alpha2.EMQX {
	return s.emqx
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
	if existedSts.UID == "" ||
		existedSts.Spec.Template.Spec.Containers[0].Image != s.emqxStatusMachine.emqx.Spec.Image ||
		existedSts.Status.ObservedGeneration != existedSts.Generation {
		return
	}
	// statefulSet is ready
	if existedSts.UID == "" ||
		existedSts.Status.UpdateRevision != existedSts.Status.CurrentRevision ||
		existedSts.Status.UpdatedReplicas != existedSts.Status.Replicas ||
		existedSts.Status.ReadyReplicas != existedSts.Status.Replicas {
		return
	}
	// core nodes is ready
	if s.emqxStatusMachine.emqx.Status.CoreNodesStatus.ReadyReplicas < s.emqxStatusMachine.emqx.Status.CoreNodesStatus.Replicas {
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

	// core nodes is ready
	if s.emqxStatusMachine.emqx.Status.CoreNodesStatus.ReadyReplicas < s.emqxStatusMachine.emqx.Status.CoreNodesStatus.Replicas {
		return
	}

	if isExistReplicant(s.emqxStatusMachine.emqx) {
		// replicaSet is ready
		if existedRs.UID == "" ||
			existedRs.Spec.Template.Spec.Containers[0].Image != s.emqxStatusMachine.emqx.Spec.Image ||
			existedRs.Status.ReadyReplicas != existedRs.Status.Replicas {
			return
		}

		// replicant nodes is ready
		if s.emqxStatusMachine.emqx.Status.ReplicantNodesStatus.ReadyReplicas < s.emqxStatusMachine.emqx.Status.ReplicantNodesStatus.Replicas {
			return
		}
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

	if isExistReplicant(s.emqxStatusMachine.emqx) {
		if s.emqxStatusMachine.emqx.Status.ReplicantNodesStatus.ReadyReplicas != s.emqxStatusMachine.emqx.Status.ReplicantNodesStatus.Replicas {
			s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2alpha2.Ready)
		}
	}

	if s.emqxStatusMachine.emqx.Status.CoreNodesStatus.ReadyReplicas != s.emqxStatusMachine.emqx.Status.CoreNodesStatus.Replicas {
		s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2alpha2.Ready)
		s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2alpha2.CodeNodesReady)
	}

	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
}
