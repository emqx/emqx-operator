package v2alpha2

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	emperror "emperror.dev/errors"
	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addListener struct {
	*EMQXReconciler
}

func (a *addListener) reconcile(ctx context.Context, instance *appsv2alpha2.EMQX, r innerReq.RequesterInterface) subResult {
	if !instance.Status.IsConditionTrue(appsv2alpha2.CodeNodesReady) {
		return subResult{}
	}

	pods := a.getPodList(ctx, instance)
	if len(pods) == 0 {
		return subResult{}
	}

	resources := []client.Object{}
	svc := generateListenerService(instance, a.getServicePorts(instance, r))
	if svc == nil {
		return subResult{}
	}

	resources = append(resources,
		generateEndpoints(svc, pods),
		svc,
	)

	if err := a.CreateOrUpdateList(instance, a.Scheme, resources); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to create or update listener service and endpoints")}
	}

	return subResult{}
}

func (a *addListener) getPodList(ctx context.Context, instance *appsv2alpha2.EMQX) []corev1.Pod {
	// labels := appsv2alpha2.AddLabel(instance.Spec.CoreTemplate.Labels, appsv1.ControllerRevisionHashLabelKey, instance.Status.CoreNodesStatus.CurrentRevision)
	labels := instance.Spec.CoreTemplate.Labels
	if isExistReplicant(instance) {
		labels = appsv2alpha2.AddLabel(instance.Spec.ReplicantTemplate.Labels, appsv2alpha2.PodTemplateHashLabelKey, instance.Status.ReplicantNodesStatus.CurrentRevision)
	}

	podList := &corev1.PodList{}
	_ = a.Client.List(ctx, podList,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(labels),
	)
	return podList.Items
}

func (a *addListener) getServicePorts(instance *appsv2alpha2.EMQX, r innerReq.RequesterInterface) []corev1.ServicePort {
	listenerPorts, err := getAllListenersByAPI(r)
	if err != nil {
		a.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedToGetListenerPorts", err.Error())
	}

	return listenerPorts
}

func generateListenerService(instance *appsv2alpha2.EMQX, ports []corev1.ServicePort) *corev1.Service {
	listener := instance.Spec.ListenersServiceTemplate.DeepCopy()
	// We don't need to set the selector for the service
	// because the Operator will manager the endpoints
	// please check https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors
	listener.Spec.Selector = map[string]string{}
	listener.Spec.Ports = appsv2alpha2.MergeServicePorts(
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

func generateEndpoints(svc *corev1.Service, pods []corev1.Pod) *corev1.Endpoints {
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

func getAllListenersByAPI(r innerReq.RequesterInterface) ([]corev1.ServicePort, error) {
	ports, err := getListenerPortsByAPI(r, "api/v5/listeners")
	if err != nil {
		return nil, err
	}

	gateways, err := getGatewaysByAPI(r)
	if err != nil {
		return nil, err
	}

	for _, gateway := range gateways {
		if strings.ToLower(gateway.Status) == "running" {
			apiPath := fmt.Sprintf("api/v5/gateways/%s/listeners", gateway.Name)
			gatewayPorts, err := getListenerPortsByAPI(r, apiPath)
			if err != nil {
				return nil, err
			}
			ports = append(ports, gatewayPorts...)
		}
	}

	return ports, nil
}

func getGatewaysByAPI(r innerReq.RequesterInterface) ([]emqxGateway, error) {
	resp, body, err := r.Request("GET", "api/v5/gateways", nil)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get API api/v5/gateways")
	}
	if resp.StatusCode != 200 {
		return nil, emperror.Errorf("failed to get API %s, status : %s, body: %s", "api/v5/gateways", resp.Status, body)
	}
	gateway := []emqxGateway{}
	if err := json.Unmarshal(body, &gateway); err != nil {
		return nil, emperror.Wrap(err, "failed to parse gateways")
	}
	return gateway, nil
}

func getListenerPortsByAPI(r innerReq.RequesterInterface, apiPath string) ([]corev1.ServicePort, error) {
	resp, body, err := r.Request("GET", apiPath, nil)
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
