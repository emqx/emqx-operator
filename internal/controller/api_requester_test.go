package controller

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"syscall"
	"testing"
	"time"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	"github.com/emqx/emqx-operator/internal/emqx/api"
	req "github.com/emqx/emqx-operator/internal/requester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

func TestRequesterFilter(t *testing.T) {
	var coreSetName = "emqx-core"
	var coreSetUID types.UID = "123"

	instance := &crd.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Status: crd.EMQXStatus{
			CoreNodesStatus: crd.CoreNodesStatus{},
			ReplicantNodesStatus: crd.ReplicantNodesStatus{
				CurrentRevision: "cur",
				UpdateRevision:  "upd",
			},
			CoreNodes: []crd.EMQXNode{
				{
					PodName:     coreSetName + "-0",
					Name:        "emqx@core-0",
					Status:      "running",
					OTPRelease:  "27.2-3/15.2",
					Version:     "5.10.0",
					Role:        "core",
					Sessions:    0,
					Connections: 0,
				},
				{
					PodName:     coreSetName + "-1",
					Name:        "emqx@core-1",
					Status:      "running",
					OTPRelease:  "27.2-3/15.2",
					Version:     "5.10.0",
					Role:        "core",
					Sessions:    0,
					Connections: 0,
				},
			},
			ReplicantNodes: []crd.EMQXNode{},
		},
	}

	coreOwnerReference := metav1.OwnerReference{
		APIVersion: "apps/v1",
		Kind:       "StatefulSet",
		Name:       coreSetName,
		UID:        coreSetUID,
		Controller: ptr.To(true),
	}

	podSpec := corev1.PodSpec{
		Containers: []corev1.Container{{
			Name: crd.DefaultContainerName,
			Ports: []corev1.ContainerPort{{
				Name:          "dashboard",
				ContainerPort: 18083,
			}},
		}}}

	state := &reconcileState{
		coreSets: []*appsv1.StatefulSet{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: coreSetName,
					UID:  coreSetUID,
				},
				Status: appsv1.StatefulSetStatus{
					Replicas:       2,
					UpdateRevision: "upd",
				},
			},
		},
		replicantSets: []*appsv1.ReplicaSet{},
		pods: []*corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              coreSetName + "-0",
					Labels:            crd.CoreLabels(),
					CreationTimestamp: metav1.NewTime(time.Now().Add(-1 * time.Minute)),
					OwnerReferences:   []metav1.OwnerReference{coreOwnerReference},
				},
				Spec: podSpec,
				Status: corev1.PodStatus{
					PodIP:      "",
					Phase:      corev1.PodPending,
					Conditions: []corev1.PodCondition{},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              coreSetName + "-1",
					Labels:            crd.CoreLabels(),
					CreationTimestamp: metav1.NewTime(time.Now().Add(-1 * time.Second)),
					OwnerReferences:   []metav1.OwnerReference{coreOwnerReference},
				},
				Spec: podSpec,
				Status: corev1.PodStatus{
					PodIP:      "10.0.0.2",
					Phase:      corev1.PodRunning,
					Conditions: []corev1.PodCondition{{Type: corev1.ContainersReady, Status: corev1.ConditionTrue}},
				},
			},
		},
	}

	builder := &apiRequesterBuilder{
		username: "emqx",
		password: "emqx",
	}

	var requester req.RequesterInterface

	requester = builder.forCore(state)
	assert.NotNil(t, requester)
	assert.Equal(t, state.pods[1].Name, requester.(*coreClusterRequester).preferredEndpoint().Description)

	requester = builder.forPod(state.pods[0])
	assert.Nil(t, requester)

	requester = builder.forPod(state.pods[1])
	assert.NotNil(t, requester)

	// Filter by the single core StatefulSet:
	requester = builder.forCore(state, &podsManagedBy{state.coreSet()})
	assert.NotNil(t, requester)
	assert.Equal(t, state.pods[1].Name, requester.(*coreClusterRequester).preferredEndpoint().Description)

	requester = builder.forCore(state, &podsWithEMQXVersion{instance, "5.10."})
	assert.NotNil(t, requester)

	requester = builder.forCore(state, &podsWithEMQXVersion{instance, "6."})
	assert.Nil(t, requester)

}

