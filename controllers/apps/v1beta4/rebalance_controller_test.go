package v1beta4

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type fakeEmqxHttpAPI struct {
	request func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error)
	Pod     *corev1.Pod
}

func (f *fakeEmqxHttpAPI) GetUsername() string { return "" }
func (f *fakeEmqxHttpAPI) GetPassword() string { return "" }
func (f *fakeEmqxHttpAPI) GetPod() *corev1.Pod { return f.Pod }
func (f *fakeEmqxHttpAPI) Request(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
	return f.request(method, path, body)
}

func TestGetRebalanceStatus(t *testing.T) {
	f := &fakeEmqxHttpAPI{}

	t.Run("check request args", func(t *testing.T) {
		f.request = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
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

	t.Run("check request return error", func(t *testing.T) {
		f.request = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
			return nil, nil, errors.New("fake error")
		}
		_, err := getRebalanceStatus(f)
		assert.Error(t, err, "fake error")
	})

	t.Run("check request return error status code", func(t *testing.T) {
		f.request = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
			return &http.Response{StatusCode: http.StatusBadRequest}, nil, nil
		}
		_, err := getRebalanceStatus(f)
		assert.ErrorContains(t, err, "request api failed")
	})

	t.Run("check request return unexpected JSON", func(t *testing.T) {
		f.request = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
			return &http.Response{StatusCode: http.StatusOK}, nil, nil
		}
		_, err := getRebalanceStatus(f)
		assert.ErrorContains(t, err, "failed to unmarshal rebalances")
	})
}

func TestStartRebalance(t *testing.T) {
	f := &fakeEmqxHttpAPI{
		Pod: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "emqx-ee-0",
			},
		},
	}
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

	emqxNodeName := "emqx-ee@emqx-ee-0.emqx-ee-headless.default.svc.cluster.local"
	emqx := &appsv1beta4.EmqxEnterprise{
		Spec: appsv1beta4.EmqxEnterpriseSpec{
			Template: appsv1beta4.EmqxTemplate{
				Spec: appsv1beta4.EmqxTemplateSpec{
					EmqxContainer: appsv1beta4.EmqxContainer{
						EmqxConfig: map[string]string{
							"name":             "emqx-ee",
							"cluster.dns.name": "emqx-ee-headless.default.svc.cluster.local",
						},
					},
				},
			},
		},
		Status: appsv1beta4.EmqxEnterpriseStatus{
			EmqxNodes: []appsv1beta4.EmqxNode{
				{
					Node: emqxNodeName,
				},
			},
		},
	}

	t.Run("check request args", func(t *testing.T) {
		startPath := fmt.Sprintf("api/v4/load_rebalance/%s/start", emqxNodeName)
		f.request = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
			assert.Equal(t, "POST", method)
			assert.Equal(t, startPath, path)
			assert.Equal(t, getRequestBytes(rebalance, []string{emqxNodeName}), body)
			resp = &http.Response{StatusCode: http.StatusOK}
			respBody = []byte(`{"data":[],"code":0}`)
			err = nil
			return
		}

		err := startRebalance(f, rebalance, emqx)
		assert.Nil(t, err)
	})

	t.Run("check request return bad request", func(t *testing.T) {
		f.request = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
			return &http.Response{StatusCode: http.StatusBadRequest}, nil, nil
		}

		err := startRebalance(f, rebalance, emqx)
		assert.ErrorContains(t, err, "request api failed")
	})

	t.Run("check request start rebalance err", func(t *testing.T) {
		f.request = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
			resp = &http.Response{StatusCode: http.StatusOK}
			respBody = []byte(`{"message":"fake error","code":400}`)
			err = nil
			return
		}
		err := startRebalance(f, rebalance, emqx)
		assert.ErrorContains(t, err, "fake error")
	})

	t.Run("check request return error", func(t *testing.T) {
		f.request = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
			return nil, nil, errors.New("fake error")
		}
		err := startRebalance(f, rebalance, emqx)
		assert.Error(t, err, "fake error")
	})
}

