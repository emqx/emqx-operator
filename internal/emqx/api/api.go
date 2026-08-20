package api

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"syscall"

	emperror "emperror.dev/errors"
	req "github.com/emqx/emqx-operator/internal/requester"
)

type apiError struct {
	StatusCode int
	Message    string
}

var (
	ErrorNotFound           = apiError{StatusCode: 404}
	ErrorServiceUnavailable = apiError{StatusCode: 503}
)

func IsUnavailable(err error) bool {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errno == syscall.ECONNREFUSED || errno == syscall.ECONNABORTED
	}
	return false
}

func IsConnectionClosed(err error) bool {
	return errors.Is(err, net.ErrClosed)
}

func (e apiError) Error() string {
	return fmt.Sprintf("HTTP %d, response: %s", e.StatusCode, e.Message)
}

func (e apiError) Is(target error) bool {
	if target, ok := target.(apiError); ok {
		return e.StatusCode == target.StatusCode
	}
	return false
}

func get(req req.RequesterInterface, path string) ([]byte, error) {
	return request(req, "GET", path, nil, nil)
}

func post(req req.RequesterInterface, path string, body []byte) ([]byte, error) {
	return request(req, "POST", path, body, nil)
}

func delete(req req.RequesterInterface, path string) ([]byte, error) {
	return request(req, "DELETE", path, nil, nil)
}

func request(req req.RequesterInterface, method string, path string, body []byte, header http.Header) ([]byte, error) {
	return requestWithQuery(req, method, path, body, header)
}

func requestWithQuery(req req.RequesterInterface, method string, path string, body []byte, header http.Header, query ...string) ([]byte, error) {
	if req == nil {
		return nil, emperror.New("no requester")
	}
	url := req.GetURL(path, query...)
	resp, body, err := req.Request(method, url, body, header)
	if err != nil {
		return nil, emperror.Wrapf(err, "error accessing %s API %s", req.GetDescription(), url.String())
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		err := apiError{StatusCode: resp.StatusCode, Message: string(body)}
		return nil, emperror.Wrapf(err, "error accessing %s API %s", req.GetDescription(), url.String())
	}
	return body, nil
}
