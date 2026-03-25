package controller

import (
	"context"
	"encoding/json"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func FindPodCondition(pod *corev1.Pod, conditionType corev1.PodConditionType) *corev1.PodCondition {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}

func IsPodConditionTrue(pod *corev1.Pod, conditionType corev1.PodConditionType) bool {
	condition := FindPodCondition(pod, conditionType)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

func SwitchPodConditionStatus(condition *corev1.PodCondition, status corev1.ConditionStatus) {
	if condition.Status != status {
		condition.Status = status
		condition.LastTransitionTime = metav1.Now()
	}
}

func UpdatePodCondition(
	ctx context.Context,
	k8sClient client.Client,
	pod *corev1.Pod,
	condition corev1.PodCondition,
) error {
	patchBytes, _ := json.Marshal(corev1.Pod{
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{condition},
		},
	})
	patch := client.RawPatch(types.StrategicMergePatchType, patchBytes)
	return k8sClient.Status().Patch(ctx, pod, patch)
}

func PodReadyDuration(pod *corev1.Pod) time.Duration {
	cond := FindPodCondition(pod, corev1.PodReady)
	if cond == nil || cond.Status != corev1.ConditionTrue {
		return 0
	}
	return time.Since(cond.LastTransitionTime.Time)
}

func IsPodManagedBy(pod *corev1.Pod, object metav1.Object) bool {
	if metav1.GetControllerOf(pod) != nil && metav1.GetControllerOf(pod).UID == object.GetUID() {
		return true
	}
	return false
}
