package v2alpha1

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	emperror "emperror.dev/errors"
	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addListener struct {
	*EMQXReconciler
}

func (a *addListener) reconcile(ctx context.Context, instance *appsv2alpha1.EMQX, p *portForwardAPI) subResult {
	if !instance.Status.IsRunning() && !instance.Status.IsCoreNodesReady() {
		return subResult{}
	}

	pods := a.getPodList(ctx, instance)
	if len(pods) == 0 {
		return subResult{}
	}

	resources := []client.Object{}
	svc := generateListenerService(instance, a.getServicePorts(instance, p))
	if svc == nil {
		return subResult{}
	}

	resources = append(resources, svc,
		generateEndpoints(svc, pods),
		generateEndpointSlice(svc, pods),
	)

	if err := a.CreateOrUpdateList(instance, a.Scheme, resources); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to create or update listener service and endpointSlice")}
	}

	return subResult{}
}

func (a *addListener) getPodList(ctx context.Context, instance *appsv2alpha1.EMQX) []*corev1.Pod {
	dList := getDeploymentList(ctx, a.Client,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
	)
	if len(dList) == 0 {
		return nil
	}
	currentDeployment := dList[len(dList)-1]

	podMap := getPodMap(ctx, a.Client,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
	)
	return podMap[currentDeployment.UID]
}

func (a *addListener) getServicePorts(instance *appsv2alpha1.EMQX, p *portForwardAPI) []corev1.ServicePort {
	listenerPorts, err := getAllListenersByAPI(p)
	if err != nil {
		a.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedToGetListenerPorts", err.Error())
	}

	return listenerPorts
}

func generateListenerService(instance *appsv2alpha1.EMQX, ports []corev1.ServicePort) *corev1.Service {
	listener := instance.Spec.ListenersServiceTemplate.DeepCopy()
	// We don't need to set the selector for the service
	// because the Operator will manager the endpointSlice
	// please check https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors
	listener.Spec.Selector = map[string]string{}
	listener.Spec.Ports = appsv2alpha1.MergeServicePorts(
		listener.Spec.Ports,
		ports,
	)
	if len(listener.Spec.Ports) == 0 {
		return nil
	}
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.Namespace,
			Name:        listener.Name,
			Labels:      listener.Labels,
			Annotations: listener.Annotations,
		},
		Spec: listener.Spec,
	}
}

func generateEndpoints(svc *corev1.Service, pods []*corev1.Pod) *corev1.Endpoints {
	subSet := corev1.EndpointSubset{}
	for _, port := range svc.Spec.Ports {
		subSet.Ports = append(subSet.Ports, corev1.EndpointPort{
			Name:     port.Name,
			Port:     port.Port,
			Protocol: port.Protocol,
		})
	}
	for _, p := range pods {
		pod := p.DeepCopy()
		subSet.Addresses = append(subSet.Addresses, corev1.EndpointAddress{
			IP:       pod.Status.PodIP,
			NodeName: pointer.String(pod.Spec.NodeName),
			TargetRef: &corev1.ObjectReference{
				Kind:      "Pod",
				Namespace: pod.Namespace,
				Name:      pod.Name,
				UID:       pod.UID,
			},
		})

	}

	return &corev1.Endpoints{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Endpoints",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   svc.Namespace,
			Name:        svc.Name,
			Annotations: svc.Annotations,
			Labels:      svc.Labels,
		},
		Subsets: []corev1.EndpointSubset{subSet},
	}
}

