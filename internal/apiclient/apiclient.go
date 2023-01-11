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
	"io"
	"net/http"
	"net/url"

	emperror "emperror.dev/errors"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type APIClient struct {
	Clientset *kubernetes.Clientset
	Config    *rest.Config
}

func NewAPIClient(mgr manager.Manager) *APIClient {
	return &APIClient{
		Clientset: kubernetes.NewForConfigOrDie(mgr.GetConfig()),
		Config:    mgr.GetConfig(),
	}
}

func (a *APIClient) RequestAPI(pod *corev1.Pod, username, password, port, method, path string, body []byte) (*http.Response, []byte, error) {
	o := NewPortForwardOptions(a.Clientset, a.Config, pod, port)
	resp, body, err := o.Do(username, password, method, path, body, doHttpRequest)

	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != 200 {
		return nil, nil, emperror.Errorf("failed to request API: %s, status: %s", path, resp.Status)
	}
	// For EMQX 4.4
	if code := gjson.GetBytes(body, "code"); code.Int() != 0 {
		return nil, nil, emperror.Errorf("failed to request API: %s, code: %d, message: %s", path, code.Int(), gjson.GetBytes(body, "message").String())
	}
	return resp, body, nil
}

func doHttpRequest(username, password, method string, url url.URL, body []byte) (*http.Response, []byte, error) {
	httpClient := http.Client{}
	req, err := http.NewRequest(method, url.String(), bytes.NewReader(body))
	if err != nil {
		return nil, nil, emperror.Wrap(err, "failed to create request")
	}
	req.SetBasicAuth(username, password)
	req.Close = true
	resp, err := httpClient.Do(req)
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
