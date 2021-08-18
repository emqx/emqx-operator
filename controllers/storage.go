package controllers

import (
	"fmt"

	"github.com/emqx/emqx-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func makePvcFromSpec(instance *v1alpha1.Emqx, item string) *v1.PersistentVolumeClaim {
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", "pvc", item),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: instance.APIVersion,
					Kind:       instance.Kind,
					Name:       instance.Name,
					UID:        instance.UID,
				},
			},
		},
		Spec: makePvcSpec(instance, item),
	}
	pvc.Namespace = instance.Namespace
	return pvc
}

func makePvcSpec(instance *v1alpha1.Emqx, item string) v1.PersistentVolumeClaimSpec {
	storageCommonValue := getStorageCommonValue(instance)
	pvcSpec := v1.PersistentVolumeClaimSpec{
		AccessModes: storageCommonValue.AccessModes,
		Resources: v1.ResourceRequirements{
			Requests: storageCommonValue.Capacity,
		},
		StorageClassName: &storageCommonValue.StorageClassName,
		VolumeMode:       &storageCommonValue.VolumeMode,
		VolumeName:       fmt.Sprintf("%s-%s", "pv", item),
	}
	return pvcSpec
}
