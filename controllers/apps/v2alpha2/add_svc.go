package v2alpha2

import (
	"context"

	emperror "emperror.dev/errors"
	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addSvc struct {
	*EMQXReconciler
}

func (a *addSvc) reconcile(ctx context.Context, instance *appsv2alpha2.EMQX, _ innerReq.RequesterInterface) subResult {
	resources := []client.Object{
		generateHeadlessService(instance),
		generateDashboardService(instance),
	}

	configMap := &corev1.ConfigMap{}
	if err := a.Client.Get(ctx, types.NamespacedName{
		Name:      instance.BootstrapConfigNamespacedName().Name,
		Namespace: instance.Namespace,
	}, configMap); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get configmap")}
	}

	ports, err := appsv2alpha2.GetListenersServicePorts(configMap.Data["emqx.conf"])
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get listeners service ports")}
	}

	listeners := generateListenerService(instance, ports)
	if listeners != nil {
		resources = append(resources, listeners)
		if instance.Status.IsConditionTrue(appsv2alpha2.CoreNodesReady) {
			pods := a.getPodList(ctx, instance)
			if len(pods) > 0 {
				resources = append(resources, generateEndpoints(listeners, pods))
			}
		}
	}

	if err := a.CreateOrUpdateList(instance, a.Scheme, resources); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to create or update services")}
	}

	return subResult{}
}

func (a *addSvc) getPodList(ctx context.Context, instance *appsv2alpha2.EMQX) []corev1.Pod {
	labels := appsv2alpha2.AddLabel(
		instance.Spec.CoreTemplate.Labels,
		appsv2alpha2.PodTemplateHashLabelKey,
		instance.Status.CoreNodesStatus.CurrentRevision,
	)
	if isExistReplicant(instance) {
		labels = appsv2alpha2.AddLabel(
			instance.Spec.ReplicantTemplate.Labels,
			appsv2alpha2.PodTemplateHashLabelKey,
			instance.Status.ReplicantNodesStatus.CurrentRevision,
		)
	}

	podList := &corev1.PodList{}
	_ = a.Client.List(ctx, podList,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(labels),
	)

	list := []corev1.Pod{}
	for _, pod := range podList.Items {
		if pod.Status.PodIP != "" {
			list = append(list, pod)
		}
	}
	return list
}

func generateHeadlessService(instance *appsv2alpha2.EMQX) *corev1.Service {
	headlessSvc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.HeadlessServiceNamespacedName().Name,
			Namespace: instance.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type:                     corev1.ServiceTypeClusterIP,
			ClusterIP:                corev1.ClusterIPNone,
			SessionAffinity:          corev1.ServiceAffinityNone,
			PublishNotReadyAddresses: true,
			Ports: []corev1.ServicePort{
				{
					Name:       "ekka",
					Port:       4370,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(4370),
				},
			},
			Selector: instance.Spec.CoreTemplate.Labels,
		},
	}
	return headlessSvc
}

func generateDashboardService(instance *appsv2alpha2.EMQX) *corev1.Service {
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
