package requester

import (
	"bytes"
	"io"
	"net/http"
	"net/url"

	emperror "emperror.dev/errors"
)

type RequesterInterface interface {
	GetHost() string
	GetUsername() string
	GetPassword() string
	Request(method, path string, body []byte) (resp *http.Response, respBody []byte, err error)
}

type Requester struct {
	Host     string
	Username string
	Password string
}

func (requester *Requester) GetUsername() string {
	return requester.Username
}

func (requester *Requester) GetPassword() string {
	return requester.Password
}

func (requester *Requester) GetHost() string {
	return requester.Host
}

func (requester *Requester) Request(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
	url := url.URL{
		Scheme: "http",
		Host:   requester.GetHost(),
		Path:   path,
	}

	httpClient := http.Client{}
	req, err := http.NewRequest(method, url.String(), bytes.NewReader(body))
	if err != nil {
		return nil, nil, emperror.Wrap(err, "failed to create request")
	}
	req.SetBasicAuth(requester.GetUsername(), requester.GetPassword())
	req.Header.Set("Content-Type", "application/json")
	req.Close = true
	resp, err = httpClient.Do(req)
	if err != nil {
		return nil, nil, emperror.Wrap(err, "failed to request API")
	}

	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, emperror.Wrap(err, "failed to read response body")
	}
	return resp, body, nil
}

// Mock
type FakeRequester struct {
	ReqFunc func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error)
}

func (f *FakeRequester) GetHost() string     { return "" }
func (f *FakeRequester) GetUsername() string { return "" }
func (f *FakeRequester) GetPassword() string { return "" }
func (f *FakeRequester) Request(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
	return f.ReqFunc(method, path, body)
}
