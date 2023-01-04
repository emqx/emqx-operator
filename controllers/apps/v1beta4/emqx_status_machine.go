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
	"context"

	emperror "emperror.dev/errors"
	"github.com/emqx/emqx-operator/apis/apps/v1beta4"
	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type status interface {
	nextStatus(appsv1beta4.Emqx) *requeue
}

type emqxStatusMachine struct {
	emqx v1beta4.Emqx
	*EmqxReconciler
	ctx context.Context

	empty             status
	init              status
	running           status
	blueGreenUpdating status

	currentStatus status
}

func newEmqxStatusMachine(emqx appsv1beta4.Emqx, reconcile *EmqxReconciler, ctx context.Context) *emqxStatusMachine {
	emqxStatusMachine := &emqxStatusMachine{
		emqx:           emqx,
		EmqxReconciler: reconcile,
		ctx:            ctx,
	}

	emptyStatus := &emptyStatus{emqxStatusMachine: emqxStatusMachine}
	initStatus := &initStatus{emqxStatusMachine: emqxStatusMachine}
	runningStatus := &runningStatus{emqxStatusMachine: emqxStatusMachine}
	blueGreenUpdatingStatus := &blueGreenUpdatingStatus{emqxStatusMachine: emqxStatusMachine}

	emqxStatusMachine.empty = emptyStatus
	emqxStatusMachine.init = initStatus
	emqxStatusMachine.running = runningStatus
	emqxStatusMachine.blueGreenUpdating = blueGreenUpdatingStatus
	emqxStatusMachine.setCurrentStatus(emqx)

	return emqxStatusMachine
}

func (s *emqxStatusMachine) setCurrentStatus(emqx appsv1beta4.Emqx) {
	conditions := emqx.GetStatus().GetConditions()
	if conditions == nil {
		s.currentStatus = s.empty
		return
	}

	if conditions[0].Type == appsv1beta4.ConditionInitResourceReady {
		s.currentStatus = s.init
		return
	}

	if conditions[0].Type == appsv1beta4.ConditionRunning {
		s.currentStatus = s.running
		return
	}

	if conditions[0].Type == appsv1beta4.ConditionBlueGreenUpdating {
		s.currentStatus = s.blueGreenUpdating
		return
	}
}

type emptyStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *emptyStatus) nextStatus(instance appsv1beta4.Emqx) *requeue {
	instance.GetStatus().AddCondition(
		appsv1beta4.ConditionInitResourceReady,
		corev1.ConditionFalse,
		"",
		"",
	)
	s.emqxStatusMachine.emqx = instance
	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
	if err := s.emqxStatusMachine.EmqxReconciler.Client.Status().Update(s.emqxStatusMachine.ctx, s.emqxStatusMachine.emqx); err != nil {
		return &requeue{err: emperror.Wrap(err, "failed to update emqx status")}
	}
	return nil
}

type initStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *initStatus) nextStatus(instance appsv1beta4.Emqx) *requeue {
	if !instance.GetStatus().IsInitResourceReady() {
		bootstrap_user := generateBootstrapUserSecret(instance)
		plugins, err := s.emqxStatusMachine.EmqxReconciler.createInitPluginList(instance)
		if err != nil {
			return &requeue{err: emperror.Wrap(err, "failed to get init plugin list")}
		}
		resource := []client.Object{}
		resource = append(resource, bootstrap_user)
		resource = append(resource, plugins...)

		conditionStatus := corev1.ConditionTrue
		for _, r := range resource {
			u := &unstructured.Unstructured{}
			u.SetGroupVersionKind(r.GetObjectKind().GroupVersionKind())
			if err := s.emqxStatusMachine.EmqxReconciler.Client.Get(s.emqxStatusMachine.ctx, client.ObjectKeyFromObject(r), u); err != nil {
				if k8sErrors.IsNotFound(err) {
					conditionStatus = corev1.ConditionFalse
					break
				}
				return &requeue{err: err}
			}
		}

		instance.GetStatus().AddCondition(
			appsv1beta4.ConditionInitResourceReady,
			conditionStatus,
			"",
			"",
		)
		s.emqxStatusMachine.emqx = instance
		s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
		if err := s.emqxStatusMachine.EmqxReconciler.Client.Status().Update(s.emqxStatusMachine.ctx, s.emqxStatusMachine.emqx); err != nil {
			return &requeue{err: emperror.Wrap(err, "failed to update emqx status")}
		}
		return nil
	}
	if err := addReadyReplicas(s.emqxStatusMachine.EmqxReconciler, instance); err != nil {
		return &requeue{err: err}
	}

	if err := addRunningOrUpdating(s.emqxStatusMachine.EmqxReconciler, instance); err != nil {
		return &requeue{err: err}
	}

	s.emqxStatusMachine.emqx = instance
	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
	if err := s.emqxStatusMachine.EmqxReconciler.Client.Status().Update(s.emqxStatusMachine.ctx, s.emqxStatusMachine.emqx); err != nil {
		return &requeue{err: emperror.Wrap(err, "failed to update emqx status")}
	}
	return nil
}

type runningStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *runningStatus) nextStatus(instance appsv1beta4.Emqx) *requeue {
	if err := addReadyReplicas(s.emqxStatusMachine.EmqxReconciler, instance); err != nil {
		return &requeue{err: err}
	}

	if err := addRunningOrUpdating(s.emqxStatusMachine.EmqxReconciler, instance); err != nil {
		return &requeue{err: err}
	}

	s.emqxStatusMachine.emqx = instance
	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
	if err := s.emqxStatusMachine.EmqxReconciler.Client.Status().Update(s.emqxStatusMachine.ctx, s.emqxStatusMachine.emqx); err != nil {
		return &requeue{err: emperror.Wrap(err, "failed to update emqx status")}
	}
	return nil
}

