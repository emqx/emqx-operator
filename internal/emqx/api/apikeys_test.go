package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIKeysCRUD(t *testing.T) {
	r := mockAPIRequester(t, map[string]responseMock{
		"GET api/v5/api_key/operator": respondWith(
			200,
			`{"name":"operator","api_key":"key","desc":"managed","enable":true,"expired_at":"infinity","role":"administrator"}`,
		),
		"POST api/v5/api_key": respondWith(
			200,
			`{"name":"operator","api_key":"key","api_secret":"secret","desc":"managed","enable":true,"expired_at":"2030-01-01T00:00:00Z","role":"administrator"}`,
		),
		"PUT api/v5/api_key/operator": respondWith(
			200,
			`{"name":"operator","api_key":"key","desc":"updated","enable":true,"expired_at":"2030-01-01T00:00:00Z","role":"viewer"}`,
		),
		"DELETE api/v5/api_key/operator": respondWith(204, ""),
	})

	apiKey, err := GetAPIKey(r, "operator")
	require.NoError(t, err)
	assert.Equal(t, "key", apiKey.APIKey)

	expiresAt := time.Unix(1893456000, 0).UTC()
	created, err := CreateAPIKey(r, APIKeyRequest{
		Name:        "operator",
		Description: "managed",
		Role:        "administrator",
		ExpiresAt:   &expiresAt,
	})
	require.NoError(t, err)
	assert.Equal(t, "secret", created.APISecret)

	updated, err := UpdateAPIKey(r, "operator", APIKeyRequest{
		Description: "updated",
		Role:        "viewer",
		ExpiresAt:   &expiresAt,
	})
	require.NoError(t, err)
	assert.Equal(t, "viewer", updated.Role)

	require.NoError(t, DeleteAPIKey(r, "operator"))
}

func TestAPIKeyCreateNeverExpires(t *testing.T) {
	r := mockAPIRequester(t, map[string]responseMock{
		"POST api/v5/api_key": respondWith(
			200,
			`{"name":"operator","api_key":"key","api_secret":"secret","desc":"","enable":true,"expired_at":"infinity","role":"administrator"}`,
			func(request map[string]interface{}) {
				require.NotContains(t, request, "expired_at")
			},
		),
	})

	_, err := CreateAPIKey(r, APIKeyRequest{Name: "operator"})
	require.NoError(t, err)
}
