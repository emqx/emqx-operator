package portforward

import (
	"bytes"
	emperror "emperror.dev/errors"
	"fmt"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"net/http"
	"net/url"
)

type PortForwardOptions struct {
	stopChan  chan struct{}
	readyChan chan struct{}
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

	// out, errOut := new(bytes.Buffer), new(bytes.Buffer)
	readyChan, stopChan := make(chan struct{}), make(chan struct{})
	fw, err := portforward.New(dialer, []string{fmt.Sprintf(":%s", port)}, stopChan, readyChan, nil, nil)
	if err != nil {
		return nil, emperror.Wrap(err, "error creating a new PortForwarder with localhost listen addresses")
	}

	o := &PortForwardOptions{
		readyChan:     readyChan,
		stopChan:      stopChan,
		PortForwarder: fw,
	}

	err = o.forwardPorts()
	if err != nil {
		o.Close()
		return nil, err
	}
	return o, nil
}

func (o *PortForwardOptions) forwardPorts() error {
	// the select will return after the listener be ready
	// set length to 1 to avoid backup when writing err to the chan
	errChan := make(chan error, 1)
	go func() {
		errChan <- o.PortForwarder.ForwardPorts()
	}()

	select {
	case err := <-errChan:
		close(errChan)
		return err
	case <-o.readyChan:
		// wait for listener ready
		return nil
	}
}

func (o *PortForwardOptions) Close() {
	if o.PortForwarder != nil {
		o.PortForwarder.Close()
		close(o.stopChan)
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
		return nil, nil, emperror.NewWithDetails("failed to request API", "method", method, "path", url.Path)
	}

	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, emperror.Wrap(err, "failed to read response body")
	}
	return resp, body, nil
}
