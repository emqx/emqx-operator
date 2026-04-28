package api

import (
	"encoding/json"
	"fmt"
	"time"

	emperror "emperror.dev/errors"
	req "github.com/emqx/emqx-operator/internal/requester"
)

const apiKeysPath = "api/v5/api_key"

type APIKey struct {
	Name        string `json:"name"`
	APIKey      string `json:"api_key,omitempty"`
	APISecret   string `json:"api_secret,omitempty"`
	Description string `json:"desc,omitempty"`
	Enabled     bool   `json:"enable"`
	Expired     bool   `json:"expired,omitempty"`
	ExpiredAt   string `json:"expired_at,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	Role        string `json:"role,omitempty"`

	// TODO: Currently unused, to be employed in multi-tenancy CRDs.
	// Namespace   string `json:"namespace,omitempty"`
}

type APIKeyRequest struct {
	Name        string
	Description string
	Role        string
	ExpiresAt   *time.Time
}

func GetAPIKey(req req.RequesterInterface, name string) (*APIKey, error) {
	body, err := get(req, apiKeyPath(name))
	if err != nil {
		return nil, err
	}

	apiKey := &APIKey{}
	if err := json.Unmarshal(body, apiKey); err != nil {
		return nil, emperror.Wrap(err, "unexpected API key format")
	}
	return apiKey, nil
}

func CreateAPIKey(req req.RequesterInterface, apiKey APIKeyRequest) (*APIKey, error) {
	body, err := json.Marshal(apiKeyCreateBody(apiKey))
	if err != nil {
		return nil, err
	}

	respBody, err := post(req, apiKeysPath, body)
	if err != nil {
		return nil, err
	}

	created := &APIKey{}
	if err := json.Unmarshal(respBody, created); err != nil {
		return nil, emperror.Wrap(err, "unexpected API key format")
	}
	return created, nil
}

func UpdateAPIKey(req req.RequesterInterface, name string, apiKey APIKeyRequest) (*APIKey, error) {
	body, err := json.Marshal(apiKeyUpdateBody(apiKey))
	if err != nil {
		return nil, err
	}

	respBody, err := request(req, "PUT", apiKeyPath(name), body, nil)
	if err != nil {
		return nil, err
	}

	updated := &APIKey{}
	if err := json.Unmarshal(respBody, updated); err != nil {
		return nil, emperror.Wrap(err, "unexpected API key format")
	}
	return updated, nil
}

func DeleteAPIKey(req req.RequesterInterface, name string) error {
	_, err := delete(req, apiKeyPath(name))
	return err
}

func apiKeyPath(name string) string {
	return fmt.Sprintf("%s/%s", apiKeysPath, name)
}

func apiKeyCreateBody(apiKey APIKeyRequest) map[string]interface{} {
	body := apiKeyUpdateBody(apiKey)
	body["name"] = apiKey.Name
	return body
}

func apiKeyUpdateBody(apiKey APIKeyRequest) map[string]interface{} {
	body := map[string]interface{}{
		"desc":   apiKey.Description,
		"enable": true,
	}
	if apiKey.Role != "" {
		body["role"] = apiKey.Role
	}
	if apiKey.ExpiresAt != nil {
		body["expired_at"] = apiKey.ExpiresAt.Unix()
	}
	return body
}