func TestCorePreference(t *testing.T) {
	const coreSetName = "emqx-core"

	coreSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: coreSetName,
			UID:  "abcdef",
		},
		Status: appsv1.StatefulSetStatus{
			UpdateRevision: "rev-new",
		},
	}
	coreSetReference := metav1.OwnerReference{
		APIVersion: "apps/v1",
		Kind:       "StatefulSet",
		Name:       coreSetName,
		UID:        "abcdef",
		Controller: ptr.To(true),
	}

	mkPod := func(name, revision, ip string, port int32, ready bool, deleting bool) *corev1.Pod {
		labels := crd.CoreLabels()
		labels[appsv1.ControllerRevisionHashLabelKey] = revision
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              name,
				Labels:            labels,
				CreationTimestamp: metav1.NewTime(time.Now()),
				OwnerReferences:   []metav1.OwnerReference{coreSetReference},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name: crd.DefaultContainerName,
					Ports: []corev1.ContainerPort{{
						Name:          "dashboard",
						ContainerPort: port,
					}},
				}}},
			Status: corev1.PodStatus{
				PodIP: ip,
				Phase: corev1.PodRunning,
			},
		}
		if ready {
			pod.Status.Conditions = []corev1.PodCondition{
				{Type: corev1.ContainersReady, Status: corev1.ConditionTrue},
			}
		}
		if deleting {
			pod.DeletionTimestamp = ptr.To(metav1.NewTime(time.Now()))
		}
		return pod
	}

	builder := &apiRequesterBuilder{
		username: "emqx",
		password: "emqx",
	}

	t.Run("skips deleting and not ready pods", func(t *testing.T) {
		state := &reconcileState{
			coreSets: []*appsv1.StatefulSet{coreSet},
			pods: []*corev1.Pod{
				mkPod(coreSetName+"-0", "rev-new", "10.0.0.1", 18083, true, true),
				mkPod(coreSetName+"-1", "rev-new", "10.0.0.2", 18083, false, false),
				mkPod(coreSetName+"-2", "rev-new", "10.0.0.3", 18083, true, false),
			},
		}

		requester := builder.forCore(state)

		assert.NotNil(t, requester)
		assert.Equal(t, coreSetName+"-2", requester.(*coreClusterRequester).preferredEndpoint().Description)
	})

	t.Run("returns eligible pods in preference order for non-HTTP callers", func(t *testing.T) {
		state := &reconcileState{
			coreSets: []*appsv1.StatefulSet{coreSet},
			pods: []*corev1.Pod{
				mkPod(coreSetName+"-0", "rev-new", "", 18083, true, true),
				mkPod(coreSetName+"-1", "rev-new", "", 18083, false, false),
				mkPod(coreSetName+"-2", "rev-old", "", 18083, true, false),
				mkPod(coreSetName+"-3", "rev-new", "", 18083, true, false),
			},
		}

		pods := preferredCorePods(state, podsManagedBy{coreSet})

		require.Len(t, pods, 2)
		assert.Equal(t, coreSetName+"-3", pods[0].Name)
		assert.Equal(t, coreSetName+"-2", pods[1].Name)
	})

	t.Run("deprioritizes sole outdated pod over larger fresh ordinal", func(t *testing.T) {
		state := &reconcileState{
			coreSets: []*appsv1.StatefulSet{coreSet},
			pods: []*corev1.Pod{
				mkPod(coreSetName+"-0", "rev-old", "10.0.0.1", 18083, true, false),
				mkPod(coreSetName+"-1", "rev-new", "10.0.0.2", 18083, true, false),
			},
		}

		requester := builder.forCore(state)

		assert.NotNil(t, requester)
		assert.Equal(t, coreSetName+"-1", requester.(*coreClusterRequester).preferredEndpoint().Description)
	})

	t.Run("prefers smaller ordinal when more than one outdated pod remains", func(t *testing.T) {
		state := &reconcileState{
			coreSets: []*appsv1.StatefulSet{coreSet},
			pods: []*corev1.Pod{
				mkPod(coreSetName+"-0", "rev-old", "10.0.0.1", 18083, true, false),
				mkPod(coreSetName+"-1", "rev-old", "10.0.0.2", 18083, true, false),
				mkPod(coreSetName+"-2", "rev-new", "10.0.0.3", 18083, true, false),
			},
		}

		requester := builder.forCore(state)

		assert.NotNil(t, requester)
		assert.Equal(t, coreSetName+"-0", requester.(*coreClusterRequester).preferredEndpoint().Description)
	})

	t.Run("orders all usable endpoints by preference", func(t *testing.T) {
		state := &reconcileState{
			coreSets: []*appsv1.StatefulSet{coreSet},
			pods: []*corev1.Pod{
				mkPod(coreSetName+"-3", "rev-new", "10.0.0.4", 18083, true, false),
				mkPod(coreSetName+"-2", "rev-new", "10.0.0.3", 18083, true, false),
				mkPod(coreSetName+"-1", "rev-new", "", 18083, true, false),
				mkPod(coreSetName+"-0", "rev-old", "10.0.0.1", 18083, true, false),
			},
		}
		r := builder.forCore(state)
		require.NotNil(t, r)
		var names []string
		for _, endpoint := range r.(*coreClusterRequester).endpoints {
			names = append(names, endpoint.(*req.Requester).Description)
		}
		require.Equal(t, []string{coreSetName + "-2", coreSetName + "-3", coreSetName + "-0"}, names)
	})

	t.Run("prefers smaller ordinal among equally fresh pods", func(t *testing.T) {
		state := &reconcileState{
			coreSets: []*appsv1.StatefulSet{coreSet},
			pods: []*corev1.Pod{
				mkPod(coreSetName+"-2", "rev-new", "10.0.0.3", 18083, true, false),
				mkPod(coreSetName+"-1", "rev-new", "10.0.0.2", 18083, true, false),
			},
		}

		requester := builder.forCore(state)

		assert.NotNil(t, requester)
		assert.Equal(t, coreSetName+"-1", requester.(*coreClusterRequester).preferredEndpoint().Description)
	})
}

