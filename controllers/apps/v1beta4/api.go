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

func (r *EmqxReconciler) requestAPI(instance appsv1beta4.Emqx, method, apiPort, path string, body []byte) (*http.Response, []byte, error) {
	inCluster := true
	if path == "api/v4/nodes" && instance.GetStatus().GetEmqxNodes() == nil {
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
	pod := podMap[latestReadySts.UID][0]

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
	resp, body, err := apiClient.Do(method, path, body)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != 200 {
		return nil, nil, emperror.Errorf("failed to request API: %s, status: %s", path, resp.Status)
	}
	if code := gjson.GetBytes(body, "code"); code.Int() != 0 {
		return nil, nil, emperror.Errorf("failed to request API: %s, code: %d, message: %s", path, code.Int(), gjson.GetBytes(body, "message").String())
	}
	return resp, body, nil
}

func (r *EmqxReconciler) getNodeStatusesByAPI(instance appsv1beta4.Emqx) ([]appsv1beta4.EmqxNode, error) {
	_, body, err := r.requestAPI(instance, "GET", "8081", "api/v4/nodes", nil)
	if err != nil {
		return nil, err
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

	_, body, err := r.requestAPI(instance, "GET", "8081", "api/v4/listeners", nil)
	if err != nil {
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

func (r *EmqxReconciler) getEvacuationStatusByAPI(instance appsv1beta4.Emqx) ([]appsv1beta4.EmqxEvacuationStatus, error) {
	_, body, err := r.requestAPI(instance, "GET", "8081", "api/v4/load_rebalance/global_status", nil)
	if err != nil {
		return nil, err
	}

	evacuationStatuses := []appsv1beta4.EmqxEvacuationStatus{}
	data := gjson.GetBytes(body, "evacuations")
	if err := json.Unmarshal([]byte(data.Raw), &evacuationStatuses); err != nil {
		return nil, emperror.Wrap(err, "failed to unmarshal node statuses")
	}
	return evacuationStatuses, nil
}

func (r *EmqxReconciler) evacuateNodeByAPI(instance appsv1beta4.Emqx, migrateToPods []*corev1.Pod, nodeName string) error {
	enterprise, ok := instance.(*appsv1beta4.EmqxEnterprise)
	if !ok {
		return emperror.New("failed to evacuate node, only support emqx enterprise")
	}

	migrateTo := []string{}
	for _, pod := range migrateToPods {
		emqxNodeName := getEmqxNodeName(instance, pod)
		migrateTo = append(migrateTo, emqxNodeName)
	}

	body := map[string]interface{}{
		"conn_evict_rate": enterprise.Spec.EmqxBlueGreenUpdate.EvacuationStrategy.ConnEvictRate,
		"sess_evict_rate": enterprise.Spec.EmqxBlueGreenUpdate.EvacuationStrategy.SessEvictRate,
		"migrate_to":      migrateTo,
	}
	if enterprise.Spec.EmqxBlueGreenUpdate.EvacuationStrategy.WaitTakeover > 0 {
		body["wait_takeover"] = enterprise.Spec.EmqxBlueGreenUpdate.EvacuationStrategy.WaitTakeover
	}

	b, err := json.Marshal(body)
	if err != nil {
		return emperror.Wrap(err, "marshal body failed")
	}

	_, _, err = r.requestAPI(instance, "POST", "8081", "api/v4/load_rebalance/"+nodeName+"/evacuation/start", b)
	return err
}
