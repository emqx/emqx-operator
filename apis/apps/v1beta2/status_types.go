package v1beta2

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
	ClusterConditionAvailable   ConditionType = "Available"
	ClusterConditionHealthy     ConditionType = "Healthy"
	ClusterConditionRunning     ConditionType = "Running"
	ClusterConditionCreating    ConditionType = "Creating"
	ClusterConditionRecovering  ConditionType = "Recovering"
	ClusterConditionScaling     ConditionType = "Scaling"
	ClusterConditionScalingDown ConditionType = "ScalingDown"
	ClusterConditionUpgrading   ConditionType = "Upgrading"
	ClusterConditionUpdating    ConditionType = "Updating"
	ClusterConditionFailed      ConditionType = "Failed"
)

//+kubebuilder:object:generate=false
type EmqxStatus interface {
	DescConditionsByTime()
	GetConditions() []Condition
	SetScalingUpCondition(message string)
	SetCreateCondition(message string)
	SetScalingDownCondition(message string)
	SetUpgradingCondition(message string)
	SetUpdatingCondition(message string)
	SetReadyCondition(message string)
	SetFailedCondition(message string)
	setClusterCondition(c Condition)
	ClearCondition(t ConditionType)
}

// Emqx Status defines the observed state of EMQX
type Status struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Conditions []Condition `json:"conditions,omitempty"`
}

func (ecs *Status) DescConditionsByTime() {
	sort.Slice(ecs.Conditions, func(i, j int) bool {
		// return ecs.Conditions[i].LastUpdateAt.After(ecs.Conditions[j].LastUpdateAt)
		return ecs.Conditions[j].LastUpdateAt.Before(&ecs.Conditions[i].LastUpdateAt)
	})
}

func (ecs *Status) GetConditions() []Condition {
	return ecs.Conditions
}

func (ecs *Status) SetScalingUpCondition(message string) {
	c := newClusterCondition(ClusterConditionScaling, corev1.ConditionTrue, "Scaling up", message)
	ecs.setClusterCondition(*c)
}

func (ecs *Status) SetCreateCondition(message string) {
	c := newClusterCondition(ClusterConditionCreating, corev1.ConditionTrue, "Creating", message)
	ecs.setClusterCondition(*c)
}

func (ecs *Status) SetScalingDownCondition(message string) {
	c := newClusterCondition(ClusterConditionScaling, corev1.ConditionTrue, "Scaling down", message)
	ecs.setClusterCondition(*c)
}

func (ecs *Status) SetUpgradingCondition(message string) {
	c := newClusterCondition(ClusterConditionUpgrading, corev1.ConditionTrue, "Cluster upgrading", message)
	ecs.setClusterCondition(*c)
}

func (ecs *Status) SetUpdatingCondition(message string) {
	c := newClusterCondition(ClusterConditionUpdating, corev1.ConditionTrue, "Cluster updating", message)
	ecs.setClusterCondition(*c)
}

func (ecs *Status) SetReadyCondition(message string) {
	c := newClusterCondition(ClusterConditionHealthy, corev1.ConditionTrue, "Cluster available", message)
	ecs.setClusterCondition(*c)
}

func (ecs *Status) SetFailedCondition(message string) {
	c := newClusterCondition(ClusterConditionFailed, corev1.ConditionTrue, "Cluster failed", message)
	ecs.setClusterCondition(*c)
}

func (ecs *Status) setClusterCondition(c Condition) {
	pos, cp := getClusterCondition(ecs, c.Type)
	if cp != nil &&
		cp.Status == c.Status && cp.Reason == c.Reason && cp.Message == c.Message {
		now := metav1.Now()
		nowString := now.Format(time.RFC3339)
		ecs.Conditions[pos].LastUpdateAt = now
		ecs.Conditions[pos].LastUpdateTime = nowString
		return
	}

	if cp != nil {
		ecs.Conditions[pos] = c
	} else {
		ecs.Conditions = append(ecs.Conditions, c)
	}
}

func (ecs *Status) ClearCondition(t ConditionType) {
	pos, _ := getClusterCondition(ecs, t)
	if pos == -1 {
		return
	}
	ecs.Conditions = append(ecs.Conditions[:pos], ecs.Conditions[pos+1:]...)
}

func getClusterCondition(status *Status, t ConditionType) (int, *Condition) {
	for i, c := range status.Conditions {
		if t == c.Type {
			return i, &c
		}
	}
	return -1, nil
}

func newClusterCondition(condType ConditionType, status corev1.ConditionStatus, reason, message string) *Condition {
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
