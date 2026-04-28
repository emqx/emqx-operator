package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDSReplicationStatusTransitions(t *testing.T) {
	// DB "messages" has a shard with a transition; DB "sessions" has a shard
	// without transitions. Before the fix, `dbStatus` was reused across loop
	// iterations, so "sessions" would inherit "messages"' transitions because
	// the `omitempty` tag causes the field to be absent from the JSON.
	r := mockAPIRequester(t, map[string]responseMock{
		"GET api/v5/ds/storages": respondWith(200, `["messages", "sessions"]`),
		"GET api/v5/ds/storages/messages": respondWith(200, `{
			"name": "messages",
			"shards": [{
				"id": "0",
				"replicas": [{"site": "site-a", "status": "up"}],
				"transitions": [{"site": "site-b", "transition": "joining"}]
			}]
		}`),
		"GET api/v5/ds/storages/sessions": respondWith(200, `{
			"name": "sessions",
			"shards": [{
				"id": "0",
				"replicas": [{"site": "site-a", "status": "up"}]
			}]
		}`),
	})

	status, err := GetDSReplicationStatus(r)
	require.NoError(t, err)
	require.Len(t, status.DBs, 2)

	messages := status.DBs[0]
	sessions := status.DBs[1]

	assert.Equal(t, "messages", messages.Name)
	assert.Len(t, messages.Shards[0].Transitions, 1, "messages should have 1 transition")

	assert.Equal(t, "sessions", sessions.Name)
	assert.Empty(t, sessions.Shards[0].Transitions,
		"sessions must NOT inherit transitions from messages (omitempty bug)")
}

func TestGetDSCluster(t *testing.T) {
	r := mockAPIRequester(t, map[string]responseMock{
		"GET api/v5/ds/sites": respondWith(200, `["site-a", "site-b"]`),
		"GET api/v5/ds/sites/site-a": respondWith(200, `{
			"node": "emqx@node1",
			"up": true,
			"shards": [{"storage": "messages", "id": "0", "status": "up"}]
		}`),
		"GET api/v5/ds/sites/site-b": respondWith(200, `{
			"node": "emqx@node2",
			"up": false,
			"shards": [{"storage": "messages", "id": "0", "status": "joining"}]
		}`),
	})

	cluster, err := GetDSCluster(r)
	require.NoError(t, err)
	require.Len(t, cluster.Sites, 2)

	assert.Equal(t, "site-a", cluster.Sites[0].ID)
	assert.Equal(t, "emqx@node1", cluster.Sites[0].Node)
	assert.True(t, cluster.Sites[0].Up)

	assert.Equal(t, "site-b", cluster.Sites[1].ID)
	assert.False(t, cluster.Sites[1].Up)
}

func TestGetDSAPIError(t *testing.T) {
	var err error
	r := mockAPIRequester(t, map[string]responseMock{})
	_, err = GetDSReplicationStatus(r)
	assert.Error(t, err)
	_, err = GetDSCluster(r)
	assert.Error(t, err)
}
