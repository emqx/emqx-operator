package v1beta3

import (
	"sort"
	"time"

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
	ConditionPluginInitialized ConditionType = "PluginInitialized"
	ConditionRunning           ConditionType = "Running"
)

// +kubebuilder:object:generate=false
type EmqxStatus interface {
	IsRunning() bool
	IsPluginInitialized() bool
	GetConditions() []Condition
	SetCondition(c Condition)
	ClearCondition(t ConditionType)
}

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

// Emqx Status defines the observed state of EMQX
type Status struct {
	// Represents the latest available observations of a EMQX current state.
	Conditions []Condition `json:"conditions,omitempty"`
	// Nodes of the EMQX cluster
	EmqxNodes []EmqxNode `json:"emqxNodes,omitempty"`
	// replicas is the number of Pods created by the EMQX Custom Resource controller.
	Replicas int32 `json:"replicas,omitempty"`
	// readyReplicas is the number of pods created for this EMQX Custom Resource with a EMQX Ready.
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`
}

func NewCondition(condType ConditionType, status corev1.ConditionStatus, reason, message string) *Condition {
	return &Condition{
		Type:    condType,
		Status:  status,
		Reason:  reason,
		Message: message,
	}
}

func (s *Status) IsRunning() bool {
	index := indexCondition(s, ConditionRunning)
	return index == 0 && s.Conditions[index].Status == corev1.ConditionTrue
}

func (s *Status) IsPluginInitialized() bool {
	// Init Plugin
	index := indexCondition(s, ConditionPluginInitialized)
	if index == -1 {
		return false
	}
	return s.Conditions[index].Status == corev1.ConditionTrue
}

func (s *Status) GetConditions() []Condition {
	return s.Conditions
}

func (s *Status) SetCondition(c Condition) {
	now := metav1.Now()
	c.LastUpdateAt = now
	c.LastUpdateTime = now.Format(time.RFC3339)
	c.LastTransitionTime = now.Format(time.RFC3339)
	pos := indexCondition(s, c.Type)
	if pos >= 0 {
		if s.Conditions[pos].Status == c.Status && s.Conditions[pos].LastTransitionTime != "" {
			c.LastTransitionTime = s.Conditions[pos].LastTransitionTime
		}
		s.Conditions[pos] = c
	} else {
		s.Conditions = append(s.Conditions, c)
	}
	s.sortConditions(s.Conditions)
}

func (s *Status) ClearCondition(t ConditionType) {
	pos := indexCondition(s, t)
	if pos == -1 {
		return
	}
	s.Conditions = append(s.Conditions[:pos], s.Conditions[pos+1:]...)
}

func (s *Status) sortConditions(conditions []Condition) {
	sort.Slice(conditions, func(i, j int) bool {
		return s.Conditions[j].LastUpdateAt.Before(&s.Conditions[i].LastUpdateAt)
	})
}

func indexCondition(status *Status, t ConditionType) int {
	for i, c := range status.Conditions {
		if t == c.Type {
			return i
		}
	}
	return -1
}
