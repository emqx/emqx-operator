package controller

import (
	"errors"
	"testing"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	req "github.com/emqx/emqx-operator/internal/requester"
	"github.com/emqx/emqx-operator/test/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func TestGenerateDashboardService(t *testing.T) {

	t.Run("check metadata", func(t *testing.T) {
		emqx := &crd.EMQX{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "emqx",
				Namespace: "emqx",
				Labels: map[string]string{
					"emqx-label-key": "emqx",
				},
				Annotations: map[string]string{
					"emqx-annotation-key": "emqx",
				},
			},
			Spec: crd.EMQXSpec{
				DashboardServiceTemplate: &crd.ServiceTemplate{
					Enabled: ptr.To(true),
					TemplateObjectMeta: crd.TemplateObjectMeta{
						Labels: map[string]string{
							"dashboard-label-key": "dashboard",
						},
						Annotations: map[string]string{
							"dashboard-annotation-key": "dashboard",
						},
					},
				},
			},
		}
		got := generateDashboardService(emqx)
		assert.Equal(t, metav1.ObjectMeta{
			Name:      "emqx-dashboard",
			Namespace: "emqx",
			Labels: map[string]string{
				"apps.emqx.io/instance":   "emqx",
				"apps.emqx.io/managed-by": "emqx-operator",
				"dashboard-label-key":     "dashboard",
				// "emqx-label-key":          "emqx",
			},
			Annotations: map[string]string{
				"dashboard-annotation-key": "dashboard",
				// "emqx-annotation-key":      "emqx",
			},
		}, got.ObjectMeta)
	})

	t.Run("check selector", func(t *testing.T) {
		emqx := &crd.EMQX{
			ObjectMeta: metav1.ObjectMeta{
				Name: "emqx",
			},
		}
		got := generateDashboardService(emqx)
		assert.Equal(t, map[string]string{
			crd.LabelInstance:  "emqx",
			crd.LabelManagedBy: "emqx-operator",
			crd.LabelMriaRole:  "core",
		}, got.Spec.Selector)
	})

	t.Run("check http ports", func(t *testing.T) {
		emqx := &crd.EMQX{}
		emqx.Spec.Config.Roots = crd.ConfigRoots{
			"dashboard": apiextv1.JSON{Raw: util.FromYAMLString(`
                listeners:
                    http:
                        bind: 18083
            `)},
		}
		got := generateDashboardService(emqx)
		assert.Equal(t, []corev1.ServicePort{
			{
				Name:       "dashboard",
				Protocol:   corev1.ProtocolTCP,
				Port:       18083,
				TargetPort: intstr.FromInt(18083),
			},
		}, got.Spec.Ports)
	})

	t.Run("check https ports", func(t *testing.T) {
		emqx := &crd.EMQX{}
		emqx.Spec.Config.Roots = crd.ConfigRoots{
			"dashboard": apiextv1.JSON{Raw: util.FromYAMLString(`
                listeners:
                    http:
                        bind: 0
                    https:
                        bind: 18084
            `)},
		}
		got := generateDashboardService(emqx)
		assert.Equal(t, []corev1.ServicePort{
			{
				Name:       "dashboard-https",
				Protocol:   corev1.ProtocolTCP,
				Port:       18084,
				TargetPort: intstr.FromInt(18084),
			},
		}, got.Spec.Ports)
	})

	t.Run("check http and https ports", func(t *testing.T) {
		emqx := &crd.EMQX{}
		emqx.Spec.Config.Roots = crd.ConfigRoots{
			"dashboard": apiextv1.JSON{Raw: util.FromYAMLString(`
                listeners:
                    http:
                        bind: 18083
                    https:
                        bind: 18084
            `)},
		}
		got := generateDashboardService(emqx)
		assert.ElementsMatch(t, []corev1.ServicePort{
			{
				Name:       "dashboard",
				Protocol:   corev1.ProtocolTCP,
				Port:       18083,
				TargetPort: intstr.FromInt(18083),
			},
			{
				Name:       "dashboard-https",
				Protocol:   corev1.ProtocolTCP,
				Port:       18084,
				TargetPort: intstr.FromInt(18084),
			},
		}, got.Spec.Ports)
	})

	t.Run("check empty ports", func(t *testing.T) {
		emqx := &crd.EMQX{}
		emqx.Spec.Config.Roots = crd.ConfigRoots{
			"dashboard": apiextv1.JSON{Raw: util.FromYAMLString(`
                listeners:
                    http:
                        bind: 0
                    https:
                        bind: 0
            `)},
		}
		got := generateDashboardService(emqx)
		assert.Nil(t, got)
	})
}

