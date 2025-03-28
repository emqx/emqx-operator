package v2beta1

import (
	"testing"

	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	config "github.com/emqx/emqx-operator/controllers/apps/v2beta1/config"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func TestGenerateHeadlessSVC(t *testing.T) {
	instance := &appsv2beta1.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: appsv2beta1.EMQXSpec{
			CoreTemplate: appsv2beta1.EMQXCoreTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Labels: appsv2beta1.DefaultCoreLabels(emqx),
				},
			},
		},
	}
	expect := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "emqx",
			Name:      "emqx-headless",
			Labels:    appsv2beta1.DefaultLabels(emqx),
		},
		Spec: corev1.ServiceSpec{
			Type:                     corev1.ServiceTypeClusterIP,
			ClusterIP:                corev1.ClusterIPNone,
			SessionAffinity:          corev1.ServiceAffinityNone,
			PublishNotReadyAddresses: true,
			Ports: []corev1.ServicePort{
				{
					// default Erlang distribution port
					Name:       "erlang-dist",
					Port:       4370,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(4370),
				},
				{
					// emqx back plane gen_rpc port
					Name:       "gen-rpc",
					Port:       5369,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(5369),
				},
			},
			Selector: appsv2beta1.DefaultCoreLabels(emqx),
		},
	}
	assert.Equal(t, expect, generateHeadlessService(instance))
}

func TestGenerateDashboardService(t *testing.T) {

	loadConf := func(data string) *config.Conf {
		conf, _ := config.EMQXConf(config.MergeDefaults(data))
		return conf
	}

	t.Run("check metadata", func(t *testing.T) {
		emqx := &appsv2beta1.EMQX{
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
			Spec: appsv2beta1.EMQXSpec{
				DashboardServiceTemplate: &appsv2beta1.ServiceTemplate{
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
		emqx := &appsv2beta1.EMQX{}
		emqx.Spec.DashboardServiceTemplate = &appsv2beta1.ServiceTemplate{
			Enabled: ptr.To(false),
		}
		got := generateDashboardService(emqx, loadConf(""))
		assert.Nil(t, got)
	})

	t.Run("check selector", func(t *testing.T) {
		emqx := &appsv2beta1.EMQX{
			ObjectMeta: metav1.ObjectMeta{
				Name: "emqx",
			},
		}
		got := generateDashboardService(emqx, loadConf(""))
		assert.Equal(t, map[string]string{
			appsv2beta1.LabelsInstanceKey:  "emqx",
			appsv2beta1.LabelsManagedByKey: "emqx-operator",
			appsv2beta1.LabelsDBRoleKey:    "core",
		}, got.Spec.Selector)
	})

	t.Run("check http ports", func(t *testing.T) {
		emqx := &appsv2beta1.EMQX{}
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
		emqx := &appsv2beta1.EMQX{}
		got := generateDashboardService(emqx, loadConf(`
		dashboard.listeners.http.bind  = 0
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
		emqx := &appsv2beta1.EMQX{}
		got := generateDashboardService(emqx, loadConf(`
		dashboard.listeners.http.bind  = 18083
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
		emqx := &appsv2beta1.EMQX{}
		got := generateDashboardService(emqx, loadConf(`
		dashboard.listeners.http.bind  = 0
		dashboard.listeners.https.bind = 0
		`))
		assert.Nil(t, got)
	})
}
