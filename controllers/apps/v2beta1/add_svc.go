package v2beta1

import (
	"context"

	emperror "emperror.dev/errors"
	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addSvc struct {
	*EMQXReconciler
}

func (a *addSvc) reconcile(ctx context.Context, instance *appsv2beta1.EMQX, _ innerReq.RequesterInterface) subResult {
	configMap := &corev1.ConfigMap{}
	if err := a.Client.Get(ctx, types.NamespacedName{
		Name:      instance.ConfigsNamespacedName().Name,
		Namespace: instance.Namespace,
	}, configMap); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get configmap")}
	}

	configStr := configMap.Data["emqx.conf"]
	resources := []client.Object{
		generateHeadlessService(instance),
		generateDashboardService(instance, configStr),
		generateListenerService(instance, configStr),
	}

	if err := a.CreateOrUpdateList(instance, a.Scheme, resources); err != nil {
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
			Name:      instance.HeadlessServiceNamespacedName().Name,
			Namespace: instance.Namespace,
			Labels:    instance.Labels,
		},
		Spec: corev1.ServiceSpec{
			Type:                     corev1.ServiceTypeClusterIP,
			ClusterIP:                corev1.ClusterIPNone,
			SessionAffinity:          corev1.ServiceAffinityNone,
			PublishNotReadyAddresses: true,
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
			Selector: instance.Spec.CoreTemplate.Labels,
		},
	}
	return headlessSvc
}

func generateDashboardService(instance *appsv2beta1.EMQX, configStr string) *corev1.Service {
	port, err := appsv2beta1.GetDashboardServicePort(configStr)
	if err != nil {
		port = &corev1.ServicePort{
			Name:       "dashboard",
			Protocol:   corev1.ProtocolTCP,
			Port:       18083,
			TargetPort: intstr.Parse("18083"),
		}
	}

	svc := instance.Spec.DashboardServiceTemplate.DeepCopy()
	svc.Spec.Ports = appsv2beta1.MergeServicePorts(
		svc.Spec.Ports,
		[]corev1.ServicePort{
			*port,
		},
	)

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.Namespace,
			Name:        svc.Name,
			Labels:      svc.Labels,
			Annotations: svc.Annotations,
		},
		Spec: svc.Spec,
	}
}

func generateListenerService(instance *appsv2beta1.EMQX, configStr string) *corev1.Service {
	ports, err := appsv2beta1.GetListenersServicePorts(configStr)
	if err != nil {
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

	svc := instance.Spec.ListenersServiceTemplate.DeepCopy()
	svc.Spec.Ports = appsv2beta1.MergeServicePorts(
		svc.Spec.Ports,
		ports,
	)
	svc.Spec.Selector = instance.Spec.CoreTemplate.Labels
	if appsv2beta1.IsExistReplicant(instance) {
		svc.Spec.Selector = instance.Spec.ReplicantTemplate.Labels
	}
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.Namespace,
			Name:        svc.Name,
			Labels:      svc.Labels,
			Annotations: svc.Annotations,
		},
		Spec: svc.Spec,
	}
}
