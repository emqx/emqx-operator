package portforward

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"

	emperror "emperror.dev/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type PortForwardOptions struct {
	StopChannel  chan struct{}
	ReadyChannel chan struct{}

	*portforward.PortForwarder
}

func NewPortForwardOptions(clientset *kubernetes.Clientset, config *rest.Config, pod *corev1.Pod, port string) (*PortForwardOptions, error) {
	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, emperror.Wrap(err, "error creating round tripper")
	}

	portForwardURL := clientset.
		CoreV1().
		RESTClient().
		Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("portforward").
		URL()
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", portForwardURL)

	out, errOut := new(bytes.Buffer), new(bytes.Buffer)
	readyChan, stopChan := make(chan struct{}), make(chan struct{})
	fw, err := portforward.New(dialer, []string{fmt.Sprintf(":%s", port)}, stopChan, readyChan, out, errOut)
	if err != nil {
		return nil, emperror.Wrap(err, "error creating a new PortForwarder with localhost listen addresses")
	}

	return &PortForwardOptions{
		ReadyChannel:  readyChan,
		StopChannel:   stopChan,
		PortForwarder: fw,
	}, nil
}

func (o *PortForwardOptions) ForwardPorts() error {
	errChan := make(chan error)
	go func() {
		errChan <- o.PortForwarder.ForwardPorts()
	}()

	select {
	case err := <-errChan:
		return emperror.Wrap(err, "error forwarding ports")
	case <-o.ReadyChannel:
		return nil
	}
}

func (o *PortForwardOptions) RequestAPI(username, password, method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
	ports, err := o.GetPorts()
	if err != nil {
		return nil, nil, emperror.Wrap(err, "failed to get local ports for port forwarding")
	}
	if len(ports) == 0 {
		return nil, nil, emperror.Wrap(err, "no local ports for port forwarding")
	}

	url := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%d", ports[0].Local),
		Path:   path,
	}

	httpClient := http.Client{}
	req, err := http.NewRequest(method, url.String(), bytes.NewReader(body))
	if err != nil {
		return nil, nil, emperror.Wrap(err, "failed to create request")
	}
	req.SetBasicAuth(username, password)
	req.Close = true
	resp, err = httpClient.Do(req)
	if err != nil {
		return nil, nil, emperror.NewWithDetails("failed to request API", "url", url.Path, "method", method)
	}

	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, emperror.Wrap(err, "failed to read response body")
	}
	return resp, body, nil
}