func TestStopRebalance(t *testing.T) {
	f := &fakeEmqxHttpAPI{}
	emqxNodeName := "emqx-ee@emqx-ee-0.emqx-ee-headless.default.svc.cluster.local"
	rebalance := &appsv1beta4.Rebalance{
		Status: appsv1beta4.RebalanceStatus{
			Phase: "Processing",
			RebalanceStates: []appsv1beta4.RebalanceState{
				{
					CoordinatorNode: emqxNodeName,
				},
			},
		},
	}

	t.Run("check requestAPI args", func(t *testing.T) {
		stopPath := fmt.Sprintf("api/v4/load_rebalance/%s/stop", emqxNodeName)
		f.request = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
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

	t.Run("check request return bad request", func(t *testing.T) {
		f.request = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
			return &http.Response{StatusCode: http.StatusBadRequest}, nil, nil
		}

		err := stopRebalance(f, rebalance)
		assert.ErrorContains(t, err, "request api failed")
	})

	t.Run("check request stop rebalance err", func(t *testing.T) {
		f.request = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
			resp = &http.Response{StatusCode: http.StatusOK}
			respBody = []byte(`{"message": "rebalance is disabled","code":400}`)
			err = nil
			return
		}
		err := stopRebalance(f, rebalance)
		assert.ErrorContains(t, err, "rebalance is disabled")
	})

	t.Run("check request return error", func(t *testing.T) {
		f.request = func(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
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

func TestRebalanceStatusHandler(t *testing.T) {
	finalizer := "apps.emqx.io/finalizer"
	rebalance := &appsv1beta4.Rebalance{
		ObjectMeta: metav1.ObjectMeta{
			Finalizers: []string{finalizer},
		},
	}
	emqxEnterprise := &appsv1beta4.EmqxEnterprise{}
	emqxHttpAPI := &fakeEmqxHttpAPI{}
	defStartFun := func(e EmqxHttpAPI, rebalance *appsv1beta4.Rebalance, emqx *appsv1beta4.EmqxEnterprise) error {
		return nil
	}
	defGetFun := func(EmqxHttpAPI) ([]appsv1beta4.RebalanceState, error) {
		return []appsv1beta4.RebalanceState{}, nil
	}
	t.Run("check start rebalance failed", func(t *testing.T) {
		r := rebalance.DeepCopy()

		startFun := func(e EmqxHttpAPI, rebalance *appsv1beta4.Rebalance, emqx *appsv1beta4.EmqxEnterprise) error {
			return errors.New("fake error")
		}
		rebalanceStatusHandler(r, emqxEnterprise, emqxHttpAPI, startFun, defGetFun)
		assert.Equal(t, appsv1beta4.RebalancePhaseFailed, r.Status.Phase)
	})
	t.Run("check start rebalance success", func(t *testing.T) {
		r := rebalance.DeepCopy()
		rebalanceStatusHandler(r, emqxEnterprise, emqxHttpAPI, defStartFun, defGetFun)
		assert.Equal(t, appsv1beta4.RebalancePhaseProcessing, r.Status.Phase)
	})

	t.Run("check get rebalance status failed", func(t *testing.T) {
		r := rebalance.DeepCopy()
		r.Status.Phase = appsv1beta4.RebalancePhaseProcessing

		getFun := func(EmqxHttpAPI) ([]appsv1beta4.RebalanceState, error) {
			return nil, errors.New("fake error")
		}

		rebalanceStatusHandler(r, emqxEnterprise, emqxHttpAPI, defStartFun, getFun)
		assert.Equal(t, appsv1beta4.RebalancePhaseFailed, r.Status.Phase)
	})

	t.Run("check get rebalance status return empty list", func(t *testing.T) {
		r := rebalance.DeepCopy()
		r.Status.Phase = appsv1beta4.RebalancePhaseProcessing

		rebalanceStatusHandler(r, emqxEnterprise, emqxHttpAPI, defStartFun, defGetFun)
		assert.Equal(t, appsv1beta4.RebalancePhaseCompleted, r.Status.Phase)
	})

	t.Run("check get rebalance status success", func(t *testing.T) {
		r := rebalance.DeepCopy()
		r.Status.Phase = appsv1beta4.RebalancePhaseProcessing

		getFun := func(EmqxHttpAPI) ([]appsv1beta4.RebalanceState, error) {
			return []appsv1beta4.RebalanceState{
				{
					State: "processing",
				},
			}, nil
		}

		rebalanceStatusHandler(r, emqxEnterprise, emqxHttpAPI, defStartFun, getFun)
		assert.Equal(t, appsv1beta4.RebalancePhaseProcessing, r.Status.Phase)
		assert.Equal(t, "processing", r.Status.RebalanceStates[0].State)
	})

	t.Run("check failed handler", func(t *testing.T) {
		r := rebalance.DeepCopy()
		r.Status.Phase = appsv1beta4.RebalancePhaseFailed
		r.Status.RebalanceStates = []appsv1beta4.RebalanceState{
			{State: "fake"},
		}
		rebalanceStatusHandler(r, emqxEnterprise, emqxHttpAPI, defStartFun, defGetFun)
		assert.Nil(t, r.Status.RebalanceStates)
	})

	t.Run("check completed handler", func(t *testing.T) {
		r := rebalance.DeepCopy()
		r.Status.Phase = appsv1beta4.RebalancePhaseCompleted
		r.Status.RebalanceStates = []appsv1beta4.RebalanceState{
			{State: "fake"},
		}
		rebalanceStatusHandler(r, emqxEnterprise, emqxHttpAPI, defStartFun, defGetFun)
		assert.Nil(t, r.Status.RebalanceStates)
	})
}
