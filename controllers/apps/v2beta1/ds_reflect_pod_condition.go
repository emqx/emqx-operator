package v2beta1

import (
	"context"
	"strings"

	emperror "emperror.dev/errors"
	appsv2beta1 "github.com/emqx/emqx-operator/apis/apps/v2beta1"
	ds "github.com/emqx/emqx-operator/controllers/apps/v2beta1/ds"
	req "github.com/emqx/emqx-operator/internal/requester"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type dsReflectPodCondition struct {
	*EMQXReconciler
}

func (u *dsReflectPodCondition) reconcile(
	ctx context.Context,
	logger logr.Logger,
	instance *appsv2beta1.EMQX,
	r req.RequesterInterface,
) subResult {
	if r == nil {
		return subResult{}
	}

	r = u.getSuitableRequester(ctx, instance, r)

	// If EMQX DS API is not available, skip this reconciliation step.
	// We need this API to be available to ask it about replication status.
	cluster, err := ds.GetCluster(r)
	if err != nil && emperror.Is(err, ds.APIErrorUnavailable) {
		return subResult{}
	}
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to fetch DS cluster status")}
	}

	list := &corev1.PodList{}
	err = u.Client.List(
		ctx,
		list,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(appsv2beta1.DefaultLabels(instance)),
	)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to list pods")}
	}

	for _, p := range list.Items {
		pod := p.DeepCopy()
		node := u.findNode(instance, pod)
		if node == nil || node.Edition != "Enterprise" {
			continue
		}
		condition := corev1.PodCondition{
			Type:               appsv2beta1.DSReplicationSite,
			Status:             corev1.ConditionUnknown,
			LastTransitionTime: metav1.Now(),
		}
		site := cluster.FindSite(node.Node)
		if site != nil {
			if len(site.Shards) > 0 {
				condition.Status = corev1.ConditionTrue
			} else {
				condition.Status = corev1.ConditionFalse
			}
		}
		existing := appsv2beta1.FindPodCondition(pod, appsv2beta1.DSReplicationSite)
		if existing == nil || existing.Status != condition.Status {
			err := updatePodCondition(ctx, u.Client, pod, condition)
			if err != nil {
				return subResult{err: emperror.Wrapf(err, "failed to update pod %s status", pod.Name)}
			}
		}
	}

	return subResult{}
}

func (u *dsReflectPodCondition) findNode(instance *appsv2beta1.EMQX, pod *corev1.Pod) *appsv2beta1.EMQXNode {
	for _, node := range instance.Status.CoreNodes {
		if node.PodName == pod.Name {
			return &node
		}
	}
	for _, node := range instance.Status.ReplicantNodes {
		if node.PodName == pod.Name {
			return &node
		}
	}
	return nil
}

func (u *dsReflectPodCondition) getPod(
	ctx context.Context,
	instance *appsv2beta1.EMQX,
	podName string,
) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	key := types.NamespacedName{Namespace: instance.Namespace, Name: podName}
	err := u.Client.Get(ctx, key, pod)
	return pod.DeepCopy(), err
}

func (u *dsReflectPodCondition) getSuitableRequester(
	ctx context.Context,
	instance *appsv2beta1.EMQX,
	r req.RequesterInterface,
) req.RequesterInterface {
	// Prefer Enterprise node which is part of "update" StatefulSet (if any).
	for _, core := range instance.Status.CoreNodes {
		if core.Edition == "Enterprise" && strings.Contains(core.PodName, instance.Status.CoreNodesStatus.UpdateRevision) {
			pod, err := u.getPod(ctx, instance, core.PodName)
			if err == nil && pod.DeletionTimestamp == nil && pod.Status.PodIP != "" {
				return r.SwitchHost(pod.Status.PodIP)
			}
		}
	}
	// If no suitable pod found, return the original requester.
	return r
}
