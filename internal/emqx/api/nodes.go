package api

import (
	"encoding/json"
	"fmt"

	emperror "emperror.dev/errors"
	req "github.com/emqx/emqx-operator/internal/requester"
)

type EMQXNode struct {
	// EMQX node name, example: emqx@127.0.0.1
	Node string `json:"node,omitempty"`
	// EMQX node status, example: Running
	NodeStatus string `json:"node_status,omitempty"`
	// Erlang/OTP version used by EMQX, example: 24.2/12.2
	OTPRelease string `json:"otp_release,omitempty"`
	// EMQX version
	Version string `json:"version,omitempty"`
	// EMQX cluster node role, enum: "core" "replicant"
	Role string `json:"role,omitempty"`
	// Number of MQTT sessions
	Connections int64 `json:"connections,omitempty"`
	// Number of connected MQTT clients
	LiveConnections int64 `json:"live_connections,omitempty"`
	// EMQX node uptime, milliseconds
	Uptime int64 `json:"uptime,omitempty"`
}

func NodeInfo(req req.RequesterInterface, nodeName string) (*EMQXNode, error) {
	body, err := get(req, fmt.Sprintf("api/v5/nodes/%s", nodeName))
	if emperror.Is(err, ErrorNotFound) {
		return &EMQXNode{
			Node:       nodeName,
			NodeStatus: "stopped",
		}, nil
	}
	if err != nil {
		return nil, err
	}

	nodeInfo := &EMQXNode{}
	if err := json.Unmarshal(body, &nodeInfo); err != nil {
		return nil, emperror.Wrap(err, "unexpected node info format")
	}
	return nodeInfo, nil
}

// ForceLeave removes a node from the EMQX cluster.
// DELETE /api/v5/cluster/{node}/force_leave
func ForceLeave(req req.RequesterInterface, nodeName string) error {
	path := fmt.Sprintf("api/v5/cluster/%s/force_leave", nodeName)
	_, err := delete(req, path)
	if emperror.Is(err, ErrorNotFound) {
		return nil
	}
	return err
}

func Nodes(req req.RequesterInterface) ([]EMQXNode, error) {
	body, err := get(req, "api/v5/nodes")
	if err != nil {
		return nil, err
	}

	nodeInfos := []EMQXNode{}
	if err := json.Unmarshal(body, &nodeInfos); err != nil {
		return nil, emperror.Wrap(err, "unexpected node info format")
	}
	return nodeInfos, nil
}
