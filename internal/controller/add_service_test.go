package controller

import (
	"testing"

	crdv2 "github.com/emqx/emqx-operator/api/v2"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func loadConf(data string) *config.EMQX {
	conf, _ := config.EMQXConfigWithDefaults(data)
	return conf
}
func TestGenerateDashboardService(t *testing.T) {

	t.Run("check metadata", func(t *testing.T) {
		emqx := &crdv2.EMQX{
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
			Spec: crdv2.EMQXSpec{
				DashboardServiceTemplate: &crdv2.ServiceTemplate{
					Enabled: ptr.To(true),
					ObjectMeta: metav1.ObjectMeta{
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
		got := generateDashboardService(emqx, loadConf(""))
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

	t.Run("check disabled", func(t *testing.T) {
		emqx := &crdv2.EMQX{}
		emqx.Spec.DashboardServiceTemplate = &crdv2.ServiceTemplate{
			Enabled: ptr.To(false),
		}
		got := generateDashboardService(emqx, loadConf(""))
		assert.Nil(t, got)
	})

	t.Run("check selector", func(t *testing.T) {
		emqx := &crdv2.EMQX{
			ObjectMeta: metav1.ObjectMeta{
				Name: "emqx",
			},
		}
		got := generateDashboardService(emqx, loadConf(""))
		assert.Equal(t, map[string]string{
			crdv2.LabelInstance:  "emqx",
			crdv2.LabelManagedBy: "emqx-operator",
			crdv2.LabelDBRole:    "core",
		}, got.Spec.Selector)
	})

	t.Run("check http ports", func(t *testing.T) {
		emqx := &crdv2.EMQX{}
		got := generateDashboardService(emqx, loadConf(`
		dashboard.listeners.http.bind = 18083
		`))
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
		emqx := &crdv2.EMQX{}
		got := generateDashboardService(emqx, loadConf(`
		dashboard.listeners.http.bind = 0
		dashboard.listeners.https.bind = 18084
		`))
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
		emqx := &crdv2.EMQX{}
		got := generateDashboardService(emqx, loadConf(`
		dashboard.listeners.http.bind = 18083
		dashboard.listeners.https.bind = 18084
		`))
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
		emqx := &crdv2.EMQX{}
		got := generateDashboardService(emqx, loadConf(`
		dashboard.listeners.http.bind = 0
		dashboard.listeners.https.bind = 0
		`))
		assert.Nil(t, got)
	})
}

func TestGenerateListenersService(t *testing.T) {
	t.Run("check metadata", func(t *testing.T) {
		emqx := &crdv2.EMQX{
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
			Spec: crdv2.EMQXSpec{
				ListenersServiceTemplate: &crdv2.ServiceTemplate{
					Enabled: ptr.To(true),
					ObjectMeta: metav1.ObjectMeta{
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
		got := generateListenerService(newReconcileRound(), emqx, loadConf(""))
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

	t.Run("check disabled", func(t *testing.T) {
		emqx := &crdv2.EMQX{}
		emqx.Spec.ListenersServiceTemplate = &crdv2.ServiceTemplate{
			Enabled: ptr.To(false),
		}
		got := generateListenerService(newReconcileRound(), emqx, loadConf(""))
		assert.Nil(t, got)
	})

	t.Run("check core pod selector by default", func(t *testing.T) {
		emqx := &crdv2.EMQX{
			ObjectMeta: metav1.ObjectMeta{
				Name: "emqx",
			},
		}
		got := generateListenerService(newReconcileRound(), emqx, loadConf(""))
		assert.Equal(t, map[string]string{
			crdv2.LabelInstance:  "emqx",
			crdv2.LabelManagedBy: "emqx-operator",
			crdv2.LabelDBRole:    "core",
		}, got.Spec.Selector)
	})

	t.Run("check default ports", func(t *testing.T) {
		emqx := &crdv2.EMQX{}
		got := generateListenerService(newReconcileRound(), emqx, loadConf(""))
		assert.ElementsMatch(t, []corev1.ServicePort{
			{
				Name:       "tcp-default",
				Port:       1883,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(1883),
			},
			{
				Name:       "ssl-default",
				Port:       8883,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(8883),
			},
			{
				Name:       "ws-default",
				Port:       8083,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(8083),
			},
			{
				Name:       "wss-default",
				Port:       8084,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(8084),
			},
		}, got.Spec.Ports)
	})

	t.Run("check ports", func(t *testing.T) {
		emqx := &crdv2.EMQX{}
		conf, _ := config.EMQXConfigWithDefaults(`
		gateway.lwm2m.listeners.udp.default.bind = 5783
		`)
		got := generateListenerService(newReconcileRound(), emqx, conf)
		assert.ElementsMatch(t, []corev1.ServicePort{
			{
				Name:       "lwm2m-udp-default",
				Port:       5783,
				Protocol:   corev1.ProtocolUDP,
				TargetPort: intstr.FromInt(5783),
			},
		}, got.Spec.Ports)
	})
}
