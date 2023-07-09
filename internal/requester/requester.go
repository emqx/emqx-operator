package requester

import (
	"bytes"
	"io"
	"net/http"
	"net/url"

	emperror "emperror.dev/errors"
)

type HeaderOpt struct {
	Key   string
	Value string
}

type RequesterInterface interface {
	GetURL(path string, query ...string) url.URL
	GetHost() string
	GetUsername() string
	GetPassword() string
	Request(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error)
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

func (requester *Requester) GetURL(path string, query ...string) url.URL {
	url := url.URL{
		Scheme: "http",
		Host:   requester.GetHost(),
		Path:   path,
	}
	for _, q := range query {
		if url.RawQuery == "" {
			url.RawQuery = q
			continue
		}
		url.RawQuery += "&" + q
	}
	return url
}

func (requester *Requester) Request(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
	if url.Scheme == "" {
		url.Scheme = "http"
	}
	if url.Host == "" {
		url.Host = requester.Host
	}

	req, err := http.NewRequest(method, url.String(), bytes.NewReader(body))
	if err != nil {
		return nil, nil, emperror.Wrap(err, "failed to create request")
	}

	for k, v := range header {
		req.Header[k] = v
	}
	if req.Header.Get("Authorization") == "" {
		req.SetBasicAuth(requester.GetUsername(), requester.GetPassword())
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	req.Close = true

	httpClient := http.Client{}
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
	ReqFunc func(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error)
}

func (f *FakeRequester) GetURL(path string, query ...string) url.URL { return url.URL{Path: path} }
func (f *FakeRequester) GetHost() string                             { return "" }
func (f *FakeRequester) GetUsername() string                         { return "" }
func (f *FakeRequester) GetPassword() string                         { return "" }
func (f *FakeRequester) Request(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
	return f.ReqFunc(method, url, body, header)
}
