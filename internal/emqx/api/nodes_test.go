package api

import (
	"testing"

	req "github.com/emqx/emqx-operator/internal/requester"
	"github.com/stretchr/testify/require"
)

func TestUnreachableNodeReports(t *testing.T) {
	r := req.MockRequests(
		"GET api/v5/nodes", `[{"node":"emqx@silent","node_status":"unreachable"}]`,
		"GET api/v5/nodes/emqx@silent", `{"node":"emqx@silent","node_status":"unreachable"}`,
	)
	want := EMQXNode{Node: "emqx@silent", NodeStatus: NodeStatusUnreachable}
	nodes, err := Nodes(r)
	require.NoError(t, err)
	require.Equal(t, []EMQXNode{want}, nodes)
	node, err := NodeInfo(r, want.Node)
	require.NoError(t, err)
	require.Equal(t, &want, node)
}