type blueGreenUpdatingStatus struct {
	emqxStatusMachine *emqxStatusMachine
}

func (s *blueGreenUpdatingStatus) nextStatus(instance appsv1beta4.Emqx) *requeue {
	if err := addReadyReplicas(s.emqxStatusMachine.EmqxReconciler, instance); err != nil {
		return &requeue{err: err}
	}

	if err := addRunningOrUpdating(s.emqxStatusMachine.EmqxReconciler, instance); err != nil {
		return &requeue{err: err}
	}

	s.emqxStatusMachine.emqx = instance
	s.emqxStatusMachine.setCurrentStatus(s.emqxStatusMachine.emqx)
	if err := s.emqxStatusMachine.EmqxReconciler.Client.Status().Update(s.emqxStatusMachine.ctx, s.emqxStatusMachine.emqx); err != nil {
		return &requeue{err: emperror.Wrap(err, "failed to update emqx status")}
	}
	return nil
}

func addReadyReplicas(r *EmqxReconciler, instance appsv1beta4.Emqx) error {
	emqxNodes, err := r.getNodeStatusesByAPI(instance)
	if err != nil {
		r.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedToGetNodeStatues", err.Error())
		return emperror.Wrap(err, "failed to get node statuses")
	}

	readyReplicas := int32(0)
	for _, node := range emqxNodes {
		if node.NodeStatus == "Running" {
			readyReplicas++
		}
	}
	instance.GetStatus().SetEmqxNodes(emqxNodes)
	instance.GetStatus().SetReadyReplicas(readyReplicas)
	instance.GetStatus().SetReplicas(*instance.GetSpec().GetReplicas())
	return nil
}

func addRunningOrUpdating(r *EmqxReconciler, instance appsv1beta4.Emqx) error {
	inClusterStss, err := r.getInClusterStatefulSets(instance)
	if err != nil {
		return emperror.Wrap(err, "failed to get in cluster statefulsets")
	}
	if len(inClusterStss) == 1 {
		instance.GetStatus().SetCurrentStatefulSetVersion(inClusterStss[0].Status.CurrentRevision)
		if instance.GetStatus().GetReadyReplicas() == instance.GetStatus().GetReplicas() {
			instance.GetStatus().AddCondition(
				appsv1beta4.ConditionRunning,
				corev1.ConditionTrue,
				"ClusterReady",
				"All resources are ready",
			)
		} else {
			instance.GetStatus().AddCondition(
				appsv1beta4.ConditionRunning,
				corev1.ConditionFalse,
				"ClusterNotReady",
				"Some resources are not ready",
			)
		}
		if enterprise, ok := instance.(*appsv1beta4.EmqxEnterprise); ok {
			enterprise.Status.EmqxBlueGreenUpdateStatus = nil
		}
	}
	if len(inClusterStss) > 1 {
		enterprise, ok := instance.(*appsv1beta4.EmqxEnterprise)
		if !ok {
			return emperror.New("blueGreenUpdatingStatus only support EmqxEnterprise")
		}

		originSts := inClusterStss[len(inClusterStss)-1]
		currentSts := inClusterStss[len(inClusterStss)-2]

		enterprise.GetStatus().SetCurrentStatefulSetVersion(currentSts.Status.CurrentRevision)
		enterprise.GetStatus().AddCondition(
			appsv1beta4.ConditionBlueGreenUpdating,
			corev1.ConditionTrue,
			"BlueGreenUpdateStarted",
			"",
		)

		ok, err := checkEndpointSliceIsReady(r, enterprise, currentSts)
		if err != nil {
			return emperror.Wrap(err, "failed to check endpoint slice is ready")
		}
		if ok {
			evacuationsStatus, err := r.getEvacuationStatusByAPI(enterprise)
			if err != nil {
				return emperror.Wrap(err, "failed to get evacuation status")
			}
			enterprise.Status.EmqxBlueGreenUpdateStatus = &appsv1beta4.EmqxBlueGreenUpdateStatus{
				OriginStatefulSet:  originSts.Name,
				CurrentStatefulSet: currentSts.Name,
				StartedAt:          metav1.Now(),
				EvacuationsStatus:  evacuationsStatus,
			}
		}
	}
	return nil
}

func checkEndpointSliceIsReady(r *EmqxReconciler, instance appsv1beta4.Emqx, currentSts *appsv1.StatefulSet) (bool, error) {
	// make sure that only latest ready sts is in endpoints
	endpointSlice := &discoveryv1.EndpointSliceList{}
	if err := r.Client.List(context.Background(), endpointSlice,
		client.InNamespace(instance.GetNamespace()),
		client.MatchingLabels(instance.GetSpec().GetServiceTemplate().Labels),
	); err != nil {
		return false, err
	}

	podMap, _ := r.getPodMap(instance, []*appsv1.StatefulSet{currentSts})

	hitEndpoints := 0
	for _, endpointSlice := range endpointSlice.Items {
		if len(endpointSlice.Endpoints) != int(*instance.GetSpec().GetReplicas()) {
			continue
		}
		for _, endpoint := range endpointSlice.Endpoints {
			if endpoint.Conditions.Ready == nil || !*endpoint.Conditions.Ready {
				continue
			}

			for _, pod := range podMap[currentSts.UID] {
				if endpoint.TargetRef.UID != pod.UID {
					continue
				}
				hitEndpoints++
			}
		}
	}
	if hitEndpoints != len(podMap[currentSts.UID]) {
		// Wait for endpoints to be ready
		return false, nil
	}
	return true, nil
}
