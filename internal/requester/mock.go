package requester

import (
	"fmt"
	"net/http"
	"net/url"

	emperror "emperror.dev/errors"
)

type MockRequester interface {
	Mock(method string, url url.URL, body []byte, header http.Header) (
		resp *http.Response, respBody []byte, err error,
	)
}

type mockedRequests map[string]mockedResponse

func (rs mockedRequests) Mock(method string, url url.URL, body []byte, header http.Header) (
	*http.Response, []byte, error,
) {
	mresp, ok := rs[method+" "+url.Path]
	if ok {
		return mresp.resp()
	}
	mresp, ok = rs["*"]
	if ok {
		return mresp.resp()
	}
	return &http.Response{StatusCode: http.StatusNotImplemented}, nil, nil
}

type mockedResponse interface {
	resp() (resp *http.Response, respBody []byte, err error)
}

func MockOK(body string) mockedResponse {
	return mockHTTPResponse{http.StatusOK, []byte(body)}
}

func MockUnavail() mockedResponse {
	return mockHTTPResponse{http.StatusServiceUnavailable, []byte{}}
}

type mockHTTPResponse struct {
	status int
	body   []byte
}

func (m mockHTTPResponse) resp() (*http.Response, []byte, error) {
	return &http.Response{StatusCode: m.status}, m.body, nil
}

func MockError(err error) mockError {
	return mockError{err}
}

type mockError struct {
	err error
}

func (m mockError) resp() (*http.Response, []byte, error) {
	return nil, nil, m.err
}

type MockRequesterFunction func(method string, url url.URL, body []byte, header http.Header) (
	resp *http.Response, respBody []byte, err error,
)

func (f MockRequesterFunction) Mock(method string, url url.URL, body []byte, header http.Header) (
	*http.Response, []byte, error,
) {
	return f(method, url, body, header)
}

type mockRequester struct {
	m MockRequester
}

func NewMockRequester(f MockRequesterFunction) RequesterInterface {
	return &mockRequester{f}
}

func MockRequests(args ...any) RequesterInterface {
	requests := make(mockedRequests, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			panic(fmt.Sprintf("argument %d must be a string, got %T", i, args[i]))
		}
		switch value := args[i+1].(type) {
		case string:
			requests[key] = MockOK(value)
		case error:
			requests[key] = MockError(value)
		case mockedResponse:
			requests[key] = value
		default:
			panic(fmt.Sprintf("argument %d must be a string, error, or mocked response, got %T", i+1, args[i+1]))
		}
	}
	return &mockRequester{requests}
}

func (f *mockRequester) Request(method string, url url.URL, body []byte, header http.Header) (
	*http.Response, []byte, error,
) {
	var resp *http.Response
	respMock, respBody, err := f.m.Mock(method, url, body, header)
	if respMock != nil {
		// Match http.Client responses so API errors can use Request.URL.
		// Copy to avoid mutating a reused mock response; preserve any explicit request.
		respCopy := *respMock
		resp = &respCopy
		if respMock.Request == nil {
			resp.Request = &http.Request{Method: method, URL: &url}
		}
	}
	return resp, respBody, emperror.Wrapf(err, "error accessing MockRequester API %s", url.String())
}
