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

package apiclient

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"

	emperror "emperror.dev/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type PortForwardOptions struct {
	*kubernetes.Clientset
	Config       *restclient.Config
	Namespace    string
	PodName      string
	PodPorts     []string
	StopChannel  chan struct{}
	ReadyChannel chan struct{}

	*portforward.PortForwarder
}

func NewPortForwardOptions(clientset *kubernetes.Clientset, config *rest.Config, pod *corev1.Pod, port string) *PortForwardOptions {
	return &PortForwardOptions{
		Clientset: clientset,
		Config:    config,
		Namespace: pod.Namespace,
		PodName:   pod.Name,
		PodPorts: []string{
			fmt.Sprintf(":%s", port),
		},
		ReadyChannel: make(chan struct{}),
		StopChannel:  make(chan struct{}),
	}
}

func (o *PortForwardOptions) New() error {
	portForwardURL := o.Clientset.
		CoreV1().
		RESTClient().
		Post().
		Resource("pods").
		Namespace(o.Namespace).
		Name(o.PodName).
		SubResource("portforward").
		URL()

	out, errOut := new(bytes.Buffer), new(bytes.Buffer)

	transport, upgrader, err := spdy.RoundTripperFor(o.Config)
	if err != nil {
		return emperror.Wrap(err, "error creating round tripper")
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", portForwardURL)

	fw, err := portforward.New(dialer, o.PodPorts, o.StopChannel, o.ReadyChannel, out, errOut)
	if err != nil {
		return emperror.Wrap(err, "error creating a new PortForwarder with localhost listen addresses")
	}
	o.PortForwarder = fw
	return nil
}

type DoHttpRequest func(username, password, method string, url url.URL, body []byte) (*http.Response, []byte, error)

func (o *PortForwardOptions) Do(username, password, method, path string, body []byte, f DoHttpRequest) (*http.Response, []byte, error) {
	err := o.New()
	if err != nil {
		return nil, nil, emperror.Wrap(err, "failed to create port forward")
	}

	defer close(o.StopChannel)

	errChan := make(chan error)
	go func() {
		if err := o.ForwardPorts(); err != nil {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return nil, nil, err
	case <-o.ReadyChannel:
		ports, err := o.GetPorts()
		if err != nil {
			return nil, nil, err
		}
		if len(ports) == 0 {
			return nil, nil, emperror.Errorf("not found listener port")
		}

		url := url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("localhost:%d", ports[0].Local),
			Path:   path,
		}
		return f(username, password, method, url, body)
	}
}
