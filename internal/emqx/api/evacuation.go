package api

import (
	"encoding/json"
	"fmt"
	"strings"

	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v3alpha1"
	req "github.com/emqx/emqx-operator/internal/requester"
	corev1 "k8s.io/api/core/v1"
)

// EMQX node evacuation state values reported by the API.
const EvacuationStateProhibiting = "prohibiting"

const URLAvailabilityCheck = "api/v5/load_rebalance/availability_check"

type nodeEvacuationStatusResponse struct {
	Evacuations []NodeEvacuationStatus `json:"evacuations"`
}

type NodeEvacuationStatus struct {
	Node                   string              `json:"node,omitempty"`
	Stats                  NodeEvacuationStats `json:"stats,omitempty"`
	State                  string              `json:"state,omitempty"`
	SessionRecipients      []string            `json:"session_recipients,omitempty"`
	SessionGoal            int32               `json:"session_goal,omitempty"`
	SessionEvictionRate    int32               `json:"session_eviction_rate,omitempty"`
	ConnectionGoal         int32               `json:"connection_goal,omitempty"`
	ConnectionEvictionRate int32               `json:"connection_eviction_rate,omitempty"`
}

type NodeEvacuationStats struct {
	InitialSessions  int32 `json:"initial_sessions,omitempty"`
	InitialConnected int32 `json:"initial_connected,omitempty"`
	CurrentSessions  int32 `json:"current_sessions,omitempty"`
	CurrentConnected int32 `json:"current_connected,omitempty"`
}

func ClusterEvacuationStatus(req req.RequesterInterface) ([]NodeEvacuationStatus, error) {
	body, err := get(req, "api/v5/load_rebalance/global_status")
	if err != nil {
		return nil, err
	}

	response := nodeEvacuationStatusResponse{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, emperror.Wrap(err, "unexpected node evacuation status format")
	}
	return response.Evacuations, nil
}

func StartEvacuation(
	r req.RequesterInterface,
	strategy crd.EvacuationStrategy,
	migrateTo []string,
	nodeName string,
) error {
	path := fmt.Sprintf("api/v5/load_rebalance/%s/evacuation/start", nodeName)
	request := map[string]interface{}{
		"conn_evict_rate": strategy.ConnEvictRate,
		"sess_evict_rate": strategy.SessEvictRate,
		"migrate_to":      migrateTo,
	}
	if strategy.WaitTakeover > 0 {
		request["wait_takeover"] = strategy.WaitTakeover
	}
	// Specify `wait_health_check` if different from the default.
	if strategy.WaitHealthCheck != 60 {
		request["wait_health_check"] = fmt.Sprintf("%ds", strategy.WaitHealthCheck)
	}

	body, _ := json.Marshal(request)
	_, err := post(r, path, body)

	// TODO:
	// the api/v5/load_rebalance/global_status have some bugs, so we need to ignore the 400 error
	var apiErr apiError
	if ok := emperror.As(err, &apiErr); ok {
		if apiErr.StatusCode == 400 && strings.Contains(apiErr.Message, "already_started") {
			return nil
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func StopEvacuation(
	r req.RequesterInterface,
	nodeName string,
) error {
	path := fmt.Sprintf("api/v5/load_rebalance/%s/evacuation/stop", nodeName)
	_, err := post(r, path, []byte{})
	var apiErr apiError
	if ok := emperror.As(err, &apiErr); ok {
		if apiErr.StatusCode == 400 && strings.Contains(apiErr.Message, "not_started") {
			return nil
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func AvailabilityCheck(req req.RequesterInterface) corev1.ConditionStatus {
	_, err := get(req, URLAvailabilityCheck)
	if err != nil && emperror.Is(err, ErrorServiceUnavailable) {
		return corev1.ConditionFalse
	}
	if err != nil {
		return corev1.ConditionUnknown
	}
	return corev1.ConditionTrue
}
