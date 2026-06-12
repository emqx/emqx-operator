package api

import (
	"net/http"
	"strings"

	req "github.com/emqx/emqx-operator/internal/requester"
)

func Configs(req req.RequesterInterface) (string, error) {
	header := http.Header{}
	header.Set("Accept", "text/plain")
	body, err := request(req, "GET", "api/v5/configs", nil, header)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func UpdateConfigs(req req.RequesterInterface, mode, config string) error {
	query := []string{"ignore_readonly", "true"}
	if mode != "" {
		query = append(query, "mode", strings.ToLower(mode))
	}
	header := http.Header{}
	header.Set("Content-Type", "text/plain")
	_, err := request(req, "PUT", "api/v5/configs", []byte(config), header, query...)
	if err != nil {
		return err
	}
	return nil
}
