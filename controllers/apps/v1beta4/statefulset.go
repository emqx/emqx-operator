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
	"sort"

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	innerErr "github.com/emqx/emqx-operator/internal/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getPodMap(ctx context.Context, k8sClient client.Client, instance appsv1beta4.Emqx, allSts []*appsv1.StatefulSet) (map[types.UID][]*corev1.Pod, error) {
	podList := &corev1.PodList{}
	labelSelector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: instance.GetLabels(),
	})
	err := k8sClient.List(
		ctx,
		podList,
		&client.ListOptions{
			Namespace:     instance.GetNamespace(),
			LabelSelector: labelSelector,
		},
	)
	if err != nil {
		return nil, err
	}

	podMap := make(map[types.UID][]*corev1.Pod, len(allSts))
	for _, sts := range allSts {
		podMap[sts.UID] = []*corev1.Pod{}
	}
	for _, p := range podList.Items {
		// Do not ignore inactive Pods because Recreate Deployments need to verify that no
		// Pods from older versions are running before spinning up new Pods.
		pod := p.DeepCopy()
		controllerRef := metav1.GetControllerOf(pod)
		if controllerRef == nil {
			continue
		}
		// Only append if we care about this UID.
		if _, ok := podMap[controllerRef.UID]; ok {
			podMap[controllerRef.UID] = append(podMap[controllerRef.UID], pod)
		}
	}
	for i, pods := range podMap {
		sort.Sort(PodsByNameOlder(pods))
		podMap[i] = pods
	}

	return podMap, nil
}

func getAllStatefulSet(ctx context.Context, k8sClient client.Client, instance appsv1beta4.Emqx) ([]*appsv1.StatefulSet, error) {
	existedStsList := &appsv1.StatefulSetList{}
	labelSelector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: instance.GetLabels(),
	})
	err := k8sClient.List(
		ctx,
		existedStsList,
		&client.ListOptions{
			Namespace:     instance.GetNamespace(),
			LabelSelector: labelSelector,
		},
	)
	if err != nil {
		return nil, err
	}

	allSts := []*appsv1.StatefulSet{}
	for i := range existedStsList.Items {
		if *existedStsList.Items[i].Spec.Replicas != 0 {
			allSts = append(allSts, &existedStsList.Items[i])
		}
	}

	if len(allSts) == 0 {
		return nil, innerErr.ErrStsNotReady
	}
	sort.Sort(StatefulSetsBySizeNewer(allSts))
	return allSts, nil
}

func getInClusterStatefulSets(ctx context.Context, k8sClient client.Client, instance appsv1beta4.Emqx) ([]*appsv1.StatefulSet, error) {
	allSts, err := getAllStatefulSet(ctx, k8sClient, instance)
	if err != nil {
		return nil, err
	}

	podMap, err := getPodMap(ctx, k8sClient, instance, allSts)
	if err != nil {
		return nil, err
	}

	inCluster := []*appsv1.StatefulSet{}
	for _, sts := range allSts {
		readyCount := int32(0)
		for _, pod := range podMap[sts.UID] {
			if pod.Status.Phase == corev1.PodRunning {
				emqxNodeName := getEmqxNodeName(instance, pod)
				for _, emqxNode := range instance.GetStatus().GetEmqxNodes() {
					if emqxNodeName == emqxNode.Node && emqxNode.NodeStatus == "Running" {
						readyCount++
					}
				}
			}

		}
		if readyCount == sts.Status.CurrentReplicas {
			inCluster = append(inCluster, sts)
		}
	}
	if len(inCluster) == 0 {
		return nil, innerErr.ErrStsNotReady
	}
	return inCluster, nil
}
