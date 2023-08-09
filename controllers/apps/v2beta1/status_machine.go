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
	"context"

	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type status interface {
	nextStatus()
}

type emqxStatusMachine struct {
	emqx   *appsv2beta1.EMQX
	client client.Client

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

func newEMQXStatusMachine(k8sClient client.Client, emqx *appsv2beta1.EMQX) *emqxStatusMachine {
	emqxStatusMachine := &emqxStatusMachine{
		client: k8sClient,
		emqx:   emqx,
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

func (s *emqxStatusMachine) setCurrentStatus(emqx *appsv2beta1.EMQX) {
	condition := emqx.Status.GetLastTrueCondition()
	if condition == nil {
		condition = &metav1.Condition{
			Type:    appsv2beta1.Initialized,
			Status:  metav1.ConditionTrue,
			Reason:  "Initialized",
			Message: "initialized EMQX cluster",
		}
		s.emqx.Status.SetCondition(*condition)
	}

	switch condition.Type {
	case appsv2beta1.CoreNodesProgressing:
		s.currentStatus = s.coreNodesProgressing
	case appsv2beta1.CoreNodesReady:
		s.currentStatus = s.coreNodesReady
	case appsv2beta1.ReplicantNodesProgressing:
		s.currentStatus = s.replicantNodesProgressing
	case appsv2beta1.ReplicantNodesReady:
		s.currentStatus = s.replicantNodesReady
	case appsv2beta1.Available:
		s.currentStatus = s.available
	case appsv2beta1.Ready:
		s.currentStatus = s.ready
	default:
		s.currentStatus = s.initialized
	}
}

func (s *emqxStatusMachine) NextStatus() {
	s.currentStatus.nextStatus()
}

func (s *emqxStatusMachine) GetEMQX() *appsv2beta1.EMQX {
	return s.emqx
}

type initializedStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *initializedStatus) nextStatus() {
	s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2beta1.ReplicantNodesProgressing)
	s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2beta1.ReplicantNodesReady)

	s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2beta1.CoreNodesProgressing)
	s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2beta1.CoreNodesReady)

	s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2beta1.Available)
	s.emqxStatusMachine.emqx.Status.RemoveCondition(appsv2beta1.Ready)

	s.emqxStatusMachine.emqx.Status.SetCondition(metav1.Condition{
		Type:    appsv2beta1.CoreNodesProgressing,
		Status:  metav1.ConditionTrue,
		Reason:  "CoreNodesProgressing",
		Message: "Core nodes progressing",
	})
	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
}

type coreNodesProgressingStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *coreNodesProgressingStatus) nextStatus() {
	emqx := s.emqxStatusMachine.GetEMQX()

	updateSts, _, _ := getStateFulSetList(context.Background(), s.emqxStatusMachine.client, emqx)
	if updateSts != nil && updateSts.Status.ReadyReplicas == emqx.Status.CoreNodesStatus.Replicas {
		emqx.Status.SetCondition(metav1.Condition{
			Type:    appsv2beta1.CoreNodesReady,
			Status:  metav1.ConditionTrue,
			Reason:  "CoreNodesReady",
			Message: "Core nodes is ready",
		})
	}

	s.emqxStatusMachine.setCurrentStatus(emqx)
}

type codeNodesReadyStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *codeNodesReadyStatus) nextStatus() {
	emqx := s.emqxStatusMachine.GetEMQX()

	if appsv2beta1.IsExistReplicant(emqx) {
		emqx.Status.SetCondition(metav1.Condition{
			Type:    appsv2beta1.ReplicantNodesProgressing,
			Status:  metav1.ConditionTrue,
			Reason:  appsv2beta1.ReplicantNodesProgressing,
			Message: "Replicant nodes progressing",
		})
		s.emqxStatusMachine.setCurrentStatus(emqx)
		return
	}

	emqx.Status.SetCondition(metav1.Condition{
		Type:    appsv2beta1.Available,
		Status:  metav1.ConditionTrue,
		Reason:  appsv2beta1.Available,
		Message: "Cluster is available",
	})
	s.emqxStatusMachine.setCurrentStatus(emqx)
}

type replicantNodesProgressingStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *replicantNodesProgressingStatus) nextStatus() {
	emqx := s.emqxStatusMachine.GetEMQX()

	if !appsv2beta1.IsExistReplicant(emqx) {
		s.emqxStatusMachine.initialized.nextStatus()
		return
	}

	updateRs, _, _ := getReplicaSetList(context.Background(), s.emqxStatusMachine.client, emqx)
	if updateRs != nil && updateRs.Status.ReadyReplicas == emqx.Status.ReplicantNodesStatus.Replicas {
		emqx.Status.SetCondition(metav1.Condition{
			Type:    appsv2beta1.ReplicantNodesReady,
			Status:  metav1.ConditionTrue,
			Reason:  appsv2beta1.ReplicantNodesReady,
			Message: "Replicant nodes ready",
		})
	}

	s.emqxStatusMachine.setCurrentStatus(emqx)
}

type replicantNodesReadyStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *replicantNodesReadyStatus) nextStatus() {
	if !appsv2beta1.IsExistReplicant(s.emqxStatusMachine.emqx) {
		s.emqxStatusMachine.initialized.nextStatus()
		return
	}

	s.emqxStatusMachine.emqx.Status.SetCondition(metav1.Condition{
		Type:    appsv2beta1.Available,
		Status:  metav1.ConditionTrue,
		Reason:  appsv2beta1.Available,
		Message: "Cluster is available",
	})
	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
}

type availableStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *availableStatus) nextStatus() {
	emqx := s.emqxStatusMachine.GetEMQX()

	if emqx.Status.CoreNodesStatus.ReadyReplicas != emqx.Status.CoreNodesStatus.Replicas ||
		emqx.Status.CoreNodesStatus.UpdateRevision != emqx.Status.CoreNodesStatus.CurrentRevision {
		return
	}

	if appsv2beta1.IsExistReplicant(emqx) {
		if emqx.Status.ReplicantNodesStatus.ReadyReplicas != emqx.Status.ReplicantNodesStatus.Replicas ||
			emqx.Status.ReplicantNodesStatus.UpdateRevision != emqx.Status.ReplicantNodesStatus.CurrentRevision {
			return
		}
	}

	emqx.Status.SetCondition(metav1.Condition{
		Type:    appsv2beta1.Ready,
		Status:  metav1.ConditionTrue,
		Reason:  appsv2beta1.Ready,
		Message: "Cluster is ready",
	})
	s.emqxStatusMachine.setCurrentStatus(emqx)
}

type readyStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *readyStatus) nextStatus() {}
