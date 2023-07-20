package v2alpha2

import (
	// "fmt"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EmqxVer struct {
	name string
	emqx interface{}
}

var emqxNodeName = "emqx-ee@emqx-ee-0.emqx-ee-headless.default.svc.cluster.local"
var emqxV1 = &appsv1beta4.EmqxEnterprise{
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

var emqxV2 = &appsv2alpha2.EMQX{
	Status: appsv2alpha2.EMQXStatus{
		CoreNodes: []appsv2alpha2.EMQXNode{
			{
				Node: emqxNodeName,
			},
		},
	},
}

func TestGetRebalanceStatus(t *testing.T) {
	emqxVers := []EmqxVer{
		{"v1beta4", emqxV1},
		{"v2alpha2", emqxV2},
	}

	for _, tc := range emqxVers {
		tc := tc // Create a new variable to avoid variable capture in closures
		t.Run(tc.name, func(t *testing.T) {
			f := &innerReq.FakeRequester{}

			t.Run("check request args", func(t *testing.T) {
				f.ReqFunc = func(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
					path, err := rebalanceStatusUrl(tc.emqx)
					assert.Nil(t, err)
					assert.Equal(t, path, url.Path)
					assert.Equal(t, "GET", method)
					assert.Nil(t, body)
					resp = &http.Response{StatusCode: http.StatusOK}
					respBody = []byte(`{"rebalances":[]}`)
					err = nil
					return
				}
				_, err := getRebalanceStatus(tc.emqx, f)
				assert.Nil(t, err)
			})

			t.Run("check request return error", func(t *testing.T) {
				f.ReqFunc = func(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
					return nil, nil, errors.New("fake error")
				}
				_, err := getRebalanceStatus(tc.emqx, f)
				assert.Error(t, err, "fake error")
			})

			t.Run("check request return error status code", func(t *testing.T) {
				f.ReqFunc = func(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
					return &http.Response{StatusCode: http.StatusBadRequest}, nil, nil
				}
				_, err := getRebalanceStatus(tc.emqx, f)
				assert.ErrorContains(t, err, "request api failed")
			})

			t.Run("check request return unexpected JSON", func(t *testing.T) {
				f.ReqFunc = func(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
					return &http.Response{StatusCode: http.StatusOK}, nil, nil
				}
				_, err := getRebalanceStatus(tc.emqx, f)
				assert.ErrorContains(t, err, "failed to unmarshal rebalances")
			})

		})
	}
}

func TestStartRebalanceV1(t *testing.T) {

	f := &innerReq.FakeRequester{}
	rebalance := &appsv2alpha2.Rebalance{
		Spec: appsv2alpha2.RebalanceSpec{
			RebalanceStrategy: appsv2alpha2.RebalanceStrategy{
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

	emqxVers := []EmqxVer{
		{"v1beta4", emqxV1},
		{"v2alpha2", emqxV2},
	}

	for _, tc := range emqxVers {
		tc := tc // Create a new variable to avoid variable capture in closures
		t.Run(tc.name, func(t *testing.T) {
			t.Run("check request args", func(t *testing.T) {
				f.ReqFunc = func(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
					nodes, err := getEmqxNodes(tc.emqx)
					assert.Nil(t, err)
					startPath, err := rebalanceStartUrl(tc.emqx, nodes[0])
					assert.Nil(t, err)
					assert.Equal(t, "POST", method)
					assert.Equal(t, startPath, url.Path)
					assert.Equal(t, getRequestBytes(rebalance, nodes), body)
					resp = &http.Response{StatusCode: http.StatusOK}
					respBody = []byte(`{"data":[],"code":0}`)
					err = nil
					return
				}

				err := startRebalance(tc.emqx, f, rebalance)
				assert.Nil(t, err)
			})

			t.Run("check request return bad request", func(t *testing.T) {
				f.ReqFunc = func(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
					return &http.Response{StatusCode: http.StatusBadRequest}, nil, nil
				}

				err := startRebalance(tc.emqx, f, rebalance)
				assert.ErrorContains(t, err, "request api failed")
			})

			t.Run("check request start rebalance err", func(t *testing.T) {
				f.ReqFunc = func(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
					resp = &http.Response{StatusCode: http.StatusOK}
					respBody = []byte(`{"message":"fake error","code":400}`)
					err = nil
					return
				}
				err := startRebalance(tc.emqx, f, rebalance)
				assert.ErrorContains(t, err, "fake error")
			})

			t.Run("check request return error", func(t *testing.T) {
				f.ReqFunc = func(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
					return nil, nil, errors.New("fake error")
				}
				err := startRebalance(tc.emqx, f, rebalance)
				assert.Error(t, err, "fake error")
			})
		})
	}
}

func TestStopRebalance(t *testing.T) {
	f := &innerReq.FakeRequester{}
	rebalance := &appsv2alpha2.Rebalance{
		Status: appsv2alpha2.RebalanceStatus{
			Phase: "Processing",
			RebalanceStates: []appsv2alpha2.RebalanceState{
				{
					CoordinatorNode: emqxNodeName,
				},
			},
		},
	}

	emqxVers := []EmqxVer{
		{"v1beta4", emqxV1},
		{"v2alpha2", emqxV2},
	}

	for _, tc := range emqxVers {
		tc := tc // Create a new variable to avoid variable capture in closures
		t.Run(tc.name, func(t *testing.T) {
			t.Run("check requestAPI args", func(t *testing.T) {
				f.ReqFunc = func(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
					nodes, err := getEmqxNodes(tc.emqx)
					assert.Nil(t, err)
					stopPath, err := rebalanceStopUrl(tc.emqx, nodes[0])
					assert.Nil(t, err)
					assert.Equal(t, "POST", method)
					assert.Equal(t, stopPath, url.Path)
					assert.Nil(t, body)
					resp = &http.Response{StatusCode: http.StatusOK}
					respBody = []byte(`{"data":[],"code":0}`)
					err = nil
					return
				}
				err := stopRebalance(tc.emqx, f, rebalance)
				assert.Nil(t, err)
			})

			t.Run("check request return bad request", func(t *testing.T) {
				f.ReqFunc = func(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
					return &http.Response{StatusCode: http.StatusBadRequest}, nil, nil
				}

				err := stopRebalance(tc.emqx, f, rebalance)
				assert.ErrorContains(t, err, "request api failed")
			})

			t.Run("check request stop rebalance err", func(t *testing.T) {
				f.ReqFunc = func(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
					resp = &http.Response{StatusCode: http.StatusOK}
					respBody = []byte(`{"message": "rebalance is disabled","code":400}`)
					err = nil
					return
				}
				err := stopRebalance(tc.emqx, f, rebalance)
				assert.ErrorContains(t, err, "rebalance is disabled")
			})

			t.Run("check request return error", func(t *testing.T) {
				f.ReqFunc = func(method string, url url.URL, body []byte, header http.Header) (resp *http.Response, respBody []byte, err error) {
					return nil, nil, errors.New("fake error")
				}
				err := stopRebalance(tc.emqx, f, rebalance)
				assert.Error(t, err, "fake error")
			})
		})
	}
}

func TestGetRequestBytes(t *testing.T) {
	rebalance := &appsv2alpha2.Rebalance{
		Spec: appsv2alpha2.RebalanceSpec{
			RebalanceStrategy: appsv2alpha2.RebalanceStrategy{
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

	emqxVers := []EmqxVer{
		{"v1beta4", &appsv1beta4.EmqxEnterprise{}},
		{"v2alpha2", &appsv2alpha2.EMQX{}},
	}

	for _, tc := range emqxVers {
		tc := tc // Create a new variable to avoid variable capture in closures
		t.Run(tc.name, func(t *testing.T) {
			t.Run("check get request bytes with full rebalance strategy", func(t *testing.T) {
				nodes, err := getEmqxNodes(tc.emqx)
				assert.Nil(t, err)
				bytes := getRequestBytes(rebalance, nodes)

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
				nodes, err := getEmqxNodes(tc.emqx)
				assert.Nil(t, err)
				bytes := getRequestBytes(r, nodes)

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
				nodes, err := getEmqxNodes(tc.emqx)
				assert.Nil(t, err)
				bytes := getRequestBytes(r, nodes)

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

		})
	}
}

func TestRebalanceStatusHandler(t *testing.T) {
	finalizer := "apps.emqx.io/finalizer"
	rebalance := &appsv2alpha2.Rebalance{
		ObjectMeta: metav1.ObjectMeta{
			Finalizers: []string{finalizer},
		},
	}

	f := &innerReq.FakeRequester{}

	emqxVers := []EmqxVer{
		{"v1beta4", &appsv1beta4.EmqxEnterprise{}},
		{"v2alpha2", &appsv2alpha2.EMQX{}},
	}

	for _, tc := range emqxVers {
		tc := tc // Create a new variable to avoid variable capture in closures
		t.Run(tc.name, func(t *testing.T) {
			defStartFun := func(emqx interface{}, requester innerReq.RequesterInterface, rebalance *appsv2alpha2.Rebalance) error {
				return nil
			}
			defGetFun := func(emqx interface{}, requester innerReq.RequesterInterface) ([]appsv2alpha2.RebalanceState, error) {
				return []appsv2alpha2.RebalanceState{}, nil
			}
			t.Run("check start rebalance failed", func(t *testing.T) {
				r := rebalance.DeepCopy()

				startFun := func(emqx interface{}, requester innerReq.RequesterInterface, rebalance *appsv2alpha2.Rebalance) error {
					return errors.New("fake error")
				}
				rebalanceStatusHandler(tc.emqx, r, f, startFun, defGetFun)
				assert.Equal(t, appsv2alpha2.RebalancePhaseFailed, r.Status.Phase)
			})
			t.Run("check start rebalance success", func(t *testing.T) {
				r := rebalance.DeepCopy()
				rebalanceStatusHandler(tc.emqx, r, f, defStartFun, defGetFun)
				assert.Equal(t, appsv2alpha2.RebalancePhaseProcessing, r.Status.Phase)
			})

			t.Run("check get rebalance status failed", func(t *testing.T) {
				r := rebalance.DeepCopy()
				r.Status.Phase = appsv2alpha2.RebalancePhaseProcessing

				getFun := func(emqx interface{}, requester innerReq.RequesterInterface) ([]appsv2alpha2.RebalanceState, error) {
					return nil, errors.New("fake error")
				}

				rebalanceStatusHandler(tc.emqx, r, f, defStartFun, getFun)
				assert.Equal(t, appsv2alpha2.RebalancePhaseFailed, r.Status.Phase)
			})

			t.Run("check get rebalance status return empty list", func(t *testing.T) {
				r := rebalance.DeepCopy()
				r.Status.Phase = appsv2alpha2.RebalancePhaseProcessing

				rebalanceStatusHandler(tc.emqx, r, f, defStartFun, defGetFun)
				assert.Equal(t, appsv2alpha2.RebalancePhaseCompleted, r.Status.Phase)
			})

			t.Run("check get rebalance status success", func(t *testing.T) {
				r := rebalance.DeepCopy()
				r.Status.Phase = appsv2alpha2.RebalancePhaseProcessing

				getFun := func(emqx interface{}, requester innerReq.RequesterInterface) ([]appsv2alpha2.RebalanceState, error) {
					return []appsv2alpha2.RebalanceState{
						{
							State: "processing",
						},
					}, nil
				}

				rebalanceStatusHandler(tc.emqx, r, f, defStartFun, getFun)
				assert.Equal(t, appsv2alpha2.RebalancePhaseProcessing, r.Status.Phase)
				assert.Equal(t, "processing", r.Status.RebalanceStates[0].State)
			})

			t.Run("check failed handler", func(t *testing.T) {
				r := rebalance.DeepCopy()
				r.Status.Phase = appsv2alpha2.RebalancePhaseFailed
				r.Status.RebalanceStates = []appsv2alpha2.RebalanceState{
					{State: "fake"},
				}
				rebalanceStatusHandler(tc.emqx, r, f, defStartFun, defGetFun)
				assert.Nil(t, r.Status.RebalanceStates)
			})

			t.Run("check completed handler", func(t *testing.T) {
				r := rebalance.DeepCopy()
				r.Status.Phase = appsv2alpha2.RebalancePhaseCompleted
				r.Status.RebalanceStates = []appsv2alpha2.RebalanceState{
					{State: "fake"},
				}
				rebalanceStatusHandler(tc.emqx, r, f, defStartFun, defGetFun)
				assert.Nil(t, r.Status.RebalanceStates)
			})
		})
	}
}
