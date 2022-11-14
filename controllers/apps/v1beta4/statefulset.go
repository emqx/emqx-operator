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
	"fmt"
	"sort"
	"strings"

	emperror "emperror.dev/errors"
	"github.com/banzaicloud/k8s-objectmatcher/patch"
	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *EmqxReconciler) getPodMap(instance appsv1beta4.Emqx, allSts []*appsv1.StatefulSet) (map[types.UID][]*corev1.Pod, error) {
	podList := &corev1.PodList{}
	labelSelector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: instance.GetLabels(),
	})
	err := r.Client.List(
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

func (r *EmqxReconciler) getAllStatefulSet(instance appsv1beta4.Emqx) ([]*appsv1.StatefulSet, error) {
	existedStsList := &appsv1.StatefulSetList{}
	labelSelector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: instance.GetLabels(),
	})
	err := r.Client.List(
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
		return nil, emperror.Errorf("not found statefulSet for instance: %s", instance.GetName())
	}
	sort.Sort(StatefulSetsBySizeNewer(allSts))
	return allSts, nil
}

func (r *EmqxReconciler) getInClusterStatefulSets(instance appsv1beta4.Emqx) ([]*appsv1.StatefulSet, error) {
	allSts, err := r.getAllStatefulSet(instance)
	if err != nil {
		return nil, err
	}

	podMap, err := r.getPodMap(instance, allSts)
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
		return nil, emperror.Errorf("not found in cluster statefulSets for instance: %s", instance.GetName())
	}
	return inCluster, nil
}

func (r *EmqxReconciler) getLatestReadyStatefulSet(instance appsv1beta4.Emqx, inCluster bool) (*appsv1.StatefulSet, error) {
	var list []*appsv1.StatefulSet
	var err error
	if inCluster {
		list, err = r.getInClusterStatefulSets(instance)
	} else {
		list, err = r.getAllStatefulSet(instance)
	}
	if err != nil {
		return nil, err
	}

	podMap, err := r.getPodMap(instance, list)
	if err != nil {
		return nil, err
	}

	for _, sts := range list {
		if len(podMap[sts.UID]) == int(*instance.GetSpec().GetReplicas()) {
			return sts, nil
		}
	}
	return nil, emperror.Errorf("not found ready statefulSet for instance: %s", instance.GetName())
}

func (r *EmqxReconciler) getNewStatefulSet(instance appsv1beta4.Emqx, newSts *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	allSts, _ := r.getAllStatefulSet(instance)

	patchOpts := []patch.CalculateOption{
		justCheckPodTemplate(),
	}

	for i := range allSts {
		patchResult, _ := r.Patcher.Calculate(
			allSts[i].DeepCopy(),
			newSts.DeepCopy(),
			patchOpts...,
		)
		if patchResult.IsEmpty() {
			newSts.ObjectMeta = *allSts[i].ObjectMeta.DeepCopy()
			return newSts, nil
		}
	}

	// Do-while loop
	var collisionCount *int32 = new(int32)
	for {
		podTemplateSpecHash := computeHash(&newSts.Spec.Template, collisionCount)
		name := newSts.Name + "-" + podTemplateSpecHash
		err := r.Client.Get(context.TODO(), types.NamespacedName{
			Namespace: newSts.Namespace,
			Name:      name,
		}, &appsv1.StatefulSet{})
		*collisionCount++

		if err != nil {
			if k8sErrors.IsNotFound(err) {
				newSts.Name = name
				return newSts, nil
			}
			return nil, err
		}
	}
}

func (r *EmqxReconciler) syncStatefulSet(instance appsv1beta4.Emqx, evacuationsStatus []appsv1beta4.EmqxEvacuationStatus) error {
	inClusterStss, err := r.getInClusterStatefulSets(instance)
	if err != nil {
		return err
	}

	podMap, err := r.getPodMap(instance, inClusterStss)
	if err != nil {
		return err
	}

	latestReadySts, err := r.getLatestReadyStatefulSet(instance, true)
	if err != nil {
		return err
	}

	i := 0
	for i <= len(inClusterStss)-1 {
		if inClusterStss[i].UID == latestReadySts.UID {
			break
		}
		i++
	}

	if len(inClusterStss[i+1:]) == 0 {
		return nil
	}
	otherStss := inClusterStss[i+1:]

	scaleDownSts := r.whoCanBeScaledDown(instance, evacuationsStatus, otherStss, podMap)
	if scaleDownSts != nil {
		scaleDown := *scaleDownSts.Spec.Replicas - 1
		stsCopy := scaleDownSts.DeepCopy()
		if err := r.Client.Get(context.TODO(), client.ObjectKeyFromObject(stsCopy), stsCopy); err != nil {
			if !k8sErrors.IsNotFound(err) {
				return err
			}
		}
		stsCopy.Spec.Replicas = &scaleDown

		r.EventRecorder.Event(instance, corev1.EventTypeNormal, "ScaleDown", fmt.Sprintf("scale down StatefulSet %s to %d", scaleDownSts.Name, scaleDown))
		if err := r.Client.Update(context.TODO(), stsCopy); err != nil {
			if !k8sErrors.IsConflict(err) {
				return err
			}
		}
	}

	if len(evacuationsStatus) == 0 {
		sort.Sort(StatefulSetsBySizeOlder(otherStss))
		pods := podMap[otherStss[0].UID]
		if len(pods) == 0 {
			return nil
		}
		// evacuate the last pod
		sort.Sort(PodsByNameNewer(pods))
		emqxNodeName := getEmqxNodeName(instance, pods[0])

		r.EventRecorder.Event(instance, corev1.EventTypeNormal, "Evacuate", fmt.Sprintf("evacuate node %s start", emqxNodeName))
		if err := r.evacuateNodeByAPI(instance, podMap[latestReadySts.UID], emqxNodeName); err != nil {
			return emperror.Wrap(err, "evacuate node failed")
		}
	}

	return nil
}

func (r *EmqxReconciler) whoCanBeScaledDown(instance appsv1beta4.Emqx, evacuationsStatus []appsv1beta4.EmqxEvacuationStatus, allSts []*appsv1.StatefulSet, podMap map[types.UID][]*corev1.Pod) *appsv1.StatefulSet {
	for _, e := range evacuationsStatus {
		if *e.Stats.CurrentConnected == 0 && *e.Stats.CurrentSessions == 0 && e.State == "prohibiting" {
			podName := strings.Split(strings.Split(e.Node, "@")[1], ".")[0]
			for _, sts := range allSts {
				if strings.Contains(podName, sts.Name) {
					pods := podMap[sts.UID]
					// Get latest pod for sts
					sort.Sort(PodsByNameNewer(pods))
					if pods[0].Name == podName {
						r.EventRecorder.Event(instance, corev1.EventTypeNormal, "Evacuate", fmt.Sprintf("evacuate node %s successfully", getEmqxNodeName(instance, pods[0])))
						return sts
					}
				}
			}
		}
	}
	return nil
}
