package v2beta1

import (
	"testing"

	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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
			DashboardServiceTemplate: &appsv2beta1.ServiceTemplate{
				Enabled: true,
				ObjectMeta: metav1.ObjectMeta{
					Name: "emqx-dashboard",
					Labels: map[string]string{
						appsv2beta1.LabelsInstanceKey: "emqx",
					},
					Annotations: map[string]string{
						"foo": "bar",
					},
				},
				Spec: corev1.ServiceSpec{
					Selector: appsv2beta1.DefaultCoreLabels(emqx),
					Ports: []corev1.ServicePort{
						{
							Name:       "dashboard",
							Protocol:   corev1.ProtocolTCP,
							Port:       18083,
							TargetPort: intstr.FromInt(18083),
						},
					},
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
			Name:      "emqx-dashboard",
			Namespace: "emqx",
			Labels:    appsv2beta1.DefaultLabels(emqx),
			Annotations: map[string]string{
				"foo": "bar",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: appsv2beta1.DefaultCoreLabels(emqx),
			Ports: []corev1.ServicePort{
				{
					Name:       "dashboard",
					Protocol:   corev1.ProtocolTCP,
					Port:       18083,
					TargetPort: intstr.FromInt(18083),
				},
			},
		},
	}

	assert.Equal(t, expect, generateDashboardService(instance, ""))
}
