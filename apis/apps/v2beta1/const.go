package v2beta1

import corev1 "k8s.io/api/core/v1"

const DefaultContainerName string = "emqx"

const DefaultBootstrapAPIKey string = "emqx-operator-controller"

const (
	// labels
	LabelsNameKey            string = "app.kubernetes.io/name"       // emqx
	LabelsInstanceKey        string = "app.kubernetes.io/instance"   // my-emqx
	LabelsComponentKey       string = "app.kubernetes.io/component"  // core, replicant, dashboard, listeners, config
	LabelsPartOfKey          string = "app.kubernetes.io/part-of"    // emqx
	LabelsManagedByKey       string = "app.kubernetes.io/managed-by" // emqx-operator
	LabelsPodTemplateHashKey string = "apps.emqx.io/pod-template-hash"
)

const (
	// annotations
	AnnotationsLastEMQXConfigKey string = "apps.emqx.io/last-emqx-configuration"
)

const (
	// https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-readiness-gate
	PodOnServing corev1.PodConditionType = "apps.emqx.io/on-serving"
)
