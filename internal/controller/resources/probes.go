package controller

import (
	"github.com/emqx/emqx-operator/internal/emqx/api"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// EvacuationReadinessProbe returns a readiness probe that uses the EMQX
// Evacuation & Rebalance API availability check.
//
// This endpoint returns 503 when the node is being evacuated, causing
// kubelet to mark the pod not-ready and removing it from Service endpoints
// automatically.
//
// Keep in sync with `EMQXReplicantTemplate.ReadinessProbe` default.
func EvacuationReadinessProbe() *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/" + api.URLAvailabilityCheck,
				Port: intstr.FromString("dashboard"),
			},
		},
		InitialDelaySeconds: 10,
		TimeoutSeconds:      3,
		PeriodSeconds:       5,
		FailureThreshold:    1,
		SuccessThreshold:    1,
	}
}
