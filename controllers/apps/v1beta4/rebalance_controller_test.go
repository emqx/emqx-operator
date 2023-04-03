package v1beta4

import (
	"errors"
	"net/http"
	"testing"

	innerPortFW "github.com/emqx/emqx-operator/internal/portforward"
	"github.com/stretchr/testify/assert"
)

type fakePW struct {
	requestAPI func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error)
}

func (f *fakePW) GetUsername() string                         { return "" }
func (f *fakePW) GetPassword() string                         { return "" }
func (f *fakePW) GetOptions() *innerPortFW.PortForwardOptions { return nil }
func (f *fakePW) RequestAPI(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
	return f.requestAPI(method, path, body)
}

func TestGetRebalanceStatus(t *testing.T) {
	f := &fakePW{}

	t.Run("check requestAPI args", func(t *testing.T) {
		f.requestAPI = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
			assert.Equal(t, "GET", method)
			assert.Equal(t, "api/v4/load_rebalance/global_status", path)
			assert.Nil(t, body)
			resp = &http.Response{StatusCode: http.StatusOK}
			respBody = []byte(`{"rebalances":[]}`)
			err = nil
			return
		}

		_, err := getRebalanceStatus(f)
		assert.Nil(t, err)
	})

	t.Run("check requestAPI return error", func(t *testing.T) {
		f.requestAPI = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
			return nil, nil, errors.New("fake error")
		}
		_, err := getRebalanceStatus(f)
		assert.Error(t, err, "fake error")
	})

	t.Run("check requestAPI return error status code", func(t *testing.T) {
		f.requestAPI = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
			return &http.Response{StatusCode: http.StatusBadRequest}, nil, nil
		}
		_, err := getRebalanceStatus(f)
		assert.ErrorContains(t, err, "request api failed")
	})

	t.Run("check requestAPI return unexpected JSON", func(t *testing.T) {
		f.requestAPI = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
			return &http.Response{StatusCode: http.StatusOK}, nil, nil
		}
		_, err := getRebalanceStatus(f)
		assert.ErrorContains(t, err, "failed to unmarshal rebalances")
	})

}
