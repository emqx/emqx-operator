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

func (u *updateStatus) reconcile(ctx context.Context, instance *appsv2alpha1.EMQX, r Requester) subResult {
	var err error
	var emqxNodes []appsv2alpha1.EMQXNode
	var existedSts *appsv1.StatefulSet = &appsv1.StatefulSet{}
	var existedDeploy *appsv1.Deployment = &appsv1.Deployment{}

	if r != nil {
		if emqxNodes, err = getNodeStatuesByAPI(r); err != nil {
			u.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedToGetNodeStatuses", err.Error())
		}
	}

	if err = u.Client.Get(ctx, types.NamespacedName{Name: instance.Spec.CoreTemplate.Name, Namespace: instance.Namespace}, existedSts); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return subResult{err: emperror.Wrap(err, "failed to get existed statefulSet")}
		}
	}

	dList := getDeploymentList(ctx, u.Client,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(instance.Spec.ReplicantTemplate.Labels),
	)
	if len(dList) > 0 {
		existedDeploy = dList[len(dList)-1]
	}

	emqxStatusMachine := newEMQXStatusMachine(instance)
	emqxStatusMachine.UpdateNodeCount(emqxNodes)
	emqxStatusMachine.NextStatus(existedSts, existedDeploy)
	emqxStatusMachine.GetEMQX()

	if err := u.Client.Status().Update(ctx, instance); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to update status")}
	}
	return subResult{}
}

func getNodeStatuesByAPI(r Requester) ([]appsv2alpha1.EMQXNode, error) {
	resp, body, err := r.Request("GET", "api/v5/nodes", nil)
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
