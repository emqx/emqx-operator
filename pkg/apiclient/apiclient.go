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
	"fmt"
	"io"
	"net/http"
	"net/url"

	emperror "emperror.dev/errors"
)

type APIClient struct {
	Username string
	Password string
	PortForwardOptions
}

func (c *APIClient) Do(method, path string) (*http.Response, []byte, error) {
	err := c.PortForwardOptions.New()
	if err != nil {
		return nil, nil, err
	}

	defer close(c.StopChannel)

	errChan := make(chan error)
	go func() {
		if err := c.ForwardPorts(); err != nil {
			errChan <- err
			// panic(err)
		}
	}()

	select {
	case err := <-errChan:
		return nil, nil, err
	case <-c.PortForwardOptions.ReadyChannel:
		ports, err := c.GetPorts()
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

		httpClient := http.Client{}
		req, err := http.NewRequest(method, url.String(), nil)
		if err != nil {
			return nil, nil, err
		}
		req.SetBasicAuth(c.Username, c.Password)
		req.Close = true
		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, nil, err
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return resp, nil, err
		}
		return resp, body, err
	}
}
