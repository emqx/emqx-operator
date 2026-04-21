/*
Copyright 2025-2026.

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

import corev1 "k8s.io/api/core/v1"

const DefaultContainerName string = "emqx"

const (
	// labels
	LabelInstance        string = "apps.emqx.io/instance"   // my-emqx
	LabelManagedBy       string = "apps.emqx.io/managed-by" // emqx-operator
	LabelDBRole          string = "apps.emqx.io/db-role"    // core, replicant
	LabelPodTemplateHash string = "apps.emqx.io/pod-template-hash"
)

const (
	// annotations
	AnnotationLastEMQXConfig string = "apps.emqx.io/last-emqx-configuration"
	// AnnotationScalingDown marks a pod that the operator has committed to removing.
	// Pods with this annotation bypass the maxUnavailable budget on subsequent reconcile
	// iterations, preventing deadlocks when evacuation makes the pod unavailable.
	AnnotationScalingDown string = "apps.emqx.io/scaling-down"
)

const (
	// Whether the pod is responsible for DS replication
	DSReplicationSite corev1.PodConditionType = "apps.emqx.io/ds-replication-site"
)
