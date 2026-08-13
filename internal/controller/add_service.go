package controller

import (
	"fmt"
	"slices"
	"strings"

	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v3beta1"
	"github.com/emqx/emqx-operator/internal/controller/config"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	"github.com/emqx/emqx-operator/internal/emqx/api"
	"github.com/emqx/emqx-operator/internal/requester"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Responsibilities:
//   - Sets up Service resources: for MQTT/Gateway EMQX listeners, and for the API/Dashboard endpoint.
//     Exposes enabled MQTT listeners regardless of transient status, and enabled listeners of running gateways.
//   - Switches target set of pods on readiness change, see `listenerServiceSelector`.
type addService struct {
	*EMQXReconciler
}

func (a *addService) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	// Postpone if there are no usable cores yet.
	// Should proceed once one core replica is Ready.
	req := r.preferredCoreRequester()
	if req == nil {
		return reconcilePostpone()
	}

	resources := []client.Object{}

	if instance.Spec.DashboardServiceTemplate.IsEnabled() {
		configStr, err := api.Configs(req)
		if err != nil {
			return subResult{err: emperror.Wrap(err, "failed to get emqx configs")}
		}
		conf, err := config.EMQXConfig(configStr)
		if err != nil {
			return subResult{err: emperror.Wrap(err, "failed to load emqx config")}
		}
		service := generateDashboardService(instance, conf)
		if service != nil {
			resources = append(resources, service)
		}
	}

	if instance.Spec.ListenersServiceTemplate.IsEnabled() {
		ports, err := discoverListenerServicePorts(req)
		if err != nil {
			return subResult{err: err}
		}
		service := generateListenerService(r, instance, ports)
		resources = append(resources, service)
	}

	if err := a.CreateOrUpdateList(r.ctx, a.Scheme, r.log, instance, resources); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to create or update services")}
	}
	return subResult{}
}

func generateDashboardService(instance *crd.EMQX, conf *config.EMQX) *corev1.Service {
	meta := &crd.TemplateObjectMeta{}
	spec := &corev1.ServiceSpec{}
	if instance.Spec.DashboardServiceTemplate != nil {
		meta = instance.Spec.DashboardServiceTemplate.TemplateObjectMeta.DeepCopy()
		spec = instance.Spec.DashboardServiceTemplate.Spec.DeepCopy()
	}

	ports := conf.GetDashboardServicePorts()
	if len(ports) == 0 {
		return nil
	}

	spec.Ports = util.AppendMissingServicePorts(spec.Ports, ports)
	spec.Selector = instance.DefaultLabelsWith(crd.CoreLabels())

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.Namespace,
			Name:        instance.DashboardServiceNamespacedName().Name,
			Labels:      instance.DefaultLabelsWith(meta.Labels),
			Annotations: meta.Annotations,
		},
		Spec: *spec,
	}
}

func generateListenerService(
	r *reconcileRound,
	instance *crd.EMQX,
	ports []corev1.ServicePort,
) *corev1.Service {
	meta := &crd.TemplateObjectMeta{}
	spec := &corev1.ServiceSpec{}
	if instance.Spec.ListenersServiceTemplate != nil {
		meta = instance.Spec.ListenersServiceTemplate.TemplateObjectMeta.DeepCopy()
		spec = instance.Spec.ListenersServiceTemplate.Spec.DeepCopy()
	}

	spec.Ports = util.AppendMissingServicePorts(spec.Ports, ports)
	spec.Selector = listenerServiceSelector(r, instance)
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.Namespace,
			Name:        instance.ListenersServiceNamespacedName().Name,
			Labels:      instance.DefaultLabelsWith(meta.Labels),
			Annotations: meta.Annotations,
		},
		Spec: *spec,
	}
}

func discoverListenerServicePorts(req requester.RequesterInterface) ([]corev1.ServicePort, error) {
	mqttListeners, err := api.Listeners(req)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get MQTT listeners")
	}

	gateways, err := api.Gateways(req)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get gateway overview")
	}

	ports := make([]corev1.ServicePort, 0, len(mqttListeners))
	for _, listener := range mqttListeners {
		if !listener.Enable {
			continue
		}
		port, err := listenerServicePort(listener, "")
		if err != nil {
			return nil, err
		}
		ports = append(ports, port)
	}

	for _, gateway := range gateways {
		if gateway.Status != api.GatewayStatusRunning {
			continue
		}
		listeners, err := api.GatewayListeners(req, gateway.Name)
		if err != nil {
			return nil, emperror.Wrapf(err, "failed to get listeners for gateway %q", gateway.Name)
		}
		for _, listener := range listeners {
			if !listener.Enable {
				continue
			}
			port, err := listenerServicePort(listener, gateway.Name)
			if err != nil {
				return nil, err
			}
			ports = append(ports, port)
		}
	}

	slices.SortFunc(ports, util.CompareServicePort)

	return ports, nil
}

func listenerServicePort(listener api.Listener, gateway string) (corev1.ServicePort, error) {
	identity := fmt.Sprintf("listener %s:%s", listener.Type, listener.Name)
	name := fmt.Sprintf("%s-%s", listener.Type, listener.Name)
	if gateway != "" {
		identity = fmt.Sprintf("gateway %s listener %s:%s", gateway, listener.Type, listener.Name)
		name = fmt.Sprintf("%s-%s-%s", gateway, listener.Type, listener.Name)
	}

	port, err := util.ParseBindPort(listener.Bind)
	if err != nil {
		return corev1.ServicePort{}, emperror.Wrapf(err, "invalid bind for %s", identity)
	}
	protocol, err := listenerServiceProtocol(listener.Type)
	if err != nil {
		return corev1.ServicePort{}, emperror.Wrapf(err, "invalid protocol for %s", identity)
	}
	return corev1.ServicePort{
		Name:        name,
		Protocol:    protocol,
		AppProtocol: listenerAppProtocol(listener.Type),
		Port:        port,
		TargetPort:  intstr.FromInt(int(port)),
	}, nil
}

func listenerServiceProtocol(listenerType string) (corev1.Protocol, error) {
	switch strings.ToLower(listenerType) {
	case "quic", "udp", "dtls":
		return corev1.ProtocolUDP, nil
	case "tcp", "ssl", "ws", "wss":
		return corev1.ProtocolTCP, nil
	default:
		return "", fmt.Errorf("unsupported listener type %q", listenerType)
	}
}

func listenerAppProtocol(listenerType string) *string {
	switch strings.ToLower(listenerType) {
	case "ws":
		return ptr.To("kubernetes.io/ws")
	case "wss":
		return ptr.To("kubernetes.io/wss")
	default:
		return nil
	}
}

// listenerServiceSelector chooses Service endpoints for MQTT/TLS listeners.
// ReplicaSet readiness uses Status.ReadyReplicas only (not EMQX node status).
//  1. No replicants in spec -> cores serve.
//  2. If there are ready replicants, pods belonging to all replicant sets serve.
//  3. If no replicants can serve traffic, cores serve.
func listenerServiceSelector(r *reconcileRound, instance *crd.EMQX) map[string]string {
	if instance.Spec.HasReplicants() && r.state.numReadyReplicants() > 0 {
		return instance.DefaultLabelsWith(crd.ReplicantLabels())
	}
	return instance.DefaultLabelsWith(crd.CoreLabels())
}
