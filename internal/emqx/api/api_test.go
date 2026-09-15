package api

import (
	"net/http"
	"net/url"
	"testing"

	req "github.com/emqx/emqx-operator/internal/requester"
	"github.com/stretchr/testify/require"
)

func TestHTTPErrorUsesResponseURL(t *testing.T) {
	responseURL := url.URL{Scheme: "http", Host: "fallback:18083", Path: "/api/v5/listeners", RawQuery: "node=a%2Fb"}
	r := req.NewMockRequester(func(method string, u url.URL, body []byte, header http.Header) (*http.Response, []byte, error) {
		require.Equal(t, http.MethodGet, method)
		require.Empty(t, u.Host)
		require.Equal(t, "node=a%2Fb", u.RawQuery)
		return &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Request:    &http.Request{URL: &responseURL},
		}, []byte("unavailable"), nil
	})
	body, err := requestWithQuery(r, http.MethodGet, "/api/v5/listeners", nil, nil, "node=a%2Fb")
	require.Nil(t, body)
	require.ErrorContains(t, err, "error accessing API "+responseURL.String())
	require.ErrorIs(t, err, ErrorServiceUnavailable)
	var httpError apiError
	require.ErrorAs(t, err, &httpError)
	require.Equal(t, "unavailable", httpError.Message)
}
