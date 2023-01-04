package v1beta4

import (
	"context"

	emperror "emperror.dev/errors"
	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func nextStatus(r *EmqxReconciler, ctx context.Context, instance appsv1beta4.Emqx) *requeue {
	if !instance.GetStatus().IsInitResourceReady() {
		if err := addInitResourceReady(r, ctx, instance); err != nil {
			return err
		}
		if err := r.Client.Status().Update(ctx, instance); err != nil {
			return &requeue{err: emperror.Wrap(err, "failed to update emqx status")}
		}
		return nil
	}

	if err := addReadyReplicas(r, instance); err != nil {
		return &requeue{err: err}
	}

	if err := addRunningOrUpdating(r, instance); err != nil {
		return &requeue{err: err}
	}

	if err := r.Client.Status().Update(ctx, instance); err != nil {
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

func addInitResourceReady(r *EmqxReconciler, ctx context.Context, instance appsv1beta4.Emqx) *requeue {
	bootstrap_user := generateBootstrapUserSecret(instance)
	plugins, err := r.createInitPluginList(instance)
	if err != nil {
		return &requeue{err: emperror.Wrap(err, "failed to get init plugin list")}
	}
	resources := []client.Object{}
	resources = append(resources, bootstrap_user)
	resources = append(resources, plugins...)

	conditionStatus := corev1.ConditionTrue
	for _, resource := range resources {
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(resource.GetObjectKind().GroupVersionKind())
		if err := r.Client.Get(ctx, client.ObjectKeyFromObject(resource), u); err != nil {
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
