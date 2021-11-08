package k8s

import (
	"github.com/emqx/emqx-operator/api/v1alpha2"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

// Event the client that push event to kubernetes
type Event interface {
	// NewSlaveAdd event ClusterScaling
	NewNodeAdd(object runtime.Object, message string)
	// SlaveRemove event ClusterScalingDown
	NodeRemove(object runtime.Object, message string)
	// CreateCluster event ClusterCreating
	CreateCluster(object runtime.Object)
	// UpdateCluster event ClusterUpdating
	UpdateCluster(object runtime.Object, message string)
	// UpgradedCluster event ClusterUpgrading
	UpgradedCluster(object runtime.Object, message string)
	// EnsureCluster event Ensure
	EnsureCluster(object runtime.Object)
	// CheckCluster event CheckAndHeal
	CheckCluster(object runtime.Object)
	// FailedCluster event ClusterFailed
	FailedCluster(object runtime.Object, message string)
	// HealthCluster event ClusterHealthy
	HealthCluster(object runtime.Object)
}

// EventOption is the Event client interface implementation that using API calls to kubernetes.
type EventOption struct {
	eventsCli record.EventRecorder
	logger    logr.Logger
}

// NewEvent returns a new Event client
func NewEvent(eventCli record.EventRecorder, logger logr.Logger) Event {
	return &EventOption{
		eventsCli: eventCli,
		logger:    logger,
	}
}

// NewNodeAdd implement the Event.Interface
func (e *EventOption) NewNodeAdd(object runtime.Object, message string) {
	e.eventsCli.Event(object, v1.EventTypeNormal, string(v1alpha2.ClusterConditionScaling), message)
}

// NodeRemove implement the Event.Interface
func (e *EventOption) NodeRemove(object runtime.Object, message string) {
	e.eventsCli.Event(object, v1.EventTypeNormal, string(v1alpha2.ClusterConditionScalingDown), message)
}

// CreateCluster implement the Event.Interface
func (e *EventOption) CreateCluster(object runtime.Object) {
	e.eventsCli.Event(object, v1.EventTypeNormal, string(v1alpha2.ClusterConditionCreating), "Bootstrap emqx cluster")
}

// UpdateCluster implement the Event.Interface
func (e *EventOption) UpdateCluster(object runtime.Object, message string) {
	e.eventsCli.Event(object, v1.EventTypeNormal, string(v1alpha2.ClusterConditionUpdating), message)
}

// UpgradedCluster implement the Event.Interface
func (e *EventOption) UpgradedCluster(object runtime.Object, message string) {
	e.eventsCli.Event(object, v1.EventTypeNormal, string(v1alpha2.ClusterConditionUpgrading), message)
}

// EnsureCluster implement the Event.Interface
func (e *EventOption) EnsureCluster(object runtime.Object) {
	e.eventsCli.Event(object, v1.EventTypeNormal, "Ensure", "Makes sure of emqx cluster ready")
}

// CheckCluster implement the Event.Interface
func (e *EventOption) CheckCluster(object runtime.Object) {
	e.eventsCli.Event(object, v1.EventTypeNormal, "CheckAndHeal", "Check and heal the emqx cluster problems")
}

// FailedCluster implement the Event.Interface
func (e *EventOption) FailedCluster(object runtime.Object, message string) {
	e.eventsCli.Event(object, v1.EventTypeWarning, string(v1alpha2.ClusterConditionFailed), message)
}

// HealthCluster implement the Event.Interface
func (e *EventOption) HealthCluster(object runtime.Object) {
	e.eventsCli.Event(object, v1.EventTypeNormal, string(v1alpha2.ClusterConditionHealthy), "EMQ X cluster is healthy")
}
