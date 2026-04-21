package controller

import (
	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v3beta1"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	"github.com/emqx/emqx-operator/internal/emqx/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Responsibilities:
// - Sets up Service resources: for MQTT/Gateway EMQX listeners, and for the API/Dashboard endpoint.
// - Switches target set of pods on readiness change, see `listenerServiceSelector`.
type addService struct {
	*EMQXReconciler
}

func (a *addService) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	// Postpone if there are no usable cores yet.
	// Should proceed once one core replica is Ready.
	req := r.oldestCoreRequester()
	if req == nil {
		return reconcilePostpone()
	}

	configStr, err := api.Configs(req)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get emqx configs by api")}
	}

	conf, err := config.EMQXConfig(configStr)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to load emqx config")}
	}

	resources := []client.Object{}
	if dashboard := generateDashboardService(instance, conf); dashboard != nil {
		resources = append(resources, dashboard)
	}
	if listeners := generateListenerService(r, instance, conf); listeners != nil {
		resources = append(resources, listeners)
	}

	if err := a.CreateOrUpdateList(r.ctx, a.Scheme, r.log, instance, resources); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to create or update services")}
	}
	return subResult{}
}

func generateDashboardService(instance *crd.EMQX, conf *config.EMQX) *corev1.Service {
	meta := &metav1.ObjectMeta{}
	spec := &corev1.ServiceSpec{}
	if instance.Spec.DashboardServiceTemplate != nil {
		if !instance.Spec.DashboardServiceTemplate.IsEnabled() {
			return nil
		}
		meta = instance.Spec.DashboardServiceTemplate.ObjectMeta.DeepCopy()
		spec = instance.Spec.DashboardServiceTemplate.Spec.DeepCopy()
	}

	ports := conf.GetDashboardServicePorts()
	if len(ports) == 0 {
		return nil
	}

	spec.Ports = util.MergeServicePorts(spec.Ports, ports)
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

func generateListenerService(r *reconcileRound, instance *crd.EMQX, conf *config.EMQX) *corev1.Service {
	meta := &metav1.ObjectMeta{}
	spec := &corev1.ServiceSpec{}
	if instance.Spec.ListenersServiceTemplate != nil {
		if !instance.Spec.ListenersServiceTemplate.IsEnabled() {
			return nil
		}
		meta = instance.Spec.ListenersServiceTemplate.ObjectMeta.DeepCopy()
		spec = instance.Spec.ListenersServiceTemplate.Spec.DeepCopy()
	}

	ports := conf.GetListenersServicePorts()
	if len(ports) == 0 {
		ports = append(ports, []corev1.ServicePort{
			{
				Name:       "tcp-default",
				Protocol:   corev1.ProtocolTCP,
				Port:       1883,
				TargetPort: intstr.FromInt(1883),
			},
			{
				Name:       "ssl-default",
				Protocol:   corev1.ProtocolTCP,
				Port:       8883,
				TargetPort: intstr.FromInt(8883),
			},
			{
				Name:       "ws-default",
				Protocol:   corev1.ProtocolTCP,
				Port:       8083,
				TargetPort: intstr.FromInt(8083),
			},
			{
				Name:       "wss-default",
				Protocol:   corev1.ProtocolTCP,
				Port:       8084,
				TargetPort: intstr.FromInt(8084),
			},
		}...)
	}

	spec.Ports = util.MergeServicePorts(spec.Ports, ports)
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
