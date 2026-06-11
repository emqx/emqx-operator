package requester

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"time"

	emperror "emperror.dev/errors"
)

type HeaderOpt struct {
	Key   string
	Value string
}

type RequesterInterface interface {
	GetURL(path string) url.URL
	GetHost() string
	GetUsername() string
	GetPassword() string
	GetDescription() string
	Request(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error)
}

type Requester struct {
	Schema      string
	Host        string
	Username    string
	Password    string
	Description string
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

func (requester *Requester) GetSchema() string {
	if requester.Schema == "" {
		return "http"
	}
	return requester.Schema
}

func (requester *Requester) GetDescription() string {
	return requester.Description
}

func (requester *Requester) GetURL(path string) url.URL {
	return url.URL{
		Scheme: requester.GetSchema(),
		Host:   requester.GetHost(),
		Path:   path,
	}
}

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	},
}

func (requester *Requester) Request(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
	if url.Scheme == "" {
		url.Scheme = requester.GetSchema()
	}
	if url.Host == "" {
		url.Host = requester.GetHost()
	}

	req, err := http.NewRequest(method, url.String(), bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
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

	resp, err = httpClient.Do(req)
	if err != nil {
		return nil, nil, emperror.Wrap(err, "request failed")
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, emperror.Wrap(err, "response body unreadable")
	}
	return resp, body, nil
}

// Mock
type MockRequesterFunc func(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error)

type MockRequester struct {
	mockFunc MockRequesterFunc
}

func NewMockRequester(mockFunc MockRequesterFunc) *MockRequester {
	return &MockRequester{mockFunc: mockFunc}
}

func (f *MockRequester) GetURL(path string) url.URL { return url.URL{Path: path} }
func (f *MockRequester) GetSchema() string          { return "http" }
func (f *MockRequester) GetHost() string            { return "" }
func (f *MockRequester) GetUsername() string        { return "" }
func (f *MockRequester) GetPassword() string        { return "" }
func (f *MockRequester) GetDescription() string     { return "MockRequester" }

func (f *MockRequester) Request(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
	return f.mockFunc(method, url, body, header)
}