func TestGenerateListenersService(t *testing.T) {
	t.Run("check metadata", func(t *testing.T) {
		emqx := &crd.EMQX{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "emqx",
				Namespace: "emqx",
				Labels: map[string]string{
					"emqx-label-key": "emqx",
				},
				Annotations: map[string]string{
					"emqx-annotation-key": "emqx",
				},
			},
			Spec: crd.EMQXSpec{
				ListenersServiceTemplate: &crd.ServiceTemplate{
					Enabled: ptr.To(true),
					TemplateObjectMeta: crd.TemplateObjectMeta{
						Labels: map[string]string{
							"dashboard-label-key": "listeners",
						},
						Annotations: map[string]string{
							"dashboard-annotation-key": "listeners",
						},
					},
				},
			},
		}
		got := generateListenerService(newReconcileRound(), emqx, nil)
		assert.Equal(t, metav1.ObjectMeta{
			Name:      "emqx-listeners",
			Namespace: "emqx",
			Labels: map[string]string{
				"apps.emqx.io/instance":   "emqx",
				"apps.emqx.io/managed-by": "emqx-operator",
				"dashboard-label-key":     "listeners",
			},
			Annotations: map[string]string{
				"dashboard-annotation-key": "listeners",
			},
		}, got.ObjectMeta)
	})

	t.Run("check core pod selector by default", func(t *testing.T) {
		emqx := &crd.EMQX{
			ObjectMeta: metav1.ObjectMeta{
				Name: "emqx",
			},
		}
		got := generateListenerService(newReconcileRound(), emqx, nil)
		assert.Equal(t, map[string]string{
			crd.LabelInstance:  "emqx",
			crd.LabelManagedBy: "emqx-operator",
			crd.LabelMriaRole:  "core",
		}, got.Spec.Selector)
	})
}

func TestDiscoverListenerServicePorts(t *testing.T) {
	ports, err := discoverListenerServicePorts(
		req.MockRequests(
			"GET api/v5/listeners", `[
				{"type":"tcp","name":"default","enable":true,"bind":"1883","status":{"running":false}},
				{"type":"ssl","name":"external","enable":true,"bind":"mqtt.example:8883"},
				{"type":"ws","name":"v4","enable":true,"bind":"0.0.0.0:8083"},
				{"type":"wss","name":"v6","enable":true,"bind":"[::]:8084"},
				{"type":"quic","name":"default","enable":true,"bind":"14567"},
				{"type":"tcp","name":"disabled","enable":false,"bind":"1884"}
			]`,
			"GET api/v5/gateways", `[
				{"name":"stomp","status":"stopped"},
				{"name":"lwm2m","status":"running"},
				{"name":"nats","status":"running"}
			]`,
			"GET api/v5/gateways/lwm2m/listeners", `[
				{"type":"udp","name":"default","enable":true,"bind":"5783"},
				{"type":"dtls","name":"secure","enable":true,"bind":"[::1]:5784"},
				{"type":"udp","name":"disabled","enable":false,"bind":"5785"}
			]`,
			"GET api/v5/gateways/nats/listeners", `[
				{"type":"tcp","name":"default","enable":true,"bind":"4222"},
				{"type":"ssl","name":"secure","enable":true,"bind":"0.0.0.0:4223"},
				{"type":"ws","name":"websocket","enable":true,"bind":"4224"},
				{"type":"wss","name":"secure-websocket","enable":true,"bind":"4225"}
			]`,
		),
	)
	require.NoError(t, err)
	assert.Equal(t, []corev1.ServicePort{
		{Name: "lwm2m-dtls-secure",
			Protocol:   corev1.ProtocolUDP,
			Port:       5784,
			TargetPort: intstr.FromInt(5784)},
		{Name: "lwm2m-udp-default",
			Protocol:   corev1.ProtocolUDP,
			Port:       5783,
			TargetPort: intstr.FromInt(5783)},
		{Name: "nats-ssl-secure",
			Protocol:   corev1.ProtocolTCP,
			Port:       4223,
			TargetPort: intstr.FromInt(4223)},
		{Name: "nats-tcp-default",
			Protocol:   corev1.ProtocolTCP,
			Port:       4222,
			TargetPort: intstr.FromInt(4222)},
		{Name: "nats-ws-websocket",
			Protocol:    corev1.ProtocolTCP,
			AppProtocol: ptr.To("kubernetes.io/ws"),
			Port:        4224,
			TargetPort:  intstr.FromInt(4224)},
		{Name: "nats-wss-secure-websocket",
			Protocol:    corev1.ProtocolTCP,
			AppProtocol: ptr.To("kubernetes.io/wss"),
			Port:        4225,
			TargetPort:  intstr.FromInt(4225)},
		{Name: "quic-default",
			Protocol:   corev1.ProtocolUDP,
			Port:       14567,
			TargetPort: intstr.FromInt(14567)},
		{Name: "ssl-external",
			Protocol:   corev1.ProtocolTCP,
			Port:       8883,
			TargetPort: intstr.FromInt(8883)},
		{Name: "tcp-default",
			Protocol:   corev1.ProtocolTCP,
			Port:       1883,
			TargetPort: intstr.FromInt(1883)},
		{Name: "ws-v4",
			Protocol:    corev1.ProtocolTCP,
			AppProtocol: ptr.To("kubernetes.io/ws"),
			Port:        8083,
			TargetPort:  intstr.FromInt(8083)},
		{Name: "wss-v6",
			Protocol:    corev1.ProtocolTCP,
			AppProtocol: ptr.To("kubernetes.io/wss"),
			Port:        8084,
			TargetPort:  intstr.FromInt(8084)},
	}, ports)
}

