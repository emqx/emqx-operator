package util

import (
	"github.com/emqx/emqx-operator/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func GenerateDefaultServicePorts() []*corev1.ServicePort {
	defaultPorts := []*corev1.ServicePort{
		{
			Name:     constants.EMQX_LISTENERS__TCP__EXTERNAL_NAME,
			Port:     constants.EMQX_LISTENERS__TCP__EXTERNAL_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: constants.EMQX_LISTENERS__TCP__EXTERNAL_PORT,
			},
		},
		{
			Name:     constants.EMQX_LISTENERS__SSL__EXTERNAL_NAME,
			Port:     constants.EMQX_LISTENERS__SSL__EXTERNAL_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: constants.EMQX_LISTENERS__SSL__EXTERNAL_PORT,
			},
		},
		{
			Name:     constants.EMQX_LISTENERS__WS__EXTERNAL_NAME,
			Port:     constants.EMQX_LISTENERS__WS__EXTERNAL_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: constants.EMQX_LISTENERS__WS__EXTERNAL_PORT,
			},
		},
		{
			Name:     constants.EMQX_LISTENERS__WSS__EXTERNAL_NAME,
			Port:     constants.EMQX_LISTENERS__WSS__EXTERNAL_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: constants.EMQX_LISTENERS__WSS__EXTERNAL_PORT,
			},
		},
		{
			Name:     constants.EMQX_DASHBOARD__LISTENER__HTTP_NAME,
			Port:     constants.EMQX_DASHBOARD__LISTENER__HTTP_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: constants.EMQX_DASHBOARD__LISTENER__HTTP_PORT,
			},
		},
		{
			Name:     constants.EMQX_MANAGEMENT__LISTENER__HTTP_NAME,
			Port:     constants.EMQX_MANAGEMENT__LISTENER__HTTP_PORT,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: constants.EMQX_MANAGEMENT__LISTENER__HTTP_PORT,
			},
		},
	}
	return defaultPorts
}

func MergeServicePorts(ports []corev1.ServicePort) []corev1.ServicePort {
	mergePorts := func(port corev1.ServicePort, ports []*corev1.ServicePort) []*corev1.ServicePort {
		for _, item := range ports {
			if item.Name == port.Name {
				item.Port = port.Port
				item.Protocol = port.Protocol
				item.TargetPort = intstr.IntOrString{
					Type:   0,
					IntVal: port.TargetPort.IntVal,
				}
			}
		}
		return ports
	}

	servicePorts := GenerateDefaultServicePorts()
	for _, port := range ports {
		mergePorts(port, servicePorts)
	}
	return ConvertPorts(servicePorts)
}

func ConvertPorts(ports []*corev1.ServicePort) []corev1.ServicePort {
	var res []corev1.ServicePort
	for _, port := range ports {
		res = append(res, *port)
	}
	return res
}
