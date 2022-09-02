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
	"context"
	"net/http"

	emperror "emperror.dev/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type PortForwardOptions struct {
	kubernetes.Clientset
	Namespace    string
	PodName      string
	PodPorts     []string
	Config       *restclient.Config
	StopChannel  chan struct{}
	ReadyChannel chan struct{}

	*portforward.PortForwarder
}

func (o *PortForwardOptions) New() error {
	pod, err := o.Clientset.CoreV1().Pods(o.Namespace).Get(context.TODO(), o.PodName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if pod.Status.Phase != corev1.PodRunning {
		return emperror.Errorf("unable to forward port because pod is not running. Current status=%v", pod.Status.Phase)
	}

	portForwardURL := o.Clientset.
		CoreV1().
		RESTClient().
		Post().
		Resource("pods").
		Namespace(o.Namespace).
		Name(pod.Name).
		SubResource("portforward").
		URL()

	out, errOut := new(bytes.Buffer), new(bytes.Buffer)

	transport, upgrader, err := spdy.RoundTripperFor(o.Config)
	if err != nil {
		return err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", portForwardURL)

	fw, err := portforward.New(dialer, o.PodPorts, o.StopChannel, o.ReadyChannel, out, errOut)
	if err != nil {
		return err
	}
	o.PortForwarder = fw
	return nil
}
