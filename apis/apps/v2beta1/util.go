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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func IsExistReplicant(instance *EMQX) bool {
	return instance.Spec.ReplicantTemplate != nil && instance.Spec.ReplicantTemplate.Spec.Replicas != nil && *instance.Spec.ReplicantTemplate.Spec.Replicas > 0
}

func DefaultLabels(instance *EMQX) map[string]string {
	labels := map[string]string{}
	labels[LabelsInstanceKey] = instance.Name
	labels[LabelsManagedByKey] = "emqx-operator"
	return labels
}

func DefaultCoreLabels(instance *EMQX) map[string]string {
	return CloneAndAddLabel(
		DefaultLabels(instance),
		LabelsDBRoleKey,
		"core",
	)
}

func DefaultReplicantLabels(instance *EMQX) map[string]string {
	return CloneAndAddLabel(
		DefaultLabels(instance),
		LabelsDBRoleKey,
		"replicant",
	)
}

func CloneAndMergeMap(dst map[string]string, src ...map[string]string) map[string]string {
	new := map[string]string{}

	for key, value := range dst {
		new[key] = value
	}

	for _, m := range src {
		for key, value := range m {
			if _, ok := new[key]; !ok {
				new[key] = value
			}
		}
	}

	return new
}

// Clones the given selector and returns a new selector with the given key and value added.
// Returns the given selector, if labelKey is empty.
func CloneSelectorAndAddLabel(selector *metav1.LabelSelector, labelKey, labelValue string) *metav1.LabelSelector {
	if labelKey == "" {
		// Don't need to add a label.
		return selector
	}

	// Clone.
	newSelector := new(metav1.LabelSelector)

	// TODO(madhusudancs): Check if you can use deepCopy_extensions_LabelSelector here.
	newSelector.MatchLabels = make(map[string]string)
	if selector.MatchLabels != nil {
		for key, val := range selector.MatchLabels {
			newSelector.MatchLabels[key] = val
		}
	}
	newSelector.MatchLabels[labelKey] = labelValue

	if selector.MatchExpressions != nil {
		newMExps := make([]metav1.LabelSelectorRequirement, len(selector.MatchExpressions))
		for i, me := range selector.MatchExpressions {
			newMExps[i].Key = me.Key
			newMExps[i].Operator = me.Operator
			if me.Values != nil {
				newMExps[i].Values = make([]string, len(me.Values))
				copy(newMExps[i].Values, me.Values)
			} else {
				newMExps[i].Values = nil
			}
		}
		newSelector.MatchExpressions = newMExps
	} else {
		newSelector.MatchExpressions = nil
	}

	return newSelector
}

// AddLabelToSelector returns a selector with the given key and value added to the given selector's MatchLabels.
func AddLabelToSelector(selector *metav1.LabelSelector, labelKey, labelValue string) *metav1.LabelSelector {
	if labelKey == "" {
		// Don't need to add a label.
		return selector
	}
	if selector.MatchLabels == nil {
		selector.MatchLabels = make(map[string]string)
	}
	selector.MatchLabels[labelKey] = labelValue
	return selector
}

// Clones the given map and returns a new map with the given key and value added.
// Returns the given map, if labelKey is empty.
func CloneAndAddLabel(labels map[string]string, labelKey, labelValue string) map[string]string {
	if labelKey == "" {
		// Don't need to add a label.
		return labels
	}
	// Clone.
	newLabels := map[string]string{}
	for key, value := range labels {
		newLabels[key] = value
	}
	newLabels[labelKey] = labelValue
	return newLabels
}

// CloneAndRemoveLabel clones the given map and returns a new map with the given key removed.
// Returns the given map, if labelKey is empty.
func CloneAndRemoveLabel(labels map[string]string, labelKey string) map[string]string {
	if labelKey == "" {
		// Don't need to add a label.
		return labels
	}
	// Clone.
	newLabels := map[string]string{}
	for key, value := range labels {
		newLabels[key] = value
	}
	delete(newLabels, labelKey)
	return newLabels
}

// AddLabel returns a map with the given key and value added to the given map.
func AddLabel(labels map[string]string, labelKey, labelValue string) map[string]string {
	if labelKey == "" {
		// Don't need to add a label.
		return labels
	}
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[labelKey] = labelValue
	return labels
}

func FindPodCondition(pod *corev1.Pod, conditionType corev1.PodConditionType) *corev1.PodCondition {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}

func MergeServicePorts(ports1, ports2 []corev1.ServicePort) []corev1.ServicePort {
	ports := append(ports1, ports2...)

	result := make([]corev1.ServicePort, 0, len(ports))
	tempName := map[string]struct{}{}
	tempPort := map[int32]struct{}{}

	for _, item := range ports {
		_, nameOK := tempName[item.Name]
		_, portOK := tempPort[item.Port]

		if !nameOK && !portOK {
			tempName[item.Name] = struct{}{}
			tempPort[item.Port] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}

func MergeContainerPorts(ports1, ports2 []corev1.ContainerPort) []corev1.ContainerPort {
	ports := append(ports1, ports2...)

	result := make([]corev1.ContainerPort, 0, len(ports))
	tempName := map[string]struct{}{}
	tempPort := map[int32]struct{}{}

	for _, item := range ports {
		_, nameOK := tempName[item.Name]
		_, portOK := tempPort[item.ContainerPort]

		if !nameOK && !portOK {
			tempName[item.Name] = struct{}{}
			tempPort[item.ContainerPort] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}

func TransServicePortsToContainerPorts(ports []corev1.ServicePort) []corev1.ContainerPort {
	result := make([]corev1.ContainerPort, 0, len(ports))
	for _, item := range ports {
		result = append(result, corev1.ContainerPort{
			Name:          item.Name,
			ContainerPort: item.Port,
			Protocol:      item.Protocol,
		})
	}
	return result
}

func mergeMap(dst, src map[string]string) map[string]string {
	if dst == nil {
		dst = make(map[string]string)
	}

	for key, value := range src {
		if _, ok := dst[key]; !ok {
			dst[key] = value
		}
	}
	return dst
}
