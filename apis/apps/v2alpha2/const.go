package v2alpha2

import corev1 "k8s.io/api/core/v1"

const DefaultContainerName string = "emqx"

const DefaultBootstrapAPIKey string = "emqx-operator-controller"

const (
	// labels
	ManagerByLabelKey       string = "apps.emqx.io/managed-by"
	InstanceNameLabelKey    string = "apps.emqx.io/instance"
	DBRoleLabelKey          string = "apps.emqx.io/db-role"
	PodTemplateHashLabelKey string = "apps.emqx.io/pod-template-hash"
)

const (
	// annotations
	LastEMQXConfigAnnotationKey = "apps.emqx.io/last-emqx-configuration"
)

const (
	// https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-readiness-gate
	PodOnServing corev1.PodConditionType = "apps.emqx.io/on-serving"
)
