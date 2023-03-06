package v2alpha1

import (
	"context"
	"time"

	emperror "emperror.dev/errors"
	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addListener struct {
	*EMQXReconciler
}

func (a *addListener) reconcile(ctx context.Context, instance *appsv2alpha1.EMQX) subResult {
	if !instance.Status.IsRunning() && !instance.Status.IsCoreNodesReady() {
		return subResult{result: ctrl.Result{RequeueAfter: time.Second}}
	}

	resources := []client.Object{}
	svc := a.generateListenerService(ctx, instance)
	if svc == nil {
		return subResult{}
	}
	resources = append(resources, svc)

	endpointSlice := a.generateEndpointSlice(ctx, instance, svc)
	if endpointSlice == nil {
		return subResult{}
	}
	resources = append(resources, endpointSlice)

	if err := a.CreateOrUpdateList(instance, a.Scheme, resources); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to create or update listener service and endpointSlice")}
	}

	return subResult{}
}

func (a *addListener) generateListenerService(ctx context.Context, instance *appsv2alpha1.EMQX) *corev1.Service {
	// We don't need to set the selector for the service
	// because the Operator will manager the endpointSlice
	// please check https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors
	instance.Spec.ListenersServiceTemplate.Spec.Selector = map[string]string{}
	instance.Spec.ListenersServiceTemplate.Spec.Ports = appsv2alpha1.MergeServicePorts(
		instance.Spec.ListenersServiceTemplate.Spec.Ports,
		a.getServicePorts(ctx, instance),
	)
	if len(instance.Spec.ListenersServiceTemplate.Spec.Ports) == 0 {
		return nil
	}
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.Namespace,
			Name:        instance.Spec.ListenersServiceTemplate.Name,
			Labels:      instance.Spec.ListenersServiceTemplate.Labels,
			Annotations: instance.Spec.ListenersServiceTemplate.Annotations,
		},
		Spec: instance.Spec.ListenersServiceTemplate.Spec,
	}
}

func (a *addListener) generateEndpointSlice(ctx context.Context, instance *appsv2alpha1.EMQX, svc *corev1.Service) *discoveryv1.EndpointSlice {
	endpoints := a.getEndpoints(ctx, instance)
	if len(endpoints) == 0 {
		return nil
	}
	addressType := parseIP(endpoints[0].Addresses[0])

	labels := instance.Spec.ListenersServiceTemplate.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["kubernetes.io/service-name"] = svc.Name

	endpointSlice := &discoveryv1.EndpointSlice{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "discovery.k8s.io/v1",
			Kind:       "EndpointSlice",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   svc.Namespace,
			Name:        svc.Name,
			Annotations: svc.Annotations,
			Labels:      labels,
		},
		AddressType: addressType,
		Endpoints:   endpoints,
	}

	for _, port := range svc.Spec.Ports {
		endpointSlice.Ports = append(endpointSlice.Ports, discoveryv1.EndpointPort{
			Name:     &[]string{port.Name}[0],
			Port:     &[]int32{port.Port}[0],
			Protocol: &[]corev1.Protocol{port.Protocol}[0],
		})
	}

	return endpointSlice
}

func (a *addListener) getServicePorts(ctx context.Context, instance *appsv2alpha1.EMQX) []corev1.ServicePort {
	sts := &appsv1.StatefulSet{}
	_ = a.Client.Get(ctx, types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      instance.Spec.CoreTemplate.Name,
	}, sts)

	listenerPorts, err := newRequestAPI(a.EMQXReconciler, instance).getAllListenersByAPI(sts)
	if err != nil {
		a.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedToGetListenerPorts", err.Error())
	}

	return listenerPorts
}

func (a *addListener) getEndpoints(ctx context.Context, instance *appsv2alpha1.EMQX) []discoveryv1.Endpoint {
	podList := &corev1.PodList{}
	_ = a.Client.List(ctx, podList,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
	)

	endpoints := []discoveryv1.Endpoint{}
	for _, pod := range podList.Items {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}
		for _, node := range instance.Status.EMQXNodes {
			if node.Node == "emqx@"+pod.Status.PodIP {
				endpoints = append(endpoints, discoveryv1.Endpoint{
					Addresses: []string{pod.Status.PodIP},
					Conditions: discoveryv1.EndpointConditions{
						Ready:       &[]bool{true}[0],
						Serving:     &[]bool{true}[0],
						Terminating: &[]bool{false}[0],
					},
					NodeName: &[]string{pod.Spec.NodeName}[0],
					TargetRef: &corev1.ObjectReference{
						Kind:      pod.Kind,
						UID:       pod.UID,
						Name:      pod.Name,
						Namespace: pod.Namespace,
					},
				})
			}
		}
	}
	return endpoints
}

// ParseIP parses s as an IP address, returning the result.
func parseIP(s string) discoveryv1.AddressType {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '.':
			return discoveryv1.AddressTypeIPv4
		case ':':
			return discoveryv1.AddressTypeIPv6
		}
	}
	panic("unreachable")
}
