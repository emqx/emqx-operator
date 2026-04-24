package api

import (
	"net/http"
	"net/url"
	"testing"

	req "github.com/emqx/emqx-operator/internal/requester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockRequesterURLs(responses map[string]string) *req.MockRequester {
	return req.NewMockRequester(
		func(method string, u url.URL, body []byte, header http.Header) (*http.Response, []byte, error) {
			responseBody, ok := responses[u.Path]
			if ok {
				return &http.Response{StatusCode: 200}, []byte(responseBody), nil
			}
			return &http.Response{StatusCode: 501}, []byte{}, nil
		},
	)
}

func TestGetDSReplicationStatusTransitions(t *testing.T) {
	// DB "messages" has a shard with a transition; DB "sessions" has a shard
	// without transitions. Before the fix, `dbStatus` was reused across loop
	// iterations, so "sessions" would inherit "messages"' transitions because
	// the `omitempty` tag causes the field to be absent from the JSON.
	r := mockRequesterURLs(map[string]string{
		"api/v5/ds/storages": `["messages", "sessions"]`,
		"api/v5/ds/storages/messages": `{
			"name": "messages",
			"shards": [{
				"id": "0",
				"replicas": [{"site": "site-a", "status": "up"}],
				"transitions": [{"site": "site-b", "transition": "joining"}]
			}]
		}`,
		"api/v5/ds/storages/sessions": `{
			"name": "sessions",
			"shards": [{
				"id": "0",
				"replicas": [{"site": "site-a", "status": "up"}]
			}]
		}`,
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
	r := mockRequesterURLs(map[string]string{
		"api/v5/ds/sites": `["site-a", "site-b"]`,
		"api/v5/ds/sites/site-a": `{
			"node": "emqx@node1",
			"up": true,
			"shards": [{"storage": "messages", "id": "0", "status": "up"}]
		}`,
		"api/v5/ds/sites/site-b": `{
			"node": "emqx@node2",
			"up": false,
			"shards": [{"storage": "messages", "id": "0", "status": "joining"}]
		}`,
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
	r := mockRequesterURLs(map[string]string{})
	_, err = GetDSReplicationStatus(r)
	assert.Error(t, err)
	_, err = GetDSCluster(r)
	assert.Error(t, err)
}
