package controller

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

type admissionAction int

const (
	// Pod removal blocked; reason explains why.
	admissionWait admissionAction = iota
	// Pod may be removed right now.
	admissionRemove
	// Pod needs evacuation before it can be removed.
	admissionEvacuate
)

type admissionCause string

const (
	scaleDown     admissionCause = "scale down"
	rollingUpdate admissionCause = "rolling update"
)

type admission struct {
	Action admissionAction
	Reason string
	Cause  admissionCause
}

// podAdmission pairs a pod with its admission decision, preserving iteration order.
type podAdmission struct {
	Admission admission
	Pod       *corev1.Pod
}

func (a admission) wait(reason string, args ...any) admission {
	return a.withActionReason(admissionWait, reason, args...)
}

func (a admission) remove(reason string, args ...any) admission {
	return a.withActionReason(admissionRemove, reason, args...)
}

func (a admission) evacuate(reason string, args ...any) admission {
	return a.withActionReason(admissionEvacuate, reason, args...)
}

func (a admission) withActionReason(action admissionAction, reason string, args ...any) admission {
	a.Action = action
	if len(args) > 0 {
		a.Reason = fmt.Sprintf(reason, args...)
	} else {
		a.Reason = reason
	}
	return a
}
