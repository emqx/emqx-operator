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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addListener struct {
	*EmqxReconciler
	EmqxHttpAPI
}

func (a addListener) reconcile(ctx context.Context, instance v1beta4.Emqx, _ ...any) subResult {
	podList := &corev1.PodList{}
	_ = a.Client.List(ctx, podList,
		client.InNamespace(instance.GetNamespace()),
		client.MatchingLabels(instance.GetLabels()),
	)
	pods := []*corev1.Pod{}
	for _, p := range podList.Items {
		pod := p.DeepCopy()
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				pods = append(pods, pod)
			}
		}
	}
	if len(pods) == 0 {
		return subResult{}
	}

	// ignore error, because if statefulSet is not created, the listener port will be not found
	listenerPorts, _ := a.getListenerPortsByAPI()

	resources := []client.Object{}
	svc := generateListenerService(instance, listenerPorts)
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

func generateListenerService(instance v1beta4.Emqx, listenerPorts []corev1.ServicePort) *corev1.Service {
	serviceTemplate := instance.GetSpec().GetServiceTemplate()
	listener := serviceTemplate.DeepCopy()
	// We don't need to set the selector for the service
	// because the Operator will manager the endpoints
	// please check https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors
	listener.Spec.Selector = map[string]string{}
	listener.Spec.Ports = v1beta4.MergeServicePorts(
		listener.Spec.Ports, listenerPorts,
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
			Namespace:   instance.GetNamespace(),
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

	_, body, err := a.EmqxHttpAPI.Request("GET", "api/v4/listeners", nil)
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
