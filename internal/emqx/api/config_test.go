package api

import (
	"net/http"
	"net/url"
	"testing"

	req "github.com/emqx/emqx-operator/internal/requester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateConfigsModeQuery(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		expected string
	}{
		{name: "empty mode",
			mode: "", expected: "ignore_readonly=true"},
		{name: "lowercase mode",
			mode: "Replace", expected: "ignore_readonly=true&mode=replace"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actual url.URL
			requester := req.NewMockRequester(
				func(method string, u url.URL, body []byte, header http.Header) (*http.Response, []byte, error) {
					actual = u
					assert.Equal(t, http.MethodPut, method)
					assert.Equal(t, "api/v5/configs", u.Path)
					assert.Equal(t, "text/plain", header.Get("Content-Type"))
					assert.Equal(t, "config", string(body))
					return &http.Response{StatusCode: http.StatusNoContent}, nil, nil
				},
			)

			err := UpdateConfigs(requester, tt.mode, "config")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, actual.RawQuery)
		})
	}
}
