package v2alpha1

import (
	"context"
	"time"

	emperror "emperror.dev/errors"
	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type updateStatus struct {
	*EMQXReconciler
}

func (u *updateStatus) reconcile(ctx context.Context, instance *appsv2alpha1.EMQX) subResult {
	var err error
	instance, err = u.updateStatus(instance)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to update status")}
	}
	if u.Client.Status().Update(ctx, instance) != nil {
		return subResult{err: emperror.Wrap(err, "failed to update status")}
	}
	if !instance.Status.IsRunning() {
		return subResult{result: ctrl.Result{RequeueAfter: time.Second}}
	}
	return subResult{}
}

func (u *updateStatus) updateStatus(instance *appsv2alpha1.EMQX) (*appsv2alpha1.EMQX, error) {
	var emqxNodes []appsv2alpha1.EMQXNode
	var existedSts *appsv1.StatefulSet = &appsv1.StatefulSet{}
	var existedDeploy *appsv1.Deployment = &appsv1.Deployment{}
	var err error

	err = u.Client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.CoreTemplate.Name, Namespace: instance.Namespace}, existedSts)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return instance, nil
		}
		return nil, emperror.Wrap(err, "failed to get existed statefulSet")
	}

	err = u.Client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.ReplicantTemplate.Name, Namespace: instance.Namespace}, existedDeploy)
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, emperror.Wrap(err, "failed to get existed deployment")
	}

	emqxNodes, err = newRequestAPI(u.EMQXReconciler, instance).getNodeStatuesByAPI(existedSts)
	if err != nil {
		u.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedToGetNodeStatuses", err.Error())
	}

	emqxStatusMachine := newEMQXStatusMachine(instance)
	emqxStatusMachine.CheckNodeCount(emqxNodes)
	emqxStatusMachine.NextStatus(existedSts, existedDeploy)
	return emqxStatusMachine.GetEMQX(), nil
}
