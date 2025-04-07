/*
Copyright 2025.

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

import corev1 "k8s.io/api/core/v1"

const DefaultContainerName string = "emqx"

const DefaultBootstrapAPIKey string = "emqx-operator-controller"

const EnterpriseEdition string = "Enterprise"

const (
	// labels
	LabelsInstanceKey        string = "apps.emqx.io/instance"   // my-emqx
	LabelsManagedByKey       string = "apps.emqx.io/managed-by" // emqx-operator
	LabelsDBRoleKey          string = "apps.emqx.io/db-role"    // core, replicant
	LabelsPodTemplateHashKey string = "apps.emqx.io/pod-template-hash"
)

const (
	// annotations
	AnnotationsLastEMQXConfigKey string = "apps.emqx.io/last-emqx-configuration"
)

const (
	// Whether the pod is responsible for DS replication
	DSReplicationSite corev1.PodConditionType = "apps.emqx.io/ds-replication-site"
	// https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-readiness-gate
	PodOnServing corev1.PodConditionType = "apps.emqx.io/on-serving"
)
