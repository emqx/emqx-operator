package controllers

import (
	"Emqx/api/v1alpha1"
	"fmt"

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

func getStorageCommonValue(instance *v1alpha1.Broker) StorageCommonValue {
	storageCommonValue := StorageCommonValue{
		AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
		Capacity: map[v1.ResourceName]resource.Quantity{
			v1.ResourceStorage: instance.Spec.Storage.Capacity,
		},
		StorageClassName: "nas",
		VolumeMode:       v1.PersistentVolumeFilesystem,
	}
	return storageCommonValue

}

func makePvFromSpec(instance *v1alpha1.Broker, key StorageKey, value StorageValue) *v1.PersistentVolume {
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", "pv", key),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: instance.APIVersion,
					Kind:       instance.Kind,
					Name:       instance.Name,
					UID:        instance.UID,
				},
			},
		},
		Spec: makePvSpec(instance, key, value),
	}
	pv.Namespace = instance.Namespace
	return pv
}

func makePvSpec(instance *v1alpha1.Broker, key StorageKey, value StorageValue) v1.PersistentVolumeSpec {
	storageCommonValue := getStorageCommonValue(instance)
	csi := v1.PersistentVolumeSource{
		CSI: &v1.CSIPersistentVolumeSource{
			Driver:       instance.Spec.Storage.Driver,
			VolumeHandle: fmt.Sprintf("%s-%s", "pv", key),
			VolumeAttributes: map[string]string{
				"path":   string(value),
				"server": instance.Spec.Storage.Server,
				"vers":   "3",
			},
		},
	}
	persistentVolumeReclaimPolicy := v1.PersistentVolumeReclaimRetain

	pvSpec := v1.PersistentVolumeSpec{
		Capacity:                      storageCommonValue.Capacity,
		PersistentVolumeSource:        csi,
		AccessModes:                   storageCommonValue.AccessModes,
		PersistentVolumeReclaimPolicy: persistentVolumeReclaimPolicy,
		StorageClassName:              storageCommonValue.StorageClassName,
		VolumeMode:                    &storageCommonValue.VolumeMode,
	}
	return pvSpec
}

func makePvcFromSpec(instance *v1alpha1.Broker, key StorageKey) *v1.PersistentVolumeClaim {
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", "pvc", key),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: instance.APIVersion,
					Kind:       instance.Kind,
					Name:       instance.Name,
					UID:        instance.UID,
				},
			},
		},
		Spec: makePvcSpec(instance, key),
	}
	pvc.Namespace = instance.Namespace
	return pvc
}

func makePvcSpec(instance *v1alpha1.Broker, key StorageKey) v1.PersistentVolumeClaimSpec {
	storageCommonValue := getStorageCommonValue(instance)
	pvcSpec := v1.PersistentVolumeClaimSpec{
		AccessModes: storageCommonValue.AccessModes,
		Resources: v1.ResourceRequirements{
			Requests: storageCommonValue.Capacity,
		},
		StorageClassName: &storageCommonValue.StorageClassName,
		VolumeMode:       &storageCommonValue.VolumeMode,
		VolumeName:       fmt.Sprintf("%s-%s", "pv", key),
	}
	return pvcSpec
}