func generateEndpointSlice(svc *corev1.Service, pods []*corev1.Pod) *discoveryv1.EndpointSlice {
	ports := []discoveryv1.EndpointPort{}
	for _, port := range svc.Spec.Ports {
		ports = append(ports, discoveryv1.EndpointPort{
			Name:     pointer.String(port.Name),
			Port:     pointer.Int32(port.Port),
			Protocol: &[]corev1.Protocol{port.Protocol}[0],
		})
	}

	endpoints := []discoveryv1.Endpoint{}
	for _, p := range pods {
		pod := p.DeepCopy()
		endpoints = append(endpoints, discoveryv1.Endpoint{
			Addresses: []string{pod.Status.PodIP},
			Conditions: discoveryv1.EndpointConditions{
				Ready:   pointer.Bool(true),
				Serving: pointer.Bool(true),
			},
			NodeName: pointer.String(pod.Spec.NodeName),
			TargetRef: &corev1.ObjectReference{
				Kind:      "Pod",
				UID:       pod.UID,
				Name:      pod.Name,
				Namespace: pod.Namespace,
			},
		})
	}

	labels := svc.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["kubernetes.io/service-name"] = svc.Name

	return &discoveryv1.EndpointSlice{
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
		AddressType: parseIP(endpoints[0].Addresses[0]),
		Endpoints:   endpoints,
		Ports:       ports,
	}
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

// Access EMQX API to get all listeners
type emqxGateway struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type emqxListener struct {
	Enable bool   `json:"enable"`
	ID     string `json:"id"`
	Bind   string `json:"bind"`
	Type   string `json:"type"`
}

func getAllListenersByAPI(p *portForwardAPI) ([]corev1.ServicePort, error) {
	ports, err := getListenerPortsByAPI(p, "api/v5/listeners")
	if err != nil {
		return nil, err
	}

	gateways, err := getGatewaysByAPI(p)
	if err != nil {
		return nil, err
	}

	for _, gateway := range gateways {
		if strings.ToLower(gateway.Status) == "running" {
			apiPath := fmt.Sprintf("api/v5/gateway/%s/listeners", gateway.Name)
			gatewayPorts, err := getListenerPortsByAPI(p, apiPath)
			if err != nil {
				return nil, err
			}
			ports = append(ports, gatewayPorts...)
		}
	}

	return ports, nil
}

func getGatewaysByAPI(p *portForwardAPI) ([]emqxGateway, error) {
	resp, body, err := p.requestAPI("GET", "api/v5/gateway", nil)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get API api/v5/gateway")
	}
	if resp.StatusCode != 200 {
		return nil, emperror.Errorf("failed to get API %s, status : %s, body: %s", "api/v5/gateway", resp.Status, body)
	}
	gateway := []emqxGateway{}
	if err := json.Unmarshal(body, &gateway); err != nil {
		return nil, emperror.Wrap(err, "failed to parse gateway")
	}
	return gateway, nil
}

func getListenerPortsByAPI(p *portForwardAPI, apiPath string) ([]corev1.ServicePort, error) {
	resp, body, err := p.requestAPI("GET", apiPath, nil)
	if err != nil {
		return nil, emperror.Wrapf(err, "failed to get API %s", apiPath)
	}
	if resp.StatusCode != 200 {
		return nil, emperror.Errorf("failed to get API %s, status : %s, body: %s", apiPath, resp.Status, body)
	}
	ports := []corev1.ServicePort{}
	listeners := []emqxListener{}
	if err := json.Unmarshal(body, &listeners); err != nil {
		return nil, emperror.Wrap(err, "failed to parse listeners")
	}
	for _, listener := range listeners {
		if !listener.Enable {
			continue
		}

		var protocol corev1.Protocol
		compile := regexp.MustCompile(".*(udp|dtls|quic).*")
		if compile.MatchString(listener.Type) {
			protocol = corev1.ProtocolUDP
		} else {
			protocol = corev1.ProtocolTCP
		}

		_, strPort, _ := net.SplitHostPort(listener.Bind)
		intPort, _ := strconv.Atoi(strPort)

		ports = append(ports, corev1.ServicePort{
			Name:       strings.ReplaceAll(listener.ID, ":", "-"),
			Protocol:   protocol,
			Port:       int32(intPort),
			TargetPort: intstr.FromInt(intPort),
		})
	}
	return ports, nil
}
