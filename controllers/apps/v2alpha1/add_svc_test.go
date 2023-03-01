package v2alpha1

import (
	"testing"

	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestGenerateHeadlessSVC(t *testing.T) {
	instance := &appsv2alpha1.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: appsv2alpha1.EMQXSpec{
			CoreTemplate: appsv2alpha1.EMQXCoreTemplate{
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
					Name:       "ekka",
					Port:       4370,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(4370),
				},
			},
			Selector: coreLabels,
		},
	}
	assert.Equal(t, expect, generateHeadlessService(instance))
}

func TestGenerateDashboardService(t *testing.T) {
	instance := &appsv2alpha1.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: appsv2alpha1.EMQXSpec{
			CoreTemplate: appsv2alpha1.EMQXCoreTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Labels: coreLabels,
				},
			},
			DashboardServiceTemplate: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "emqx-dashboard",
					Labels: map[string]string{
						"apps.emqx.io/instance": "emqx",
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
				"apps.emqx.io/instance": "emqx",
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

func TestGenerateListenerService(t *testing.T) {
	listenerPorts := []corev1.ServicePort{
		{
			Name:       "mqtt",
			Protocol:   corev1.ProtocolTCP,
			Port:       1883,
			TargetPort: intstr.FromInt(1883),
		},
	}

	instance := &appsv2alpha1.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Spec: appsv2alpha1.EMQXSpec{
			CoreTemplate: appsv2alpha1.EMQXCoreTemplate{
				Spec: appsv2alpha1.EMQXCoreTemplateSpec{
					Replicas: &[]int32{1}[0],
				},
			},
			ReplicantTemplate: appsv2alpha1.EMQXReplicantTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Labels: replicantLabels,
				},
				Spec: appsv2alpha1.EMQXReplicantTemplateSpec{
					Replicas: &[]int32{1}[0],
				},
			},
			ListenersServiceTemplate: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "emqx-listeners",
					Labels: map[string]string{
						"apps.emqx.io/instance": "emqx",
					},
					Annotations: map[string]string{
						"foo": "bar",
					},
				},
				Spec: corev1.ServiceSpec{},
			},
		},
	}
	assert.Nil(t, generateListenerService(instance, []corev1.ServicePort{}))

	expect := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx-listeners",
			Namespace: "emqx",
			Labels: map[string]string{
				"apps.emqx.io/instance": "emqx",
			},
			Annotations: map[string]string{
				"foo": "bar",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: replicantLabels,
			Ports:    listenerPorts,
		},
	}
	assert.Equal(t, expect, generateListenerService(instance, listenerPorts))
}
