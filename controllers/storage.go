package controllers

import (
	"fmt"

	"github.com/emqx/emqx-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type StorageCommonValue struct {
	AccessModes      []v1.PersistentVolumeAccessMode
	Capacity         map[v1.ResourceName]resource.Quantity
	StorageClassName string
	VolumeMode       v1.PersistentVolumeMode
}

func getStorageCommonValue(instance *v1alpha1.Emqx) StorageCommonValue {
	storageCommonValue := StorageCommonValue{
		AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
		Capacity: map[v1.ResourceName]resource.Quantity{
			v1.ResourceStorage: instance.Spec.Storage.Capacity,
		},
		StorageClassName: instance.Spec.Storage.StorageClassName,
		VolumeMode:       v1.PersistentVolumeFilesystem,
	}
	return storageCommonValue
}

func makePvcSpec(instance *v1alpha1.Emqx, item string) v1.PersistentVolumeClaimSpec {
	storageCommonValue := getStorageCommonValue(instance)

	return v1.PersistentVolumeClaimSpec{
		AccessModes: storageCommonValue.AccessModes,
		Resources: v1.ResourceRequirements{
			Requests: storageCommonValue.Capacity,
		},
		StorageClassName: &storageCommonValue.StorageClassName,
		VolumeMode:       &storageCommonValue.VolumeMode,
		VolumeName:       fmt.Sprintf("%s-%s", "pv", item),
	}
}
