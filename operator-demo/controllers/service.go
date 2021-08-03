package controllers

import (
	"Emqx/api/v1alpha1"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

func makeServiceFromSpec(instance *v1alpha1.Broker) *v1.Service {
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: instance.APIVersion,
					Kind:       instance.Kind,
					Name:       instance.Name,
					UID:        instance.UID,
				},
			},
		},
		Spec: makeServiceSpec(instance),
	}
	svc.Name = instance.Name
	svc.Namespace = instance.Namespace
	return svc
}

func makeServiceSpec(instance *v1alpha1.Broker) v1.ServiceSpec {

	servicePort := []v1.ServicePort{
		{
			Name:     serviceTcpName,
			Port:     serviceTcpPort,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: serviceTcpPort,
			},
		},
		{
			Name:     serviceTcpsName,
			Port:     serviceTcpsPort,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: serviceTcpsPort,
			},
		},
		{
			Name:     serviceWsName,
			Port:     serviceWsPort,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: serviceWsPort,
			},
		},
		{
			Name:     serviceWssName,
			Port:     serviceWssPort,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: serviceWssPort,
			},
		},
		{
			Name:     "dashboard",
			Port:     18083,
			Protocol: "TCP",
			TargetPort: intstr.IntOrString{
				Type:   0,
				IntVal: 18083,
			},
		},
	}
	serviceSpec := v1.ServiceSpec{
		Type:  v1.ServiceTypeLoadBalancer,
		Ports: servicePort,
		Selector: map[string]string{
			"app":    emqxName,
			emqxName: instance.Name,
		},
	}
	return serviceSpec
}
