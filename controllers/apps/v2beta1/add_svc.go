package v2beta1

import (
	"context"

	emperror "emperror.dev/errors"
	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addSvc struct {
	*EMQXReconciler
}

func (a *addSvc) reconcile(ctx context.Context, logger logr.Logger, instance *appsv2beta1.EMQX, _ innerReq.RequesterInterface) subResult {
	configMap := &corev1.ConfigMap{}
	if err := a.Client.Get(ctx, instance.ConfigsNamespacedName(), configMap); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get configmap")}
	}

	configStr := configMap.Data["emqx.conf"]
	resources := []client.Object{generateHeadlessService(instance)}
	if dashboard := generateDashboardService(instance, configStr); dashboard != nil {
		resources = append(resources, dashboard)
	}
	if listeners := generateListenerService(instance, configStr); listeners != nil {
		resources = append(resources, listeners)
	}

	if err := a.CreateOrUpdateList(ctx, a.Scheme, logger, instance, resources); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to create or update services")}
	}
	return subResult{}
}

func generateHeadlessService(instance *appsv2beta1.EMQX) *corev1.Service {
	headlessSvc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: instance.Namespace,
			Name:      instance.HeadlessServiceNamespacedName().Name,
			Labels:    appsv2beta1.CloneAndMergeMap(appsv2beta1.DefaultLabels(instance), instance.Labels),
		},
		Spec: corev1.ServiceSpec{
			Type:                     corev1.ServiceTypeClusterIP,
			ClusterIP:                corev1.ClusterIPNone,
			SessionAffinity:          corev1.ServiceAffinityNone,
			PublishNotReadyAddresses: true,
			Selector:                 appsv2beta1.DefaultCoreLabels(instance),
			Ports: []corev1.ServicePort{
				{
					Name:       "erlang-dist",
					Port:       4370,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(4370),
				},
				{
					Name:       "gen-rpc",
					Port:       5369,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(5369),
				},
			},
		},
	}
	return headlessSvc
}

func generateDashboardService(instance *appsv2beta1.EMQX, configStr string) *corev1.Service {
	svc := &corev1.Service{}
	if instance.Spec.DashboardServiceTemplate != nil {
		if !*instance.Spec.DashboardServiceTemplate.Enabled {
			return nil
		}
		svc.ObjectMeta = *instance.Spec.DashboardServiceTemplate.ObjectMeta.DeepCopy()
		svc.Spec = *instance.Spec.DashboardServiceTemplate.Spec.DeepCopy()
	}

	ports, _ := appsv2beta1.GetDashboardServicePort(configStr)
	if len(ports) == 0 {
		return nil
	}

	svc.Spec.Ports = appsv2beta1.MergeServicePorts(svc.Spec.Ports, ports)
	svc.Spec.Selector = appsv2beta1.DefaultCoreLabels(instance)

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.Namespace,
			Name:        instance.DashboardServiceNamespacedName().Name,
			Labels:      appsv2beta1.CloneAndMergeMap(appsv2beta1.DefaultLabels(instance), svc.ObjectMeta.Labels),
			Annotations: svc.ObjectMeta.Annotations,
		},
		Spec: svc.Spec,
	}
}

func generateListenerService(instance *appsv2beta1.EMQX, configStr string) *corev1.Service {
	svc := &corev1.Service{}
	if instance.Spec.ListenersServiceTemplate != nil {
		if !*instance.Spec.ListenersServiceTemplate.Enabled {
			return nil
		}
		svc.ObjectMeta = *instance.Spec.ListenersServiceTemplate.ObjectMeta.DeepCopy()
		svc.Spec = *instance.Spec.ListenersServiceTemplate.Spec.DeepCopy()
	}

	ports, _ := appsv2beta1.GetListenersServicePorts(configStr)
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

	svc.Spec.Ports = appsv2beta1.MergeServicePorts(
		svc.Spec.Ports,
		ports,
	)
	svc.Spec.Selector = appsv2beta1.DefaultCoreLabels(instance)
	if appsv2beta1.IsExistReplicant(instance) {
		svc.Spec.Selector = appsv2beta1.DefaultReplicantLabels(instance)
	}
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.Namespace,
			Name:        instance.ListenersServiceNamespacedName().Name,
			Labels:      appsv2beta1.CloneAndMergeMap(appsv2beta1.DefaultLabels(instance), svc.ObjectMeta.Labels),
			Annotations: svc.ObjectMeta.Annotations,
		},
		Spec: svc.Spec,
	}
}
