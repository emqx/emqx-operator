package controller

import (
	"testing"

	appsv2beta1 "github.com/emqx/emqx-operator/api/v2beta1"
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
