package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"testing"

	req "github.com/emqx/emqx-operator/internal/requester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type responseMock struct {
	status       int
	body         string
	requirements []func(map[string]interface{})
}

func respondWith(
	status int,
	body string,
	requirements ...func(map[string]interface{}),
) responseMock {
	return responseMock{status, body, requirements}
}

func mockAPIRequester(t *testing.T, responses map[string]responseMock) *req.MockRequester {
	t.Helper()

	return req.NewMockRequester(
		func(method string, u url.URL, body []byte, header http.Header) (*http.Response, []byte, error) {
			var request map[string]interface{}
			if len(body) > 0 {
				require.NoError(t, json.Unmarshal(body, &request))
			}
			response, ok := responses[method+" "+u.Path]
			if ok {
				for _, requirement := range response.requirements {
					requirement(request)
				}
				return &http.Response{StatusCode: response.status}, []byte(response.body), nil
			}
			return &http.Response{StatusCode: 501}, []byte{}, nil
		},
	)
}

func TestAPIError(t *testing.T) {
	var err error
	_, err = get(
		req.NewMockRequester(
			func(method string, u url.URL, body []byte, header http.Header) (*http.Response, []byte, error) {
				return &http.Response{StatusCode: 503}, []byte("Service Unavailable"), nil
			},
		),
		"api/v5/path",
	)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrorServiceUnavailable))
	assert.False(t, errors.Is(err, ErrorNotFound))

	_, err = delete(
		req.NewMockRequester(
			func(method string, u url.URL, body []byte, header http.Header) (*http.Response, []byte, error) {
				return &http.Response{StatusCode: 404}, []byte("Not Found"), nil
			},
		),
		"api/v5/resource",
	)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrorNotFound))
}
