package v1beta4

import (
	"sort"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-readiness-gate
	PodInCluster corev1.PodConditionType = "apps.emqx.io/in-cluster"

	PodOnServing corev1.PodConditionType = "apps.emqx.io/on-Serving"
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
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

// ConditionType defines the condition that the RF can have
type ConditionType string

const (
	ConditionRunning           ConditionType = "Running"
	ConditionBlueGreenUpdating ConditionType = "BlueGreenUpdating"
	ConditionProcess           ConditionType = "Process"
	ConditionComplete          ConditionType = "Complete"
	ConditionFailed            ConditionType = "Failed"
)

// +kubebuilder:object:generate=false
type EmqxStatus interface {
	GetReplicas() int32
	SetReplicas(replicas int32)
	GetReadyReplicas() int32
	SetReadyReplicas(readyReplicas int32)
	GetEmqxNodes() []EmqxNode
	SetEmqxNodes(nodes []EmqxNode)
	GetCurrentStatefulSetVersion() string
	SetCurrentStatefulSetVersion(version string)
	GetConditions() []Condition
	AddCondition(condType ConditionType, status corev1.ConditionStatus, reason, message string)
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

type EmqxEvacuationStats struct {
	InitialSessions  *int32 `json:"initial_sessions,omitempty"`
	InitialConnected *int32 `json:"initial_connected,omitempty"`
	CurrentSessions  *int32 `json:"current_sessions,omitempty"`
	CurrentConnected *int32 `json:"current_connected,omitempty"`
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

type EmqxBlueGreenUpdateStatus struct {
	OriginStatefulSet  string                 `json:"originStatefulSet,omitempty"`
	CurrentStatefulSet string                 `json:"currentStatefulSet,omitempty"`
	StartedAt          *metav1.Time           `json:"startedAt,omitempty"`
	EvacuationsStatus  []EmqxEvacuationStatus `json:"evacuationsStatus,omitempty"`
}

func addCondition(conditions []Condition, c Condition) []Condition {
	now := metav1.Now()
	c.LastUpdateTime = now
	c.LastTransitionTime = now
	index := indexCondition(conditions, c.Type)
	if index == -1 {
		conditions = append(conditions, c)
	}
	if index == 0 {
		c.LastTransitionTime = conditions[0].LastTransitionTime
		conditions[0] = c
	}
	if index > 0 {
		conditions[index] = c
	}

	sort.Slice(conditions, func(i, j int) bool {
		return conditions[j].LastUpdateTime.Before(&conditions[i].LastUpdateTime)
	})
	return conditions
}

func indexCondition(conditions []Condition, t ConditionType) int {
	for i, c := range conditions {
		if t == c.Type {
			return i
		}
	}
	return -1
}

func IsClusterReady(s EmqxStatus) bool {
	index := indexCondition(s.GetConditions(), ConditionRunning)
	if index == 0 && s.GetConditions()[index].Status == corev1.ConditionTrue {
		return true
	}
	index = indexCondition(s.GetConditions(), ConditionBlueGreenUpdating)
	if index == 0 && s.GetConditions()[index].Status == corev1.ConditionTrue {
		return true
	}

	return false
}
