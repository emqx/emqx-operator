package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type Storage struct {
	// Name of the Storage
	StorageClassName string                          `json:"storageClassName,omitempty"`
	AccessModes      []v1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`
	Capacity         resource.Quantity               `json:"capacity,omitempty"`
	VolumeMode       *v1.PersistentVolumeMode        `json:"volumeMode,omitempty" protobuf:"bytes,8,opt,name=volumeMode,casttype=PersistentVolumeMode"`
}
