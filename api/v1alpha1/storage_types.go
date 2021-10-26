package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

type Storage struct {
	// EmptyDirVolumeSource to be used by the Prometheus StatefulSets. If specified, used in place of any volumeClaimTemplate. More
	// info: https://kubernetes.io/docs/concepts/storage/volumes/#emptydir
	EmptyDir *corev1.EmptyDirVolumeSource `json:"emptyDir,omitempty"`
	// A PVC spec to be used by the Prometheus StatefulSets.
	PersistentVolumeClaim *corev1.PersistentVolumeClaim `json:"volumeClaimTemplate,omitempty"`
}
