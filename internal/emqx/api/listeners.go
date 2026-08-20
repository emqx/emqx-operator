package api

import (
	"encoding/json"
	"fmt"
	"net/url"

	emperror "emperror.dev/errors"
	req "github.com/emqx/emqx-operator/internal/requester"
)

const GatewayStatusRunning = "running"

// Listener is the subset of an MQTT or gateway listener response needed to
// construct a Kubernetes Service port.
type Listener struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Enable bool   `json:"enable"`
	Bind   string `json:"bind"`
}

// Gateway is the subset of the gateway overview needed for listener discovery.
type Gateway struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func Listeners(requester req.RequesterInterface) ([]Listener, error) {
	body, err := get(requester, "api/v5/listeners")
	if err != nil {
		return nil, err
	}

	var listeners []Listener
	if err := json.Unmarshal(body, &listeners); err != nil {
		return nil, emperror.Wrap(err, "failed to decode MQTT listeners")
	}
	return listeners, nil
}

func Gateways(requester req.RequesterInterface) ([]Gateway, error) {
	body, err := get(requester, "api/v5/gateways")
	if err != nil {
		return nil, err
	}

	var gateways []Gateway
	if err := json.Unmarshal(body, &gateways); err != nil {
		return nil, emperror.Wrap(err, "failed to decode gateway overview")
	}
	return gateways, nil
}

func GatewayListeners(requester req.RequesterInterface, gateway string) ([]Listener, error) {
	path := fmt.Sprintf("api/v5/gateways/%s/listeners", url.PathEscape(gateway))
	body, err := get(requester, path)
	if err != nil {
		return nil, err
	}

	var listeners []Listener
	if err := json.Unmarshal(body, &listeners); err != nil {
		return nil, emperror.Wrapf(err, "failed to decode listeners for gateway %q", gateway)
	}
	return listeners, nil
}
