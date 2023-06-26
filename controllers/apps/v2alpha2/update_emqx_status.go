package v2alpha2

import (
	"context"
	"encoding/json"

	emperror "emperror.dev/errors"
	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type updateStatus struct {
	*EMQXReconciler
}

func (u *updateStatus) reconcile(ctx context.Context, instance *appsv2alpha2.EMQX, r innerReq.RequesterInterface) subResult {
	var err error
	var emqxNodes []appsv2alpha2.EMQXNode
	var existedSts *appsv1.StatefulSet = &appsv1.StatefulSet{}
	var existedRs *appsv1.ReplicaSet = &appsv1.ReplicaSet{}

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

	stsList := &appsv1.StatefulSetList{}
	_ = u.Client.List(ctx, stsList,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(appsv2alpha2.CloneAndAddLabel(
			instance.Spec.CoreTemplate.Labels,
			appsv1.DefaultDeploymentUniqueLabelKey,
			instance.Status.CoreNodesStatus.CurrentVersion,
		)),
	)
	if len(stsList.Items) > 0 {
		existedSts = stsList.Items[0].DeepCopy()
	}

	if isExistReplicant(instance) {
		if instance.Status.ReplicantNodesStatus == nil {
			instance.Status.ReplicantNodesStatus = &appsv2alpha2.EMQXNodesStatus{}
			instance.Status.ReplicantNodesStatus.Replicas = *instance.Spec.ReplicantTemplate.Spec.Replicas
		}

		rsList := &appsv1.ReplicaSetList{}
		_ = u.Client.List(ctx, rsList,
			client.InNamespace(instance.Namespace),
			client.MatchingLabels(appsv2alpha2.CloneAndAddLabel(
				instance.Spec.ReplicantTemplate.Labels,
				appsv1.DefaultDeploymentUniqueLabelKey,
				instance.Status.ReplicantNodesStatus.CurrentVersion,
			)),
		)
		if len(rsList.Items) > 0 {
			existedRs = rsList.Items[0].DeepCopy()
		}
	}

	instance.Status.CoreNodesStatus.Replicas = *instance.Spec.CoreTemplate.Spec.Replicas
	instance.Status.SetNodes(emqxNodes)
	newEMQXStatusMachine(instance).NextStatus(existedSts, existedRs)

	if err := u.Client.Status().Update(ctx, instance); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to update status")}
	}
	return subResult{}
}

func getNodeStatuesByAPI(r innerReq.RequesterInterface) ([]appsv2alpha2.EMQXNode, error) {
	resp, body, err := r.Request("GET", "api/v5/nodes", nil)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get API api/v5/nodes")
	}
	if resp.StatusCode != 200 {
		return nil, emperror.Errorf("failed to get API %s, status : %s, body: %s", "api/v5/nodes", resp.Status, body)
	}

	nodeStatuses := []appsv2alpha2.EMQXNode{}
	if err := json.Unmarshal(body, &nodeStatuses); err != nil {
		return nil, emperror.Wrap(err, "failed to unmarshal node statuses")
	}
	return nodeStatuses, nil
}
