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

package v1beta4

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	emperror "emperror.dev/errors"
	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	apiClient "github.com/emqx/emqx-operator/pkg/apiclient"
	"github.com/tidwall/gjson"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *EmqxReconciler) requestAPI(instance appsv1beta4.Emqx, method, apiPort, path string) (*http.Response, []byte, error) {
	var pod *corev1.Pod
	inCluster := true
	if path == "api/v4/nodes" {
		inCluster = false
	}
	latestReadySts, err := r.getLatestReadyStatefulSet(instance, inCluster)
	if err != nil {
		return nil, nil, err
	}
	podMap, err := r.getPodMap(instance, []*appsv1.StatefulSet{latestReadySts})
	if err != nil {
		return nil, nil, err
	}
	pod = podMap[latestReadySts.UID][0]

	if pod == nil {
		return nil, nil, emperror.Errorf("no running pod found for emqx %s", instance.GetName())
	}

	username, password, err := r.Handler.GetBootstrapUser(instance)
	if err != nil {
		return nil, nil, err
	}

	stopChan, readyChan := make(chan struct{}, 1), make(chan struct{}, 1)

	apiClient := apiClient.APIClient{
		Username: username,
		Password: password,
		PortForwardOptions: apiClient.PortForwardOptions{
			Namespace: pod.Namespace,
			PodName:   pod.Name,
			PodPorts: []string{
				fmt.Sprintf(":%s", apiPort),
			},
			Clientset:    r.clientset,
			Config:       r.config,
			ReadyChannel: readyChan,
			StopChannel:  stopChan,
		},
	}

	return apiClient.Do(method, path)
}

func (r *EmqxReconciler) getNodeStatusesByAPI(instance appsv1beta4.Emqx) ([]appsv1beta4.EmqxNode, error) {
	resp, body, err := r.requestAPI(instance, "GET", "8081", "api/v4/nodes")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, emperror.Errorf("failed to get node statuses from API: %s", resp.Status)
	}

	emqxNodes := []appsv1beta4.EmqxNode{}
	data := gjson.GetBytes(body, "data")
	if err := json.Unmarshal([]byte(data.Raw), &emqxNodes); err != nil {
		return nil, emperror.Wrap(err, "failed to unmarshal node statuses")
	}
	return emqxNodes, nil
}

func (r *EmqxReconciler) getListenerPortsByAPI(instance appsv1beta4.Emqx) ([]corev1.ServicePort, error) {
	type emqxListener struct {
		Protocol string `json:"protocol"`
		ListenOn string `json:"listen_on"`
	}

	type emqxListeners struct {
		Node      string         `json:"node"`
		Listeners []emqxListener `json:"listeners"`
	}

	intersection := func(listeners1 []emqxListener, listeners2 []emqxListener) []emqxListener {
		hSection := map[string]struct{}{}
		ans := make([]emqxListener, 0)
		for _, listener := range listeners1 {
			hSection[listener.ListenOn] = struct{}{}
		}
		for _, listener := range listeners2 {
			_, ok := hSection[listener.ListenOn]
			if ok {
				ans = append(ans, listener)
				delete(hSection, listener.ListenOn)
			}
		}
		return ans
	}

	resp, body, err := r.requestAPI(instance, "GET", "8081", "api/v4/listeners")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, err
	}

	listenerList := []emqxListeners{}
	data := gjson.GetBytes(body, "data")
	if err := json.Unmarshal([]byte(data.Raw), &listenerList); err != nil {
		return nil, emperror.Wrap(err, "failed to unmarshal node statuses")
	}

	var listeners []emqxListener
	if len(listenerList) == 1 {
		listeners = listenerList[0].Listeners
	} else {
		for i := 0; i < len(listenerList)-1; i++ {
			listeners = intersection(listenerList[i].Listeners, listenerList[i+1].Listeners)
		}
	}

	ports := []corev1.ServicePort{}
	for _, l := range listeners {
		var name string
		var protocol corev1.Protocol
		var strPort string
		var intPort int

		compile := regexp.MustCompile(".*(udp|dtls|sn).*")
		if compile.MatchString(l.Protocol) {
			protocol = corev1.ProtocolUDP
		} else {
			protocol = corev1.ProtocolTCP
		}

		if strings.Contains(l.ListenOn, ":") {
			_, strPort, err = net.SplitHostPort(l.ListenOn)
			if err != nil {
				strPort = l.ListenOn
			}
		} else {
			strPort = l.ListenOn
		}
		intPort, _ = strconv.Atoi(strPort)

		// Get name by protocol and port from API
		// protocol maybe like mqtt:wss:8084
		// protocol maybe like mqtt:tcp
		// We had to do something with the "protocol" to make it conform to the kubernetes service port name specification
		name = regexp.MustCompile(`:[\d]+`).ReplaceAllString(l.Protocol, "")
		name = strings.ReplaceAll(name, ":", "-")
		name = fmt.Sprintf("%s-%s", name, strPort)

		ports = append(ports, corev1.ServicePort{
			Name:       name,
			Protocol:   protocol,
			Port:       int32(intPort),
			TargetPort: intstr.FromInt(intPort),
		})
	}
	return ports, nil
}
