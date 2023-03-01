package v2alpha1

import (
	"context"

	emperror "emperror.dev/errors"
	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addSvc struct {
	*EMQXReconciler
}

func (a *addSvc) reconcile(ctx context.Context, instance *appsv2alpha1.EMQX) subResult {

	svcList := []client.Object{
		generateHeadlessService(instance),
		generateDashboardService(instance),
	}

	if instance.Status.IsRunning() || instance.Status.IsCoreNodesReady() {
		sts := &appsv1.StatefulSet{}
		_ = a.Client.Get(ctx, types.NamespacedName{
			Namespace: instance.Namespace,
			Name:      instance.Spec.CoreTemplate.Name,
		}, sts)

		listenerPorts, err := newRequestAPI(a.EMQXReconciler, instance).getAllListenersByAPI(sts)
		if err != nil {
			a.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedToGetListenerPorts", err.Error())
		}

		if listenersSvc := generateListenerService(instance, listenerPorts); listenersSvc != nil {
			svcList = append(svcList, listenersSvc)
		}
	}

	if err := a.CreateOrUpdateList(instance, a.Scheme, svcList); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to create or update service")}
	}

	return subResult{}
}

func generateHeadlessService(instance *appsv2alpha1.EMQX) *corev1.Service {
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

func generateDashboardService(instance *appsv2alpha1.EMQX) *corev1.Service {
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

func generateListenerService(instance *appsv2alpha1.EMQX, listenerPorts []corev1.ServicePort) *corev1.Service {
	instance.Spec.ListenersServiceTemplate.Spec.Selector = instance.Spec.ReplicantTemplate.Labels
	instance.Spec.ListenersServiceTemplate.Spec.Ports = appsv2alpha1.MergeServicePorts(
		instance.Spec.ListenersServiceTemplate.Spec.Ports,
		listenerPorts,
	)

	if len(instance.Spec.ListenersServiceTemplate.Spec.Ports) == 0 {
		return nil
	}

	// We don't need to set the selector for the service
	// because the Operator will manager the endpointSlice
	// please check https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors
	// instance.Spec.ListenersServiceTemplate.Spec.Selector = map[string]string{}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   instance.Namespace,
			Name:        instance.Spec.ListenersServiceTemplate.Name,
			Labels:      instance.Spec.ListenersServiceTemplate.Labels,
			Annotations: instance.Spec.ListenersServiceTemplate.Annotations,
		},
		Spec: instance.Spec.ListenersServiceTemplate.Spec,
	}
}
