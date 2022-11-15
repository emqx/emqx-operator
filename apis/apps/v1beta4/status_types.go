package v1beta4

import (
	"sort"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Phase of the RF status
type Phase string

// Condition saves the state information of the EMQX cluster
type Condition struct {
	// Status of cluster condition.
	Type ConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime string      `json:"lastUpdateTime,omitempty"`
	LastUpdateAt   metav1.Time `json:"-"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

// ConditionType defines the condition that the RF can have
type ConditionType string

const (
	ConditionInitResourceReady ConditionType = "InitResourceReady"
	ConditionRunning           ConditionType = "Running"
)

type EmqxNode struct {
	// EMQX node name
	Node string `json:"node,omitempty"`
	// EMQX node status
	NodeStatus string `json:"node_status,omitempty"`
	// Erlang/OTP version used by EMQX
	OTPRelease string `json:"otp_release,omitempty"`
	// EMQX version
	Version string `json:"version,omitempty"`
}

type EmqxEvacuationStats struct {
	InitialSessions  int32 `json:"initial_sessions,omitempty"`
	InitialConnected int32 `json:"initial_connected,omitempty"`
	CurrentSessions  int32 `json:"current_sessions,omitempty"`
	CurrentConnected int32 `json:"current_connected,omitempty"`
}

type EmqxEvacuationStatus struct {
	Node                   string              `json:"node,omitempty"`
	Stats                  EmqxEvacuationStats `json:"stats,omitempty"`
	State                  string              `json:"state,omitempty"`
	SessionRecipients      []string            `json:"session_recipients,omitempty"`
	SessionGoal            int32               `json:"session_goal,omitempty"`
	SessionEvictionRate    int32               `json:"session_eviction_rate,omitempty"`
	ConnectionGoal         int32               `json:"connection_goal,omitempty"`
	ConnectionEvictionRate int32               `json:"connection_eviction_rate,omitempty"`
}

func NewCondition(condType ConditionType, status corev1.ConditionStatus, reason, message string) *Condition {
	return &Condition{
		Type:    condType,
		Status:  status,
		Reason:  reason,
		Message: message,
	}
}

func sortConditions(conditions []Condition) {
	sort.Slice(conditions, func(i, j int) bool {
		return conditions[j].LastUpdateAt.Before(&conditions[i].LastUpdateAt)
	})
}

func indexCondition(conditions []Condition, t ConditionType) int {
	for i, c := range conditions {
		if t == c.Type {
			return i
		}
	}
	return -1
}
