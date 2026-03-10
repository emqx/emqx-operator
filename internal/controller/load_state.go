package controller

import (
	"context"

	crdv2 "github.com/emqx/emqx-operator/api/v2"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

type reconcileState struct {
	coreSets      []*appsv1.StatefulSet
	replicantSets []*appsv1.ReplicaSet
	pods          []*corev1.Pod
}

func (r *reconcileState) podWithName(name string) *corev1.Pod {
	for _, pod := range r.pods {
		if pod.Name == name {
			return pod
		}
	}
	return nil
}

func (r *reconcileState) podsWithRole(role string) []*corev1.Pod {
	var list []*corev1.Pod
	for _, pod := range r.pods {
		if pod.Labels[crdv2.LabelDBRole] == role {
			list = append(list, pod)
		}
	}
	return list
}

func (r *reconcileState) podsManagedBy(object metav1.Object) []*corev1.Pod {
	var list []*corev1.Pod
	if object == nil {
		return list
	}
	for _, pod := range r.pods {
		if util.IsPodManagedBy(pod, object) {
			list = append(list, pod)
		}
	}
	return list
}

// coreSet returns the single core StatefulSet, or nil if none exists.
// With the rolling update model, there is always at most one core StatefulSet.
func (r *reconcileState) coreSet() *appsv1.StatefulSet {
	if len(r.coreSets) > 0 {
		return r.coreSets[0]
	}
	return nil
}

func (r *reconcileState) partOfCoreSet(pod *corev1.Pod) bool {
	sts := r.coreSet()
	if sts == nil {
		return false
	}
	return util.IsPodManagedBy(pod, sts)
}

// partOfCoreSetLatestRevision checks if a core pod's revision matches the StatefulSet's updateRevision.
func (r *reconcileState) partOfCoreSetLatestRevision(pod *corev1.Pod) bool {
	sts := r.coreSet()
	if sts == nil {
		return false
	}
	if !util.IsPodManagedBy(pod, sts) {
		return false
	}
	podRevision := pod.Labels[appsv1.ControllerRevisionHashLabelKey]
	return podRevision == sts.Status.UpdateRevision
}

// Returns ReplicaSet representing current set of replicant nodes.
// Current set is considered outdated if CurrentRevision != UpdateRevision.
func (r *reconcileState) currentReplicantSet(instance *crdv2.EMQX) *appsv1.ReplicaSet {
	for _, rs := range r.replicantSets {
		hash := rs.Labels[crdv2.LabelPodTemplateHash]
		if hash == instance.Status.ReplicantNodesStatus.CurrentRevision {
			return rs
		}
	}
	return nil
}

// Returns ReplicaSet representing newest set of replicant nodes.
// Same as current if CurrentRevision == UpdateRevision.
func (r *reconcileState) updateReplicantSet(instance *crdv2.EMQX) *appsv1.ReplicaSet {
	for _, rs := range r.replicantSets {
		hash := rs.Labels[crdv2.LabelPodTemplateHash]
		if hash == instance.Status.ReplicantNodesStatus.UpdateRevision {
			return rs
		}
	}
	return nil
}

// partOfUpdateReplicantSet checks if a pod belongs to the update (newest) ReplicaSet.
func (r *reconcileState) partOfUpdateReplicantSet(pod *corev1.Pod, instance *crdv2.EMQX) bool {
	controllerRef := metav1.GetControllerOf(pod)
	if controllerRef == nil {
		return false
	}
	updateReplicantSet := r.updateReplicantSet(instance)
	if updateReplicantSet != nil && controllerRef.UID == updateReplicantSet.UID {
		return true
	}
	return false
}

// partOfCurrentReplicantSet checks if a pod belongs to the current (outdated) ReplicaSet.
func (r *reconcileState) partOfCurrentReplicantSet(pod *corev1.Pod, instance *crdv2.EMQX) bool {
	controllerRef := metav1.GetControllerOf(pod)
	if controllerRef == nil {
		return false
	}
	currentReplicantSet := r.currentReplicantSet(instance)
	if currentReplicantSet != nil && controllerRef.UID == currentReplicantSet.UID {
		return true
	}
	return false
}

type loadState struct {
	*EMQXReconciler
}

func (l *loadState) reconcile(r *reconcileRound, instance *crdv2.EMQX) subResult {
	state := loadReconcileState(r.ctx, l.Client, instance)
	r.state = state
	return subResult{}
}

func loadReconcileState(ctx context.Context, client k8s.Client, instance *crdv2.EMQX) *reconcileState {
	state := &reconcileState{}

	stsList := &appsv1.StatefulSetList{}
	_ = client.List(ctx, stsList,
		k8s.InNamespace(instance.Namespace),
		k8s.MatchingLabels(instance.DefaultLabelsWith(crdv2.CoreLabels())),
	)

	for _, sts := range stsList.Items {
		state.coreSets = append(state.coreSets, sts.DeepCopy())
	}

	sortByCreationTimestamp(state.coreSets)

	rsList := &appsv1.ReplicaSetList{}
	_ = client.List(ctx, rsList,
		k8s.InNamespace(instance.Namespace),
		k8s.MatchingLabels(instance.DefaultLabelsWith(crdv2.ReplicantLabels())),
	)

	for _, rs := range rsList.Items {
		state.replicantSets = append(state.replicantSets, rs.DeepCopy())
	}

	sortByCreationTimestamp(state.replicantSets)

	podList := &corev1.PodList{}
	_ = client.List(ctx, podList,
		k8s.InNamespace(instance.Namespace),
		k8s.MatchingLabels(instance.DefaultLabels()),
	)

	for _, pod := range podList.Items {
		// Disregard pods that are being deleted.
		if pod.GetDeletionTimestamp() != nil {
			continue
		}

		// Disregard pods that are not controlled by any controller.
		controllerRef := metav1.GetControllerOf(&pod)
		if controllerRef == nil {
			continue
		}

		// Add the pod to the list of pods.
		pod := pod.DeepCopy()
		state.pods = append(state.pods, pod)
	}

	return state
}
