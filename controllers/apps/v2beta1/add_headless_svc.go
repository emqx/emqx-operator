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

type addHeadlessSvc struct {
	*EMQXReconciler
}

func (a *addHeadlessSvc) reconcile(ctx context.Context, logger logr.Logger, instance *appsv2beta1.EMQX, _ innerReq.RequesterInterface) subResult {
	if err := a.CreateOrUpdateList(ctx, a.Scheme, logger, instance, []client.Object{generateHeadlessService(instance)}); err != nil {
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
