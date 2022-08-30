/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package apps

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	"github.com/emqx/emqx-operator/pkg/handler"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type requestAPI struct {
	username string
	password string
	port     string
	handler.Handler
}

func (r *requestAPI) getNodeStatuesByAPI(obj client.Object) ([]appsv2alpha1.EMQXNode, error) {
	resp, body, err := r.Handler.RequestAPI(obj, "GET", r.username, r.password, r.port, "api/v5/nodes")
	if err != nil {
		return nil, fmt.Errorf("failed to get listeners: %v", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get listener, status : %s, body: %s", resp.Status, body)
	}

	nodeStatuses := []appsv2alpha1.EMQXNode{}
	if err := json.Unmarshal(body, &nodeStatuses); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node statuses: %v", err)
	}
	return nodeStatuses, nil
}

type emqxGateway struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type emqxListener struct {
	Enable bool   `json:"enable"`
	ID     string `json:"id"`
	Bind   string `json:"bind"`
	Type   string `json:"type"`
}

func (r *requestAPI) getAllListenersByAPI(obj client.Object) ([]corev1.ServicePort, error) {
	ports, err := r.getListenerPortsByAPI(obj, "api/v5/listeners")
	if err != nil {
		return nil, err
	}

	gateways, err := r.getGatewaysByAPI(obj)
	if err != nil {
		return nil, err
	}

	for _, gateway := range gateways {
		if strings.ToLower(gateway.Status) == "running" {
			apiPath := fmt.Sprintf("api/v5/gateway/%s/listeners", gateway.Name)
			gatewayPorts, err := r.getListenerPortsByAPI(obj, apiPath)
			if err != nil {
				return nil, err
			}
			ports = append(ports, gatewayPorts...)
		}
	}

	return ports, nil
}

func (r *requestAPI) getGatewaysByAPI(obj client.Object) ([]emqxGateway, error) {
	resp, body, err := r.Handler.RequestAPI(obj, "GET", r.username, r.password, r.port, "api/v5/gateway")
	if err != nil {
		return nil, fmt.Errorf("failed to get API %s: %v", "api/v5/gateway", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get API %s, status : %s, body: %s", "api/v5/gateway", resp.Status, body)
	}
	gateway := []emqxGateway{}
	if err := json.Unmarshal(body, &gateway); err != nil {
		return nil, fmt.Errorf("failed to parse gateway: %v", err)
	}
	return gateway, nil
}

func (r *requestAPI) getListenerPortsByAPI(obj client.Object, apiPath string) ([]corev1.ServicePort, error) {
	resp, body, err := r.Handler.RequestAPI(obj, "GET", r.username, r.password, r.port, apiPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get API %s: %v", apiPath, err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get API %s, status : %s, body: %s", apiPath, resp.Status, body)
	}
	ports := []corev1.ServicePort{}
	listeners := []emqxListener{}
	if err := json.Unmarshal(body, &listeners); err != nil {
		return nil, fmt.Errorf("failed to parse listeners: %v", err)
	}
	for _, listener := range listeners {
		if !listener.Enable {
			continue
		}

		var protocol corev1.Protocol
		compile := regexp.MustCompile(".*(udp|dtls|quic).*")
		if compile.MatchString(listener.Type) {
			protocol = corev1.ProtocolUDP
		} else {
			protocol = corev1.ProtocolTCP
		}

		_, strPort, _ := net.SplitHostPort(listener.Bind)
		intPort, _ := strconv.Atoi(strPort)

		ports = append(ports, corev1.ServicePort{
			Name:       strings.ReplaceAll(listener.ID, ":", "-"),
			Protocol:   protocol,
			Port:       int32(intPort),
			TargetPort: intstr.FromInt(intPort),
		})
	}
	return ports, nil
}