func TestDiscoverListenerServicePortsEmpty(t *testing.T) {
	ports, err := discoverListenerServicePorts(
		req.MockRequests(
			"GET api/v5/listeners", `[]`,
			"GET api/v5/gateways", `[]`,
		),
	)
	require.NoError(t, err)
	assert.Empty(t, ports)
}

func TestDiscoverListenerServicePortsErrors(t *testing.T) {
	tests := []struct {
		name      string
		requester req.RequesterInterface
		contains  []string
	}{
		{
			name: "MQTT request",
			requester: req.MockRequests(
				"GET api/v5/listeners", errors.New("unavailable"),
			),
			contains: []string{"failed to get MQTT listeners", "unavailable"},
		},
		{
			name: "gateway overview request",
			requester: req.MockRequests(
				"GET api/v5/listeners", `[]`,
				"GET api/v5/gateways", errors.New("unavailable"),
			),
			contains: []string{"failed to get gateway overview", "unavailable"},
		},
		{
			name: "one gateway request",
			requester: req.MockRequests(
				"GET api/v5/listeners", `[]`,
				"GET api/v5/gateways", `[{"name":"lwm2m","status":"running"}]`,
				"GET api/v5/gateways/lwm2m/listeners", errors.New("unavailable"),
			),
			contains: []string{"gateway \"lwm2m\"", "unavailable"},
		},
		{
			name: "missing bind port",
			requester: req.MockRequests(
				"GET api/v5/listeners", `[{"type":"tcp","name":"bad","enable":true,"bind":""}]`,
				"GET api/v5/gateways", `[]`,
			),
			contains: []string{"listener tcp:bad", "missing port"},
		},
		{
			name: "non-numeric MQTT bind port",
			requester: req.MockRequests(
				"GET api/v5/listeners", `[{"type":"tcp","name":"bad","enable":true,"bind":"127.0.0.1"}]`,
				"GET api/v5/gateways", `[]`,
			),
			contains: []string{"listener tcp:bad", "non-numeric port"},
		},
		{
			name: "non-numeric gateway bind port",
			requester: req.MockRequests(
				"GET api/v5/listeners", `[]`,
				"GET api/v5/gateways", `[{"name":"coap","status":"running"}]`,
				"GET api/v5/gateways/coap/listeners", `[{"type":"udp","name":"bad","enable":true,"bind":"0.0.0.0:nope"}]`,
			),
			contains: []string{"gateway coap listener udp:bad", "non-numeric port"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := discoverListenerServicePorts(tt.requester)
			require.Error(t, err)
			for _, expected := range tt.contains {
				assert.ErrorContains(t, err, expected)
			}
		})
	}
}
