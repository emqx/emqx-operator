package controller

import (
	emperror "emperror.dev/errors"
	crdv2 "github.com/emqx/emqx-operator/api/v2"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	"github.com/emqx/emqx-operator/internal/emqx/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addService struct {
	*EMQXReconciler
}

func (a *addService) reconcile(r *reconcileRound, instance *crdv2.EMQX) subResult {
	req := r.oldestCoreRequester()
	if req == nil {
		return subResult{}
	}

	if !r.state.areCoresReady(instance) {
		return subResult{}
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
	if listeners := generateListenerService(instance, conf); listeners != nil {
		resources = append(resources, listeners)
	}

	if err := a.CreateOrUpdateList(r.ctx, a.Scheme, r.log, instance, resources); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to create or update services")}
	}
	return subResult{}
}

func generateDashboardService(instance *crdv2.EMQX, conf *config.EMQX) *corev1.Service {
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
	spec.Selector = instance.DefaultLabelsWith(crdv2.CoreLabels())

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

func generateListenerService(instance *crdv2.EMQX, conf *config.EMQX) *corev1.Service {
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
	spec.Selector = instance.DefaultLabelsWith(crdv2.CoreLabels())
	if instance.Spec.HasReplicants() && instance.Status.ReplicantNodesStatus.ReadyReplicas > 0 {
		spec.Selector = instance.DefaultLabelsWith(
			crdv2.ReplicantLabels(),
			map[string]string{
				crdv2.LabelPodTemplateHash: instance.Status.ReplicantNodesStatus.UpdateRevision,
			},
		)
	}
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
