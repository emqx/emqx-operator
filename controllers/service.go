package controllers

import (
	"github.com/emqx/emqx-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

func makeServiceSpec(instance *v1alpha1.Emqx) v1.ServiceSpec {

	servicePort := []v1.ServicePort{
		{
			Name:     SERVICE_TCP_NAME,
			Port:     SERVICE_TCP_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: SERVICE_TCP_PORT,
			},
		},
		{
			Name:     SERVICE_TCPS_NAME,
			Port:     SERVICE_TCPS_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: SERVICE_TCPS_PORT,
			},
		},
		{
			Name:     SERVICE_WS_NAME,
			Port:     SERVICE_WS_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: SERVICE_WS_PORT,
			},
		},
		{
			Name:     SERVICE_WSS_NAME,
			Port:     SERVICE_WSS_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: SERVICE_WSS_PORT,
			},
		},
		{
			Name:     SERVICE_DASHBOARD_NAME,
			Port:     SERVICE_DASHBOARD_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: SERVICE_DASHBOARD_PORT,
			},
		},
	}

	headlessServiceSpec := v1.ServiceSpec{
		Type:      v1.ServiceTypeClusterIP,
		Ports:     servicePort,
		ClusterIP: "None",
		Selector: map[string]string{
			"app":     EMQX_NAME,
			EMQX_NAME: instance.Name,
		},
	}

	return headlessServiceSpec
}
