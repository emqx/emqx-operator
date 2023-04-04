package v1beta4

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	emperror "emperror.dev/errors"
	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
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

func TestStartRebalance(t *testing.T) {
	f := &fakePW{}
	rebalance := &appsv1beta4.Rebalance{
		Spec: appsv1beta4.RebalanceSpec{
			RebalanceStrategy: appsv1beta4.RebalanceStrategy{
				ConnEvictRate:    5,
				SessEvictRate:    5,
				WaitHealthCheck:  10,
				WaitTakeover:     10,
				AbsSessThreshold: 10,
				AbsConnThreshold: 10,
				RelConnThreshold: "1.1",
				RelSessThreshold: "1.1",
			},
		},
	}

	emqx := &appsv1beta4.EmqxEnterprise{
		Status: appsv1beta4.EmqxEnterpriseStatus{
			EmqxNodes: []appsv1beta4.EmqxNode{
				{
					Node: "emqx-ee@emqx-ee-0.emqx-ee-headless.default.svc.cluster.local",
				},
			},
		},
	}

	emqxNodeName := "emqx-ee@emqx-ee-0.emqx-ee-headless.default.svc.cluster.local"

	t.Run("check requestAPI args", func(t *testing.T) {
		startPath := fmt.Sprintf("api/v4/load_rebalance/%s/start", emqxNodeName)
		f.requestAPI = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
			assert.Equal(t, "POST", method)
			assert.Equal(t, startPath, path)
			assert.Equal(t, getRequestBytes(rebalance, []string{emqxNodeName}), body)
			resp = &http.Response{StatusCode: http.StatusOK}
			respBody = []byte(`{"data":[],"code":0}`)
			err = nil
			return
		}

		err := startRebalance(f, rebalance, emqx, emqxNodeName)
		assert.Nil(t, err)
	})

	t.Run("check requestAPI return bad request", func(t *testing.T) {
		f.requestAPI = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
			return &http.Response{StatusCode: http.StatusBadRequest}, nil, nil
		}

		err := startRebalance(f, rebalance, emqx, emqxNodeName)
		assert.ErrorContains(t, err, "request api failed")
	})

	t.Run("check requestAPI start rebalance err", func(t *testing.T) {
		f.requestAPI = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
			resp = &http.Response{StatusCode: http.StatusOK}
			respBody = []byte(`{"message":[\"nothing_to_balance\"],"code":400}`)
			err = emperror.New("[\"nothing_to_balance\"]")
			return
		}
		err := startRebalance(f, rebalance, emqx, emqxNodeName)
		assert.ErrorContains(t, err, "nothing_to_balance")
	})

	t.Run("check requestAPI return error", func(t *testing.T) {
		f.requestAPI = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
			return nil, nil, errors.New("fake error")
		}
		err := startRebalance(f, rebalance, emqx, emqxNodeName)
		assert.Error(t, err, "fake error")
	})
}

func TestStopRebalance(t *testing.T) {
	f := &fakePW{}
	rebalance := &appsv1beta4.Rebalance{
		Status: appsv1beta4.RebalanceStatus{
			Phase: "Processing",
			RebalanceStates: []appsv1beta4.RebalanceState{
				{
					CoordinatorNode: "emqx-ee@emqx-ee-0.emqx-ee-headless.default.svc.cluster.local",
				},
			},
		},
	}

	emqxNodeName := "emqx-ee@emqx-ee-0.emqx-ee-headless.default.svc.cluster.local"

	t.Run("check requestAPI args", func(t *testing.T) {
		stopPath := fmt.Sprintf("api/v4/load_rebalance/%s/stop", emqxNodeName)
		f.requestAPI = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
			assert.Equal(t, "POST", method)
			assert.Equal(t, stopPath, path)
			assert.Nil(t, body)
			resp = &http.Response{StatusCode: http.StatusOK}
			respBody = []byte(`{"data":[],"code":0}`)
			err = nil
			return
		}
		err := stopRebalance(f, rebalance)
		assert.Nil(t, err)
	})

	t.Run("check requestAPI return bad request", func(t *testing.T) {
		f.requestAPI = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
			return &http.Response{StatusCode: http.StatusBadRequest}, nil, nil
		}

		err := stopRebalance(f, rebalance)
		assert.ErrorContains(t, err, "request api failed")
	})

	t.Run("check requestAPI stop rebalance err", func(t *testing.T) {
		f.requestAPI = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
			resp = &http.Response{StatusCode: http.StatusOK}
			respBody = []byte(`{"message": "rebalance is disabled","code":400}`)
			err = nil
			return
		}
		err := stopRebalance(f, rebalance)
		assert.ErrorContains(t, err, "rebalance is disabled")
	})

	t.Run("check requestAPI return error", func(t *testing.T) {
		f.requestAPI = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
			return nil, nil, errors.New("fake error")
		}
		err := stopRebalance(f, rebalance)
		assert.Error(t, err, "fake error")
	})
}

