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
	resources := []client.Object{
		generateHeadlessService(instance),
		generateDashboardService(instance),
	}

	configMap := &corev1.ConfigMap{}
	if err := a.Client.Get(ctx, types.NamespacedName{
		Name:      instance.ConfigsNamespacedName().Name,
		Namespace: instance.Namespace,
	}, configMap); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get configmap")}
	}

	ports, err := appsv2beta1.GetListenersServicePorts(configMap.Data["emqx.conf"])
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get listeners service ports")}
	}

	listeners := generateListenerService(instance, ports)
	if listeners != nil {
		resources = append(resources, listeners)
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

func generateDashboardService(instance *appsv2beta1.EMQX) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.Namespace,
			Name:        instance.Spec.DashboardServiceTemplate.Name,
			Labels:      instance.Spec.DashboardServiceTemplate.Labels,
			Annotations: instance.Spec.DashboardServiceTemplate.Annotations,
		},
		Spec: instance.Spec.DashboardServiceTemplate.Spec,
	}
}

func generateListenerService(instance *appsv2beta1.EMQX, ports []corev1.ServicePort) *corev1.Service {
	listener := instance.Spec.ListenersServiceTemplate.DeepCopy()
	listener.Spec.Ports = appsv2beta1.MergeServicePorts(
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