func TestRequesterUsesPodDashboardPort(t *testing.T) {
	builder := &apiRequesterBuilder{username: "emqx", password: "secret"}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "emqx-core-0"},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{
			Name: crd.DefaultContainerName,
			Ports: []corev1.ContainerPort{{
				Name:          "dashboard",
				ContainerPort: 28083,
			}},
		}}},
		Status: corev1.PodStatus{PodIP: "10.0.0.1"},
	}

	irequester := builder.forPod(pod)
	require.NotNil(t, irequester)
	requester := irequester.(*req.Requester)
	assert.Equal(t, "10.0.0.1:28083", requester.Host)
	assert.Equal(t, "http", requester.GetURL("status").Scheme)

	pod.Spec.Containers[0].Ports = []corev1.ContainerPort{
		{Name: "dashboard-https", ContainerPort: 28084},
	}
	irequester = builder.forPod(pod)
	require.NotNil(t, irequester)
	requester = irequester.(*req.Requester)
	assert.Equal(t, "10.0.0.1:28084", requester.Host)
	assert.Equal(t, "https", requester.GetURL("status").Scheme)

	pod.Spec.Containers[0].Ports = []corev1.ContainerPort{
		{Name: "dashboard-https", ContainerPort: 28084},
		{Name: "dashboard", ContainerPort: 28083},
	}
	irequester = builder.forPod(pod)
	require.NotNil(t, irequester)
	requester = irequester.(*req.Requester)
	assert.Equal(t, "10.0.0.1:28083", requester.Host)
	assert.Equal(t, "http", requester.GetURL("status").Scheme)

	pod.Spec.Containers[0].Ports = nil
	assert.Nil(t, builder.forPod(pod))
}

