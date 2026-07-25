package api

import (
	"net/http"
	"net/url"
	"testing"

	req "github.com/emqx/emqx-operator/internal/requester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestQueryPairs(t *testing.T) {
	var actual url.URL
	requester := req.NewMockRequester(
		func(method string, u url.URL, body []byte, header http.Header) (*http.Response, []byte, error) {
			actual = u
			return &http.Response{StatusCode: http.StatusNoContent}, nil, nil
		},
	)

	_, err := request(requester, http.MethodGet, "api/v5/configs", nil, nil, "mode", "replace", "dry_run")
	require.NoError(t, err)
	assert.Equal(t, "dry_run=&mode=replace", actual.RawQuery)
}
