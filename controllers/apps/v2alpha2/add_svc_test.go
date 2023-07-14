package v2alpha2

import (
	"testing"

	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestGenerateHeadlessSVC(t *testing.T) {
	instance := &appsv2alpha2.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: appsv2alpha2.EMQXSpec{
			CoreTemplate: appsv2alpha2.EMQXCoreTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Labels: coreLabels,
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
			Name:      "emqx-headless",
			Namespace: "emqx",
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
			Selector: coreLabels,
		},
	}
	assert.Equal(t, expect, generateHeadlessService(instance))
}

func TestGenerateDashboardService(t *testing.T) {
	instance := &appsv2alpha2.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: appsv2alpha2.EMQXSpec{
			CoreTemplate: appsv2alpha2.EMQXCoreTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Labels: coreLabels,
				},
			},
			DashboardServiceTemplate: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "emqx-dashboard",
					Labels: map[string]string{
						appsv2alpha2.InstanceNameLabelKey: "emqx",
					},
					Annotations: map[string]string{
						"foo": "bar",
					},
				},
				Spec: corev1.ServiceSpec{
					Selector: coreLabels,
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
			Labels: map[string]string{
				appsv2alpha2.InstanceNameLabelKey: "emqx",
			},
			Annotations: map[string]string{
				"foo": "bar",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: coreLabels,
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

	assert.Equal(t, expect, generateDashboardService(instance))
}
