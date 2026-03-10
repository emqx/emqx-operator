package controller

import (
	appsv1 "k8s.io/api/apps/v1"
)

func NumReplicas(sts *appsv1.StatefulSet) int32 {
	if sts.Spec.Replicas != nil {
		return *sts.Spec.Replicas
	}
	return 1
}
