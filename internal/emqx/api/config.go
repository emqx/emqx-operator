package api

import (
	"net/http"

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

func UpdateConfigs(req req.RequesterInterface, config string) error {
	header := http.Header{}
	header.Set("Content-Type", "text/plain")
	_, err := requestWithQuery(req, "PUT", "api/v5/configs", []byte(config), header, "mode=replace")
	if err != nil {
		return err
	}
	return nil
}