func TestGetRequestBytes(t *testing.T) {
	rebalance := &appsv1beta4.Rebalance{
		Spec: appsv1beta4.RebalanceSpec{
			RebalanceStrategy: appsv1beta4.RebalanceStrategy{
				ConnEvictRate:    5,
				SessEvictRate:    5,
				WaitHealthCheck:  10,
				WaitTakeover:     10,
				AbsSessThreshold: 10,
				AbsConnThreshold: 10,
				RelConnThreshold: "1.1",
				RelSessThreshold: "1.1",
			},
		},
	}

	t.Run("check get request bytes with full rebalanceStrategy", func(t *testing.T) {
		bytes := getRequestBytes(rebalance, []string{})

		body := map[string]interface{}{
			"conn_evict_rate":    rebalance.Spec.RebalanceStrategy.ConnEvictRate,
			"sess_evict_rate":    rebalance.Spec.RebalanceStrategy.SessEvictRate,
			"wait_takeover":      rebalance.Spec.RebalanceStrategy.WaitTakeover,
			"wait_health_check":  rebalance.Spec.RebalanceStrategy.WaitHealthCheck,
			"abs_conn_threshold": rebalance.Spec.RebalanceStrategy.AbsConnThreshold,
			"abs_sess_threshold": rebalance.Spec.RebalanceStrategy.AbsSessThreshold,
			"nodes":              []string{},
		}

		relConnThreshold, _ := strconv.ParseFloat(rebalance.Spec.RebalanceStrategy.RelConnThreshold, 64)
		body["rel_conn_threshold"] = relConnThreshold

		relSessThreshold, _ := strconv.ParseFloat(rebalance.Spec.RebalanceStrategy.RelSessThreshold, 64)
		body["rel_sess_threshold"] = relSessThreshold

		expectedBytes, _ := json.Marshal(body)
		assert.Equal(t, expectedBytes, bytes)
	})

	t.Run("check get request bytes without relConnThreshold", func(t *testing.T) {
		r := rebalance.DeepCopy()
		r.Spec.RebalanceStrategy.RelConnThreshold = ""
		bytes := getRequestBytes(r, []string{})

		body := map[string]interface{}{
			"conn_evict_rate":    rebalance.Spec.RebalanceStrategy.ConnEvictRate,
			"sess_evict_rate":    rebalance.Spec.RebalanceStrategy.SessEvictRate,
			"wait_takeover":      rebalance.Spec.RebalanceStrategy.WaitTakeover,
			"wait_health_check":  rebalance.Spec.RebalanceStrategy.WaitHealthCheck,
			"abs_conn_threshold": rebalance.Spec.RebalanceStrategy.AbsConnThreshold,
			"abs_sess_threshold": rebalance.Spec.RebalanceStrategy.AbsSessThreshold,
			"nodes":              []string{},
		}

		relSessThreshold, _ := strconv.ParseFloat(rebalance.Spec.RebalanceStrategy.RelSessThreshold, 64)
		body["rel_sess_threshold"] = relSessThreshold

		expectedBytes, _ := json.Marshal(body)
		assert.Equal(t, expectedBytes, bytes)
	})

	t.Run("check get request bytes without relSessThreshold", func(t *testing.T) {
		r := rebalance.DeepCopy()
		r.Spec.RebalanceStrategy.RelSessThreshold = ""
		bytes := getRequestBytes(r, []string{})

		body := map[string]interface{}{
			"conn_evict_rate":    rebalance.Spec.RebalanceStrategy.ConnEvictRate,
			"sess_evict_rate":    rebalance.Spec.RebalanceStrategy.SessEvictRate,
			"wait_takeover":      rebalance.Spec.RebalanceStrategy.WaitTakeover,
			"wait_health_check":  rebalance.Spec.RebalanceStrategy.WaitHealthCheck,
			"abs_conn_threshold": rebalance.Spec.RebalanceStrategy.AbsConnThreshold,
			"abs_sess_threshold": rebalance.Spec.RebalanceStrategy.AbsSessThreshold,
			"nodes":              []string{},
		}

		relConnThreshold, _ := strconv.ParseFloat(rebalance.Spec.RebalanceStrategy.RelConnThreshold, 64)
		body["rel_conn_threshold"] = relConnThreshold

		expectedBytes, _ := json.Marshal(body)
		assert.Equal(t, expectedBytes, bytes)
	})

}