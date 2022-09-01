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

//+kubebuilder:object:generate=false
type EmqxStatus interface {
	IsRunning() bool
	IsPluginInitialized() bool
	GetConditions() []Condition
	SetCondition(c Condition)
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
	now := metav1.Now()
	nowString := now.Format(time.RFC3339)
	return &Condition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     nowString,
		LastUpdateAt:       now,
		LastTransitionTime: nowString,
		Reason:             reason,
		Message:            message,
	}
}

func (s *Status) IsRunning() bool {
	if len(s.Conditions) == 0 {
		return false
	}
	c := s.Conditions[0]
	if c.Type == ConditionRunning && c.Status == corev1.ConditionTrue {
		return true
	}
	return false
}

func (s *Status) IsPluginInitialized() bool {
	// Init Plugin
	for _, c := range s.Conditions {
		if c.Type == ConditionPluginInitialized && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func (s *Status) GetConditions() []Condition {
	return s.Conditions
}

func (s *Status) SetCondition(c Condition) {
	pos, cp := getCondition(s, c.Type)
	if cp != nil &&
		cp.Status == c.Status && cp.Reason == c.Reason && cp.Message == c.Message {
		now := metav1.Now()
		nowString := now.Format(time.RFC3339)
		s.Conditions[pos].LastUpdateAt = now
		s.Conditions[pos].LastUpdateTime = nowString
		s.sortConditions(s.Conditions)
		return
	}

	if cp != nil {
		s.Conditions[pos] = c
	} else {
		s.Conditions = append(s.Conditions, c)
	}

	s.sortConditions(s.Conditions)
}

func (s *Status) sortConditions(conditions []Condition) {
	sort.Slice(conditions, func(i, j int) bool {
		return s.Conditions[j].LastUpdateAt.Before(&s.Conditions[i].LastUpdateAt)
	})
}

func getCondition(status *Status, t ConditionType) (int, *Condition) {
	for i, c := range status.Conditions {
		if t == c.Type {
			return i, &c
		}
	}
	return -1, nil
}
