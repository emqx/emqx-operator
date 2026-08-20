package requester

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockedRequests(t *testing.T) {
	requestError := errors.New("request failed")
	r := MockRequests(
		"GET api/v5/ok", "response body",
		"GET api/v5/error", requestError,
		"*", MockUnavail(),
	)

	resp, body, err := r.Request(http.MethodGet, r.GetURL("api/v5/ok"), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, []byte("response body"), body)

	resp, body, err = r.Request(http.MethodGet, r.GetURL("api/v5/error"), nil, nil)
	assert.Nil(t, resp)
	assert.Nil(t, body)
	assert.ErrorIs(t, err, requestError)

	resp, body, err = r.Request(http.MethodGet, r.GetURL("api/v5/missing"), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	assert.Empty(t, body)
}
