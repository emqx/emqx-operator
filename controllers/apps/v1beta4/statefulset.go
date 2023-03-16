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

func getPodMap(k8sClient client.Client, instance appsv1beta4.Emqx, allSts []*appsv1.StatefulSet) (map[types.UID][]*corev1.Pod, error) {
	podList := &corev1.PodList{}
	labelSelector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: instance.GetLabels(),
	})
	err := k8sClient.List(
		context.TODO(),
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
		sort.Sort(PodsByNameNewer(pods))
		podMap[i] = pods
	}

	return podMap, nil
}

func getAllStatefulSet(k8sClient client.Client, instance appsv1beta4.Emqx) ([]*appsv1.StatefulSet, error) {
	existedStsList := &appsv1.StatefulSetList{}
	labelSelector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: instance.GetLabels(),
	})
	err := k8sClient.List(
		context.TODO(),
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

func getInClusterStatefulSets(k8sClient client.Client, instance appsv1beta4.Emqx) ([]*appsv1.StatefulSet, error) {
	allSts, err := getAllStatefulSet(k8sClient, instance)
	if err != nil {
		return nil, err
	}

	podMap, err := getPodMap(k8sClient, instance, allSts)
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

// StatefulSetsByCreationTimestamp sorts a list of StatefulSet by creation timestamp, using their names as a tie breaker.
type StatefulSetsByCreationTimestamp []*appsv1.StatefulSet

func (o StatefulSetsByCreationTimestamp) Len() int      { return len(o) }
func (o StatefulSetsByCreationTimestamp) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o StatefulSetsByCreationTimestamp) Less(i, j int) bool {
	if o[i].CreationTimestamp.Equal(&o[j].CreationTimestamp) {
		return o[i].Name < o[j].Name
	}
	return o[i].CreationTimestamp.Before(&o[j].CreationTimestamp)
}

// StatefulSetsBySizeOlder sorts a list of StatefulSet by size in descending order, using their creation timestamp or name as a tie breaker.
// By using the creation timestamp, this sorts from old to new replica sets.
type StatefulSetsBySizeOlder []*appsv1.StatefulSet

func (o StatefulSetsBySizeOlder) Len() int      { return len(o) }
func (o StatefulSetsBySizeOlder) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o StatefulSetsBySizeOlder) Less(i, j int) bool {
	if *(o[i].Spec.Replicas) == *(o[j].Spec.Replicas) {
		return StatefulSetsByCreationTimestamp(o).Less(i, j)
	}
	return *(o[i].Spec.Replicas) > *(o[j].Spec.Replicas)
}

// StatefulSetsBySizeNewer sorts a list of StatefulSet by size in descending order, using their creation timestamp or name as a tie breaker.
// By using the creation timestamp, this sorts from new to old replica sets.
type StatefulSetsBySizeNewer []*appsv1.StatefulSet

func (o StatefulSetsBySizeNewer) Len() int      { return len(o) }
func (o StatefulSetsBySizeNewer) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o StatefulSetsBySizeNewer) Less(i, j int) bool {
	if *(o[i].Spec.Replicas) == *(o[j].Spec.Replicas) {
		return StatefulSetsByCreationTimestamp(o).Less(j, i)
	}
	return *(o[i].Spec.Replicas) > *(o[j].Spec.Replicas)
}

// PodsByCreationTimestamp sorts a list of Pod by creation timestamp, using their names as a tie breaker.
type PodsByCreationTimestamp []*corev1.Pod

func (o PodsByCreationTimestamp) Len() int      { return len(o) }
func (o PodsByCreationTimestamp) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o PodsByCreationTimestamp) Less(i, j int) bool {
	if o[i].CreationTimestamp.Equal(&o[j].CreationTimestamp) {
		return o[i].Name < o[j].Name
	}
	return o[i].CreationTimestamp.Before(&o[j].CreationTimestamp)
}

// PodsByNameOlder sorts a list of Pod by size in descending order, using their creation timestamp or name as a tie breaker.
// By using the creation timestamp, this sorts from old to new replica sets.
type PodsByNameOlder []*corev1.Pod

func (o PodsByNameOlder) Len() int      { return len(o) }
func (o PodsByNameOlder) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o PodsByNameOlder) Less(i, j int) bool {
	if o[i].Name == o[j].Name {
		return PodsByCreationTimestamp(o).Less(i, j)
	}
	return o[i].Name > o[j].Name
}

// PodsByNameNewer sorts a list of Pod by size in descending order, using their creation timestamp or name as a tie breaker.
// By using the creation timestamp, this sorts from new to old replica sets.
type PodsByNameNewer []*corev1.Pod

func (o PodsByNameNewer) Len() int      { return len(o) }
func (o PodsByNameNewer) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o PodsByNameNewer) Less(i, j int) bool {
	if o[i].Name == o[j].Name {
		return PodsByCreationTimestamp(o).Less(j, i)
	}
	return o[i].Name > o[j].Name
}
