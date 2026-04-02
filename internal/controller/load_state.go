package controller

import (
	"context"

	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v3alpha1"
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
		if pod.Labels[crd.LabelDBRole] == role {
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
	coreSet := r.coreSet()
	if coreSet == nil {
		return false
	}
	return util.IsPodManagedBy(pod, coreSet)
}

// partOfCoreSetRevision checks if a core pod's StatefulSet-assigned revision matches the specified revision.
func (r *reconcileState) partOfCoreSetRevision(pod *corev1.Pod, revision string) bool {
	coreSet := r.coreSet()
	if coreSet == nil {
		return false
	}
	if !util.IsPodManagedBy(pod, coreSet) {
		return false
	}
	podRevision := pod.Labels[appsv1.ControllerRevisionHashLabelKey]
	return podRevision == revision
}

// numCoresRevision counts number of core pods running specified StatefulSet revision.
func (r *reconcileState) numCoresRevision(revision string) int {
	coreSet := r.coreSet()
	if coreSet == nil {
		return 0
	}
	num := 0
	for _, pod := range r.podsManagedBy(coreSet) {
		podRevision := pod.Labels[appsv1.ControllerRevisionHashLabelKey]
		if podRevision == revision {
			num++
		}
	}
	return num
}

// Returns ReplicaSet representing current set of replicant nodes.
// Current set is considered outdated if CurrentRevision != UpdateRevision.
func (r *reconcileState) currentReplicantSet(instance *crd.EMQX) *appsv1.ReplicaSet {
	for _, rs := range r.replicantSets {
		hash := rs.Labels[crd.LabelPodTemplateHash]
		if hash == instance.Status.ReplicantNodesStatus.CurrentRevision {
			return rs
		}
	}
	return nil
}

// Returns ReplicaSet representing newest set of replicant nodes.
// Same as current if CurrentRevision == UpdateRevision.
func (r *reconcileState) updateReplicantSet(instance *crd.EMQX) *appsv1.ReplicaSet {
	for _, rs := range r.replicantSets {
		hash := rs.Labels[crd.LabelPodTemplateHash]
		if hash == instance.Status.ReplicantNodesStatus.UpdateRevision {
			return rs
		}
	}
	return nil
}

// partOfUpdateReplicantSet checks if a pod belongs to the update (newest) ReplicaSet.
func (r *reconcileState) partOfUpdateReplicantSet(pod *corev1.Pod, instance *crd.EMQX) bool {
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

func (r *reconcileState) areCoresAvailable(instance *crd.EMQX) bool {
	coreSet := r.coreSet()
	if coreSet == nil {
		return false
	}
	return coreSet.Status.AvailableReplicas >= instance.Spec.NumCoreReplicas()
}

func (r *reconcileState) areReplicantsAvailable(instance *crd.EMQX) bool {
	desired := instance.Spec.NumReplicantReplicas()
	if desired == 0 {
		return true
	}
	replicantSet := r.updateReplicantSet(instance)
	if replicantSet == nil {
		return false
	}
	return replicantSet.Status.AvailableReplicas >= desired
}

type loadState struct {
	*EMQXReconciler
}

func (l *loadState) reconcile(r *reconcileRound, instance *crd.EMQX) subResult {
	state, err := loadReconcileState(r.ctx, l.Client, instance)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to load reconcile round state")}
	}
	r.state = state
	return subResult{}
}

func reloadReconcileState(r *reconcileRound, client k8s.Client, instance *crd.EMQX) error {
	state, err := loadReconcileState(r.ctx, client, instance)
	if err != nil {
		return err
	}
	r.state = state
	return nil
}

func loadReconcileState(
	ctx context.Context,
	client k8s.Client,
	instance *crd.EMQX,
) (*reconcileState, error) {
	var err error
	state := &reconcileState{}

	stsList := &appsv1.StatefulSetList{}
	err = client.List(ctx, stsList,
		k8s.InNamespace(instance.Namespace),
		k8s.MatchingLabels(instance.DefaultLabelsWith(crd.CoreLabels())),
	)
	if err != nil {
		return nil, err
	}

	for _, sts := range stsList.Items {
		state.coreSets = append(state.coreSets, sts.DeepCopy())
	}

	sortByCreationTimestamp(state.coreSets)

	rsList := &appsv1.ReplicaSetList{}
	err = client.List(ctx, rsList,
		k8s.InNamespace(instance.Namespace),
		k8s.MatchingLabels(instance.DefaultLabelsWith(crd.ReplicantLabels())),
	)
	if err != nil {
		return nil, err
	}

	for _, rs := range rsList.Items {
		state.replicantSets = append(state.replicantSets, rs.DeepCopy())
	}

	sortByCreationTimestamp(state.replicantSets)

	podList := &corev1.PodList{}
	err = client.List(ctx, podList,
		k8s.InNamespace(instance.Namespace),
		k8s.MatchingLabels(instance.DefaultLabels()),
	)
	if err != nil {
		return nil, err
	}

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

	return state, nil
}
