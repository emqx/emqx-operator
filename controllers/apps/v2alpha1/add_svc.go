package v2alpha1

import (
	"context"

	emperror "emperror.dev/errors"
	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
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
