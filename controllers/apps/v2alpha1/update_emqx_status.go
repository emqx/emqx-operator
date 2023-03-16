package v2alpha1

import (
	"context"
	"encoding/json"

	emperror "emperror.dev/errors"
	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type updateStatus struct {
	*EMQXReconciler
}

func (u *updateStatus) reconcile(ctx context.Context, instance *appsv2alpha1.EMQX, p *portForwardAPI) subResult {
	var err error
	instance, err = u.updateStatus(ctx, instance, p)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to update status")}
	}
	if err := u.Client.Status().Update(ctx, instance); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to update status")}
	}
	return subResult{}
}

func (u *updateStatus) updateStatus(ctx context.Context, instance *appsv2alpha1.EMQX, p *portForwardAPI) (*appsv2alpha1.EMQX, error) {
	var emqxNodes []appsv2alpha1.EMQXNode
	var existedSts *appsv1.StatefulSet = &appsv1.StatefulSet{}
	var existedDeploy *appsv1.Deployment = &appsv1.Deployment{}
	var err error

	err = u.Client.Get(ctx, types.NamespacedName{Name: instance.Spec.CoreTemplate.Name, Namespace: instance.Namespace}, existedSts)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return instance, nil
		}
		return nil, emperror.Wrap(err, "failed to get existed statefulSet")
	}

	deploymentList := &appsv1.DeploymentList{}
	_ = u.Client.List(ctx, deploymentList,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
	)
	dList := handlerDeploymentList(deploymentList)
	if len(dList) > 0 {
		existedDeploy = dList[len(dList)-1]
	}

	emqxNodes, err = getNodeStatuesByAPI(p)
	if err != nil {
		u.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedToGetNodeStatuses", err.Error())
	}

	emqxStatusMachine := newEMQXStatusMachine(instance)
	emqxStatusMachine.CheckNodeCount(emqxNodes)
	emqxStatusMachine.NextStatus(existedSts, existedDeploy)
	return emqxStatusMachine.GetEMQX(), nil
}

func getNodeStatuesByAPI(p *portForwardAPI) ([]appsv2alpha1.EMQXNode, error) {
	resp, body, err := p.requestAPI("GET", "api/v5/nodes", nil)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get API api/v5/nodes")
	}
	if resp.StatusCode != 200 {
		return nil, emperror.Errorf("failed to get API %s, status : %s, body: %s", "api/v5/nodes", resp.Status, body)
	}

	nodeStatuses := []appsv2alpha1.EMQXNode{}
	if err := json.Unmarshal(body, &nodeStatuses); err != nil {
		return nil, emperror.Wrap(err, "failed to unmarshal node statuses")
	}
	return nodeStatuses, nil
}
