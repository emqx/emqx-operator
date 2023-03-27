package v1beta4

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	emperror "emperror.dev/errors"
	"github.com/emqx/emqx-operator/apis/apps/v1beta4"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addListener struct {
	*EmqxReconciler
	*portForwardAPI
}

func (a addListener) reconcile(ctx context.Context, instance v1beta4.Emqx, _ ...any) subResult {
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

func (a addListener) generateListenerService(ctx context.Context, instance v1beta4.Emqx) *corev1.Service {
	// ignore error, because if statefulSet is not created, the listener port will be not found
	listenerPorts, _ := a.getListenerPortsByAPI()
	// We don't need to set the selector for the service
	// because the Operator will manager the endpointSlice
	// please check https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors
	serviceTemplate := instance.GetSpec().GetServiceTemplate()
	serviceTemplate.Spec.Selector = map[string]string{}
	serviceTemplate.Spec.Ports = v1beta4.MergeServicePorts(
		instance.GetSpec().GetServiceTemplate().Spec.Ports, listenerPorts,
	)
	if len(instance.GetSpec().GetServiceTemplate().Spec.Ports) == 0 {
		return nil
	}
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.GetNamespace(),
			Name:        instance.GetSpec().GetServiceTemplate().Name,
			Labels:      instance.GetSpec().GetServiceTemplate().Labels,
			Annotations: instance.GetSpec().GetServiceTemplate().Annotations,
		},
		Spec: serviceTemplate.Spec,
	}
}

func (a addListener) generateEndpointSlice(ctx context.Context, instance v1beta4.Emqx, svc *corev1.Service) *discoveryv1.EndpointSlice {
	endpoints := a.getEndpoints(ctx, instance)
	if len(endpoints) == 0 {
		return nil
	}
	addressType := parseIP(endpoints[0].Addresses[0])

	labels := instance.GetSpec().GetServiceTemplate().Labels
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
			Name:     pointer.String(port.Name),
			Port:     pointer.Int32(port.Port),
			Protocol: &[]corev1.Protocol{port.Protocol}[0],
		})
	}

	return endpointSlice
}

func (a addListener) getEndpoints(ctx context.Context, instance v1beta4.Emqx) []discoveryv1.Endpoint {
	podList := &corev1.PodList{}
	_ = a.Client.List(ctx, podList,
		client.InNamespace(instance.GetNamespace()),
		client.MatchingLabels(instance.GetLabels()),
	)

	endpoints := []discoveryv1.Endpoint{}
	for _, pod := range podList.Items {
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				endpoints = append(endpoints, discoveryv1.Endpoint{
					Addresses: []string{pod.Status.PodIP},
					Conditions: discoveryv1.EndpointConditions{
						Ready:   pointer.Bool(true),
						Serving: pointer.Bool(true),
					},
					NodeName: pointer.String(pod.Spec.NodeName),
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

func (a addListener) getListenerPortsByAPI() ([]corev1.ServicePort, error) {
	type emqxListener struct {
		Protocol string `json:"protocol"`
		ListenOn string `json:"listen_on"`
	}

	type emqxListeners struct {
		Node      string         `json:"node"`
		Listeners []emqxListener `json:"listeners"`
	}

	intersection := func(listeners1 []emqxListener, listeners2 []emqxListener) []emqxListener {
		hSection := map[string]struct{}{}
		ans := make([]emqxListener, 0)
		for _, listener := range listeners1 {
			hSection[listener.ListenOn] = struct{}{}
		}
		for _, listener := range listeners2 {
			_, ok := hSection[listener.ListenOn]
			if ok {
				ans = append(ans, listener)
				delete(hSection, listener.ListenOn)
			}
		}
		return ans
	}

	_, body, err := a.portForwardAPI.requestAPI("GET", "api/v4/listeners", nil)
	if err != nil {
		return nil, err
	}

	listenerList := []emqxListeners{}
	data := gjson.GetBytes(body, "data")
	if err := json.Unmarshal([]byte(data.Raw), &listenerList); err != nil {
		return nil, emperror.Wrap(err, "failed to unmarshal node statuses")
	}

	var listeners []emqxListener
	if len(listenerList) == 1 {
		listeners = listenerList[0].Listeners
	} else {
		for i := 0; i < len(listenerList)-1; i++ {
			listeners = intersection(listenerList[i].Listeners, listenerList[i+1].Listeners)
		}
	}

	ports := []corev1.ServicePort{}
	for _, l := range listeners {
		var name string
		var protocol corev1.Protocol
		var strPort string
		var intPort int

		compile := regexp.MustCompile(".*(udp|dtls|sn).*")
		if compile.MatchString(l.Protocol) {
			protocol = corev1.ProtocolUDP
		} else {
			protocol = corev1.ProtocolTCP
		}

		if strings.Contains(l.ListenOn, ":") {
			_, strPort, err = net.SplitHostPort(l.ListenOn)
			if err != nil {
				strPort = l.ListenOn
			}
		} else {
			strPort = l.ListenOn
		}
		intPort, _ = strconv.Atoi(strPort)

		// Get name by protocol and port from API
		// protocol maybe like mqtt:wss:8084
		// protocol maybe like mqtt:tcp
		// We had to do something with the "protocol" to make it conform to the kubernetes service port name specification
		name = regexp.MustCompile(`:[\d]+`).ReplaceAllString(l.Protocol, "")
		name = strings.ReplaceAll(name, ":", "-")
		name = fmt.Sprintf("%s-%s", name, strPort)

		ports = append(ports, corev1.ServicePort{
			Name:       name,
			Protocol:   protocol,
			Port:       int32(intPort),
			TargetPort: intstr.FromInt(intPort),
		})
	}
	return ports, nil
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
