package controller

import (
	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v3alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type addHeadlessService struct {
	*EMQXReconciler
}

func (a *addHeadlessService) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	if err := a.CreateOrUpdate(r.ctx, a.Scheme, r.log, instance, generateHeadlessService(instance)); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to create or update services")}
	}
	return subResult{}
}

func generateHeadlessService(instance *crd.EMQX) *corev1.Service {
	headlessSvc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: instance.Namespace,
			Name:      instance.HeadlessServiceNamespacedName().Name,
			Labels:    instance.DefaultLabelsWith(instance.Labels),
		},
		Spec: corev1.ServiceSpec{
			Type:                     corev1.ServiceTypeClusterIP,
			ClusterIP:                corev1.ClusterIPNone,
			SessionAffinity:          corev1.ServiceAffinityNone,
			PublishNotReadyAddresses: true,
			Selector:                 instance.DefaultLabelsWith(crd.CoreLabels()),
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