func TestCoreClusterRequesterFailover(t *testing.T) {
	for _, tc := range []struct {
		name           string
		primaryStatus  int
		fallbackStatus int
	}{
		{"503 then success", 503, 200},
		{"EOF then success", 0, 200},
		{"500 then unauthorized", 500, 401},
		{"500 then unavailable", 500, 503},
		{"500 then EOF", 500, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []string
			makeEndpoint := func(name string, status int) *req.Requester {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls = append(calls, name)
					if status == 0 {
						conn, _, err := w.(http.Hijacker).Hijack()
						if assert.NoError(t, err) {
							_ = conn.Close()
						}
						return
					}
					// Avoid HTTP transport retries on reused connections in the EOF cases.
					w.Header().Set("Connection", "close")
					w.WriteHeader(status)
					_, _ = w.Write([]byte("config"))
				}))
				t.Cleanup(server.Close)
				u, err := url.Parse(server.URL)
				require.NoError(t, err)
				return &req.Requester{Host: u.Host, Description: name}
			}
			primary := makeEndpoint("primary", tc.primaryStatus)
			fallback := makeEndpoint("fallback", tc.fallbackStatus)
			r := &coreClusterRequester{endpoints: []req.RequesterInterface{primary, fallback}}
			if tc.fallbackStatus == 200 || tc.fallbackStatus == 401 {
				// Success and non-retryable errors must stop before a third endpoint.
				r.endpoints = append(r.endpoints, makeEndpoint("unused", 200))
			}
			// Request twice to verify requester starts from primary endpoint each time.
			for range 2 {
				config, err := api.Configs(r)
				if tc.fallbackStatus == 200 {
					require.NoError(t, err)
					require.Equal(t, "config", config)
				} else {
					require.ErrorContains(t, err, "http://"+fallback.Host+"/api/v5/configs")
					require.NotContains(t, err.Error(), primary.Host)
					switch tc.fallbackStatus {
					case 0:
						require.ErrorIs(t, err, io.EOF)
						require.ErrorContains(t, err, "error accessing fallback API")
					case 503:
						require.ErrorIs(t, err, api.ErrorServiceUnavailable)
					default:
						require.ErrorContains(t, err, "HTTP 401")
					}
				}
			}
			// Each GET restarts at the first endpoint.
			require.Equal(t, []string{"primary", "fallback", "primary", "fallback"}, calls)
		})
	}
}

func TestCoreClusterRequesterRetryPolicy(t *testing.T) {
	connectionRefused := &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
	for _, tc := range []struct {
		name      string
		method    string
		status    int
		err       error
		wantCalls int
	}{
		{"success", "GET", 200, nil, 1},
		{"redirect", "GET", 302, nil, 1},
		{"unauthorized", "GET", 401, nil, 1},
		{"not found", "GET", 404, nil, 1},
		{"rate limited", "GET", 429, nil, 1},
		{"internal error", "GET", 500, nil, 3},
		{"bad gateway", "GET", 502, nil, 3},
		{"unavailable", "GET", 503, nil, 3},
		{"gateway timeout", "GET", 504, nil, 3},
		{"EOF", "GET", 0, fmt.Errorf("wrapped: %w", io.EOF), 3},
		{"truncated body", "GET", 200, io.ErrUnexpectedEOF, 3},
		{"refused", "GET", 0, connectionRefused, 3},
		{"reset", "GET", 0, syscall.ECONNRESET, 3},
		{"closed", "GET", 0, net.ErrClosed, 3},
		{"timeout", "GET", 0, &net.DNSError{IsTimeout: true}, 3},
		{"invalid request", "GET", 0, errors.New("invalid request"), 1},
		{"internal error", "POST", 500, nil, 1},
		{"EOF", "POST", 0, io.EOF, 1},
		{"unavailable", "PUT", 503, nil, 1},
		{"truncated body", "PUT", 200, io.ErrUnexpectedEOF, 1},
		{"gateway timeout", "DELETE", 504, nil, 1},
		{"refused", "DELETE", 0, connectionRefused, 1},
		{"bad gateway", "PATCH", 502, nil, 1},
		{"reset", "PATCH", 0, syscall.ECONNRESET, 1},
		{"unavailable", "HEAD", 503, nil, 1},
		{"timeout", "HEAD", 0, &net.DNSError{IsTimeout: true}, 1},
	} {
		t.Run(tc.method+"/"+tc.name, func(t *testing.T) {
			calls := 0
			endpoint := req.NewMockRequester(func(
				m string, u url.URL, b []byte, h http.Header,
			) (*http.Response, []byte, error) {
				calls++
				require.Equal(t, tc.method, m)
				return &http.Response{StatusCode: tc.status}, []byte("response"), tc.err
			})
			r := &coreClusterRequester{endpoints: []req.RequesterInterface{endpoint, endpoint, endpoint}}
			resp, body, err := r.Request(tc.method, url.URL{Path: "/api/v5/nodes"}, nil, nil)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.status, resp.StatusCode)
			require.Equal(t, "response", string(body))
			require.Equal(t, tc.wantCalls, calls)
		})
	}
}
