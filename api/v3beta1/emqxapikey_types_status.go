/*
Copyright 2026.

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

package v3beta1

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EMQXAPIKeyStatus defines the observed state of an EMQX API key.
type EMQXAPIKeyStatus struct {
	// Conditions represent the current status of the EMQX API key.
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

const (
	EMQXAPIKeyReady   string = "Ready"
	EMQXAPIKeyIssuing string = "Issuing"
	EMQXAPIKeyExpired string = "Expired"
)

func (s *EMQXAPIKeyStatus) SetCondition(
	ty string,
	status metav1.ConditionStatus,
	observedGeneration int64,
	reason,
	message string,
) bool {
	return meta.SetStatusCondition(&s.Conditions, metav1.Condition{
		Type:               ty,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: observedGeneration,
	})
}

func (s *EMQXAPIKeyStatus) GetCondition(conditionType string) *metav1.Condition {
	return meta.FindStatusCondition(s.Conditions, conditionType)
}

func (s *EMQXAPIKeyStatus) IsConditionTrue(conditionType string) bool {
	condition := s.GetCondition(conditionType)
	return condition != nil && condition.Status == metav1.ConditionTrue
}
