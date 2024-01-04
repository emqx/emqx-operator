package v2beta1

import corev1 "k8s.io/api/core/v1"

const DefaultContainerName string = "emqx"

const DefaultBootstrapAPIKey string = "emqx-operator-controller"

const (
	// labels
	LabelsInstanceKey        string = "apps.emqx.io/instance"   // my-emqx
	LabelsManagedByKey       string = "apps.emqx.io/managed-by" // emqx-operator
	LabelsDBRoleKey          string = "apps.emqx.io/db-role"    // core, replicant
	LabelsPodTemplateHashKey string = "apps.emqx.io/pod-template-hash"
)

const (
	// https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-readiness-gate
	PodOnServing corev1.PodConditionType = "apps.emqx.io/on-serving"
)
